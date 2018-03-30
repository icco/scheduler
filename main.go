package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron"
	"gopkg.in/unrolled/render.v1"
	"gopkg.in/unrolled/secure.v1"
)

var (
	// Renderer is a renderer for all occasions. These are our preferred default options.
	Renderer = render.New(render.Options{
		Directory:                 "views",
		Extensions:                []string{".tmpl", ".html"}, // Specify extensions to load for templates.
		Charset:                   "UTF-8",                    // Sets encoding for content-types. Default is "UTF-8".
		IndentJSON:                false,                      // Don't output human readable JSON.
		IndentXML:                 true,                       // Output human readable XML.
		RequirePartials:           true,                       // Return an error if a template is missing a partial used in a layout.
		DisableHTTPErrorRendering: false,                      // Enables automatic rendering of http.StatusInternalServerError when an error occurs.
	})

	// SecureMiddlewareOptions is a set of defaults for securing web apps.
	// SSLRedirect is handeled by a different middleware because it does not
	// support whitelisting paths.
	SecureMiddlewareOptions = secure.Options{
		HostsProxyHeaders:    []string{"X-Forwarded-Host"},
		SSLRedirect:          false, // No way to whitelist for healthcheck :/
		SSLProxyHeaders:      map[string]string{"X-Forwarded-Proto": "https"},
		STSSeconds:           315360000,
		STSIncludeSubdomains: true,
		STSPreload:           true,
		FrameDeny:            true,
		ContentTypeNosniff:   true,
		BrowserXssFilter:     true,
		IsDevelopment:        os.Getenv("FLM_ENV") == "local",
	}
)

func main() {
	port := "8080"
	if fromEnv := os.Getenv("PORT"); fromEnv != "" {
		port = fromEnv
	}
	log.Printf("Starting up on %s", port)

	secureMiddleware := secure.New(SecureMiddlewareOptions)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(secureMiddleware.Handler)
	r.Use(SSLMiddleware)

	// Metrics
	r.Get("/_healthcheck.json", healthCheckHandler)
	r.Mount("/metrics", promhttp.Handler())

	// Web app
	r.Get("/", homeHandler)
	r.Get("/cron", cronHandler)

	log.Printf("Server listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	Renderer.JSON(w, http.StatusOK, map[string]string{
		"healthy":  "true",
		"revision": os.Getenv("GIT_REVISION"),
		"tag":      os.Getenv("GIT_TAG"),
		"branch":   os.Getenv("GIT_BRANCH"),
	})
}

// SSLMiddleware redirects for all paths besides /_healthcheck.json. This is a
// slight modification of the code in
// https://github.com/unrolled/secure/blob/v1/secure.go
func SSLMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_healthcheck.json" {
			ssl := strings.EqualFold(r.URL.Scheme, "https") || r.TLS != nil
			if !ssl {
				for k, v := range SecureMiddlewareOptions.SSLProxyHeaders {
					if r.Header.Get(k) == v {
						ssl = true
						break
					}
				}
			}

			if !ssl && !SecureMiddlewareOptions.IsDevelopment {
				url := r.URL
				url.Scheme = "https"
				url.Host = r.Host

				http.Redirect(w, r, url.String(), http.StatusMovedPermanently)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	cronFile, err := GetConfig()
	if err != nil {
		log.Printf("Error getting config: %+v", err)
		http.Error(w, "Bad config file", http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(cronFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func cronHandler(w http.ResponseWriter, r *http.Request) {
	cf, err := GetConfig()
	if err != nil {
		log.Printf("Error getting config: %+v", err)
		http.Error(w, "Bad config file", http.StatusInternalServerError)
		return
	}

	for _, j := range cf.Jobs {
		n, err := j.Next(time.Now())
		if err != nil {
			log.Printf("Error getting next run: %+v", err)
			continue
		}
		log.Printf("%+v - %+v", j, n)
	}

	w.Write([]byte(`"ok."`))
}

type ConfigFile struct {
	Jobs []Job `json:"jobs"`
}

type Job struct {
	Name     string `json:"name"`
	CronRule string `json:"cron"`
}

func (j *Job) Next(t time.Time) (time.Time, error) {
	sched, err := cron.Parse(j.CronRule)
	if err != nil {
		return time.Now(), err
	}

	return sched.Next(t), nil
}

// Run takes the docker image and the command, builds a task definition,
// submits it to ECS, and runs the task.
func (j *Job) Run() {

}

func GetConfig() (ConfigFile, error) {
	filename := os.Getenv("SCHEDULER_CONFIG")
	if filename == "" {
		filename = "config.example.json"
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return ConfigFile{}, err
	}

	var cf ConfigFile
	err = json.Unmarshal(data, &cf)

	return cf, nil
}
