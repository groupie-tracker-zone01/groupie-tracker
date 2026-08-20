package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

const (
	defaultPort = "8080"
	portEnv     = "PORT"
)

func main() {
	address := ":" + serverPort()

	log.Printf("serveur disponible sur http://localhost%s", address)
	if err := http.ListenAndServe(address, routes()); err != nil {
		log.Fatalf("démarrage du serveur: %v", err)
	}
}

func serverPort() string {
	if port := os.Getenv(portEnv); port != "" {
		return port
	}

	return defaultPort
}

func routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler)

	return mux
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "<h1>Groupie Tracker</h1><p>Le serveur fonctionne.</p>")
}
