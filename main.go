package main

import (
	"encoding/json"
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
	server.Handle("/", home)
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
		Healthy: true,
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
}
