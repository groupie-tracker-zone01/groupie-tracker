package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
)

func Routes(templates *template.Template, data *AppData, fullArtists []ArtistFull) http.Handler {
	mux := http.NewServeMux()
	// Static //
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	// Home page data //
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			renderErrors(w, http.StatusNotFound, templates)
			return
		}
		if r.Method != http.MethodGet {
			renderErrors(w, http.StatusMethodNotAllowed, templates)
			return
		}
		homeData := struct {
			Title   string
			Artists []ArtistFull
			Query   string
		}{
			Title:   "Home - MetaRock",
			Artists: []ArtistFull{}, // empty list
			Query:   "",
		}
		renderPage(w, templates, "home", homeData)
	})
	// Artists Page //
	mux.HandleFunc("/artists", func(w http.ResponseWriter, r *http.Request) {
		searchArtists(w, r, templates, fullArtists)
	})
	// API Routes
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

// Searches artists whose name contains the expression passed in the http request and displays them on the page if they are found.
func searchArtists(w http.ResponseWriter, r *http.Request, templates *template.Template, fullArtists []ArtistFull) {
	if r.Method != http.MethodGet {
		renderErrors(w, http.StatusMethodNotAllowed, templates)
		return
	}
	query := r.URL.Query().Get("q")
	query = strings.TrimSpace(query)
	if query == "" {
		renderErrors(w, http.StatusBadRequest, templates)
		return
	}
	artistsToShow := fullArtists
	if query != "" {
		queryLower := strings.ToLower(query)
		var filtered []ArtistFull
		for _, artist := range fullArtists {
			nameLower := strings.ToLower(artist.Name)
			if nameLower == queryLower {
				filtered = []ArtistFull{artist}
				break
			}
			if strings.Contains(strings.ToLower(artist.Name), queryLower) {
				filtered = append(filtered, artist)
				continue
			}
			if len(filtered) >= 20 {
				break
			}
		}
		artistsToShow = filtered
	}
	artistData := struct {
		Title   string
		Artists []ArtistFull
		Query   string
	}{
		Title:   "Artists - MetaRock",
		Artists: artistsToShow,
		Query:   query,
	}
	renderPage(w, templates, "artists", artistData)
}

func handleSearch(w http.ResponseWriter, r *http.Request, fullArtists []ArtistFull, templates *template.Template) {
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
		if len(results) >= 10 {
			break
		}
	}
	fmt.Println(results)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func renderPage(w http.ResponseWriter, templates *template.Template, templateName string, data any) {
	var buffer bytes.Buffer
	if err := templates.ExecuteTemplate(&buffer, templateName, data); err != nil {
		log.Println(err)
		renderErrors(w, http.StatusInternalServerError, templates)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(buffer.Bytes()); err != nil {
		log.Println(err)
	}
}

func renderErrors(w http.ResponseWriter, status int, templates *template.Template) {
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

	var buffer bytes.Buffer
	if err := templates.ExecuteTemplate(&buffer, templateName, nil); err != nil {
		http.Error(w, http.StatusText(status), status)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, err := w.Write(buffer.Bytes()); err != nil {
		log.Println(err)
	}
}
