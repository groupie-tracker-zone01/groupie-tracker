package main

import (
	"api/api"
	"encoding/json"
	"fmt"
	"html/template"
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

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	mux.HandleFunc("/", homeHandler(data))
	mux.HandleFunc("/artists", api.ArtistsHandler(data))
	mux.HandleFunc("/locations", api.LocationsHandler(data))
	mux.HandleFunc("/dates", api.DatesHandler(data))
	mux.HandleFunc("/relations", api.RelationsHandler(data))

	return mux
}

func homeHandler(data *api.AppData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		tmpl, err := template.ParseFiles(
			"templates/pages/home.html",
			"templates/base/header.html",
			"templates/base/footer.html",
		)
		if err != nil {
			http.Error(w, "Unable to load page", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if err := tmpl.ExecuteTemplate(w, "home", data); err != nil {
			log.Println(err)
		}
	}
}
