package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/robfig/cron"
)

func main() {
	port := "8080"
	if fromEnv := os.Getenv("PORT"); fromEnv != "" {
		port = fromEnv
	}

	server := http.NewServeMux()
	server.HandleFunc("/", homeHandler)
	server.HandleFunc("/cron", cronHandler)
	server.HandleFunc("/_healthcheck.json", healthCheckHandler)

	loggedRouter := handlers.LoggingHandler(os.Stdout, server)

	log.Printf("Server listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, loggedRouter))
}

type HealthRespJson struct {
	Healthy string `json:"healthy"`
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthRespJson{
		Healthy: "true",
	}

	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
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
