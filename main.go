package main

import (
	"api/api"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const (
	defaultPort = "8080"
	portEnv     = "PORT"
)

func main() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	templates := template.Must(template.ParseFiles(
		filepath.Join(dir, "templates", "pages", "home.html"),
		filepath.Join(dir, "templates", "base", "header.html"),
		filepath.Join(dir, "templates", "base", "footer.html"),
	))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Title string
		}{
			Title: "Home - MetaRock",
		}
		err := templates.ExecuteTemplate(w, "home", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
		}
	})
	fmt.Println("Server running at http://localhost:" + serverPort())
	connection := http.ListenAndServe(":8080", routes())
	if connection != nil {
		log.Fatalf("démarrage du serveur: %v", connection)
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
	mux.HandleFunc("/api", api.ApiHandler)
	mux.HandleFunc("/artists", api.DataHandler)
	mux.HandleFunc("/locations", api.DataHandler)
	mux.HandleFunc("/dates", api.DataHandler)
	mux.HandleFunc("/relation", api.DataHandler)
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
