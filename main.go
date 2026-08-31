package main

import (
	"api/api"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

const (
	defaultPort = "8080"
	portEnv     = "PORT"
)

const baseURL = "https://groupietrackers.herokuapp.com/api"

func fetchJSON(url string, target any) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API: statut inattendu %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func main() {
	data, err := api.LoadData()
	if err != nil {
		log.Fatal(err)
	}
	if err := http.ListenAndServe(":8080", routes(data)); err != nil {
		log.Fatal(err)
	}
}

func serverPort() string {
	if port := os.Getenv(portEnv); port != "" {
		return port
	}
	return defaultPort
}

func routes(data *api.AppData) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/artists", api.ArtistsHandler(data))
	mux.HandleFunc("/locations", api.LocationsHandler(data))
	mux.HandleFunc("/dates", api.DatesHandler(data))
	mux.HandleFunc("/relations", api.RelationsHandler(data))
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
