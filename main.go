package main

import (
	"encoding/json"
	"fmt"
	"groupie-tracker/api"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	data, err := api.LoadData()
	if err != nil {
		log.Fatal(err)
	}
	// API full data artists
	fullArtists := api.GetFullArtists(data)

	templates := template.Must(template.ParseFiles(
		filepath.Join(dir, "templates", "pages", "home.html"),
		filepath.Join(dir, "templates", "pages", "artists.html"),
		filepath.Join(dir, "templates", "base", "header.html"),
		filepath.Join(dir, "templates", "base", "footer.html"),
		filepath.Join(dir, "templates", "pages", "errors", "400.html"),
		filepath.Join(dir, "templates", "pages", "errors", "403.html"),
		filepath.Join(dir, "templates", "pages", "errors", "404.html"),
		filepath.Join(dir, "templates", "pages", "errors", "405.html"),
		filepath.Join(dir, "templates", "pages", "errors", "500.html"),
	))
	router := routes(templates, data, fullArtists)
	port := serverPort()
	fmt.Printf("Server running at http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

// Start the server
func serverPort() string {
	if port := os.Getenv(portEnv); port != "" {
		return port
	}

	return defaultPort
}

func routes(templates *template.Template, data *api.AppData, fullArtists []api.ArtistFull) http.Handler {
	mux := http.NewServeMux()
	//  --- Static --- //
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// --- Home page data --- //
		homeData := struct {
			Title   string
			Artists []api.ArtistFull
			Query   string
		}{
			Title:   "Home - MetaRock",
			Artists: fullArtists,
			Query:   "",
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		err := templates.ExecuteTemplate(w, "home", homeData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
		}
	})

	// --- Artists Page --- //
	mux.HandleFunc("/artists", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		query = strings.TrimSpace(query)

		if query == "" {
			w.WriteHeader(http.StatusBadRequest)
			templates.ExecuteTemplate(w, "400", nil)
			return
		}

		artistsToShow := fullArtists

		if query != "" {
			queryLower := strings.ToLower(query)
			var filtered []api.ArtistFull

			for _, artist := range fullArtists {
				nameLower := strings.ToLower(artist.Name)
				if nameLower == queryLower {
					filtered = []api.ArtistFull{artist}
					break
				}
				if strings.Contains(strings.ToLower(artist.Name), queryLower) {
					filtered = append(filtered, artist)
					continue
				}
				found := false
				for _, member := range artist.Members {
					if strings.Contains(strings.ToLower(member), queryLower) {
						filtered = append(filtered, artist)
						found = true
						break
					}
				}
				if found {
					continue
				}
				for _, date := range artist.Dates {
					if strings.Contains(strings.ToLower(date), queryLower) {
						filtered = append(filtered, artist)
						found = true
						break
					}
				}
				if found {
					continue
				}

				for _, location := range artist.Locations {
					if strings.Contains(strings.ToLower(location), queryLower) {
						filtered = append(filtered, artist)
						found = true
						break
					}
				}

				if len(filtered) >= 20 {
					break
				}
			}
			artistsToShow = filtered
		}

		artistData := struct {
			Title   string
			Artists []api.ArtistFull
			Query   string
		}{
			Title:   "Artists - MetaRock",
			Artists: artistsToShow,
			Query:   query,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		err := templates.ExecuteTemplate(w, "artists", artistData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
		}
	})

	// --- API Routes ---
	mux.HandleFunc("/api/artists", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fullArtists)
	})

	mux.HandleFunc("/api/locations", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data.Locations)
	})

	mux.HandleFunc("/api/dates", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data.Dates)
	})

	mux.HandleFunc("/api/relations", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data.Relations)
	})

	mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		handleSearch(w, r, fullArtists, templates)
	})

	return mux
}

// --- Search Handler ---
func handleSearch(w http.ResponseWriter, r *http.Request, fullArtists []api.ArtistFull, templates *template.Template) {
	query := r.URL.Query().Get("q")
	query = strings.ToLower(strings.TrimSpace(query))

	if query == "" {
		renderErrors(w, http.StatusBadRequest, templates)
		return
	}

	var results []string
	for _, artist := range fullArtists {
		if strings.Contains(strings.ToLower(artist.Name), query) {
			results = append(results, artist.Name)
			continue
		}

		found := false
		for _, member := range artist.Members {
			if strings.Contains(strings.ToLower(member), query) {
				results = append(results, artist.Name)
				found = true
				break
			}
		}
		if found {
			continue
		}

		for _, date := range artist.Dates {
			if strings.Contains(strings.ToLower(date), query) {
				results = append(results, artist.Name)
				found = true
				break
			}
		}
		if found {
			continue
		}

		for _, location := range artist.Locations {
			if strings.Contains(strings.ToLower(location), query) {
				results = append(results, artist.Name)
				break
			}
		}
		if len(results) >= 10 {
			break
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func renderErrors(w http.ResponseWriter, status int, templates *template.Template) {
	w.WriteHeader(status)
	var templateName string
	switch status {
	case http.StatusBadRequest:
		templateName = "400"
	case http.StatusForbidden:
		templateName = "403"
	case http.StatusNotFound:
		templateName = "404"
	case http.StatusMethodNotAllowed:
		templateName = "405"
	case http.StatusInternalServerError:
		templateName = "500"
	default:
		templateName = "home"
	}

	err := templates.ExecuteTemplate(w, templateName, nil)
	if err != nil {
		http.Error(w, http.StatusText(status), status)
		return
	}
}
