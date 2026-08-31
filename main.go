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
	// --- Actual directory
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// --- Templates config
	templates := template.Must(template.ParseFiles(
		filepath.Join(dir, "templates", "pages", "home.html"),
		filepath.Join(dir, "templates", "base", "header.html"),
		filepath.Join(dir, "templates", "base", "footer.html"),
	))
	router := routes(templates)

	// --- Start server
	port := serverPort()
	fmt.Printf("Server running at http://localhost:%s\n", port)

	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Erro ao iniciar o servidor: %v", err)
	}
}

func serverPort() string {
	if port := os.Getenv(portEnv); port != "" {
		return port
	}
	return defaultPort
}

func routes(templates *template.Template) http.Handler {
	mux := http.NewServeMux()

	// --- Static route mux
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// --- Route principal white the templates
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		data := struct {
			Title string
		}{
			Title: "Home - MetaRock",
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		err := templates.ExecuteTemplate(w, "home", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
		}
	})

	// --- Routes API
	mux.HandleFunc("/api", api.ApiHandler)
	mux.HandleFunc("/artists", api.DataHandler)
	mux.HandleFunc("/locations", api.DataHandler)
	mux.HandleFunc("/dates", api.DataHandler)
	mux.HandleFunc("/relation", api.DataHandler)
	return mux
}
