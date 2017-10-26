package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
)

func main() {
	port := "8080"
	if fromEnv := os.Getenv("PORT"); fromEnv != "" {
		port = fromEnv
	}

	server := http.NewServeMux()
	server.HandleFunc("/", home)
	server.HandleFunc("/_healthcheck.json", healthCheck)

	loggedRouter := handlers.LoggingHandler(os.Stdout, server)

	log.Printf("Server listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, loggedRouter))
}

type HealthRespJson struct {
	Healthy string `json:"healthy"`
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
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

func home(w http.ResponseWriter, r *http.Request) {
	cronFile, err := getConfig()
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

func cron(w http.ResponseWriter, r *http.Request) {

}

type ConfigFile struct {
	Jobs []Job `json:"jobs"`
}

type Job struct {
	Name     string `json:"name"`
	CronRule string `json:"cron"`
}

func getConfig() (ConfigFile, error) {
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
