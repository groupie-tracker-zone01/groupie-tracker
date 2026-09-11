package server

import (
	"bytes"
	"encoding/json"
	"strconv"
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
			Title       string
			Artists     []ArtistFull
			Suggestions []ArtistFull
			Query       string
		}{
			Title:       "Home - MetaRock",
			Artists:     []ArtistFull{},
			Suggestions: fullArtists,
			Query:       "",
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

// Searches artists across the fields required by the search-bar exercise.
func searchArtists(w http.ResponseWriter, r *http.Request, templates *template.Template, fullArtists []ArtistFull) {
	if r.Method != http.MethodGet {
		renderErrors(w, http.StatusMethodNotAllowed, templates)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		renderErrors(w, http.StatusBadRequest, templates)
		return
	}

	artistData := struct {
		Title   string
		Artists []ArtistFull
		Query   string
	}{
		Title:   "Artists - MetaRock",
		Artists: filterArtists(fullArtists, query),
		Query:   query,
	}
	renderPage(w, templates, "artists", artistData)
}

func handleSearch(w http.ResponseWriter, r *http.Request, fullArtists []ArtistFull, templates *template.Template) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		renderErrors(w, http.StatusBadRequest, templates)
		return
	}

	var results []string
	for _, artist := range filterArtists(fullArtists, query) {
		results = append(results, artist.Name)
		if len(results) >= 10 {
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func filterArtists(fullArtists []ArtistFull, query string) []ArtistFull {
	var filtered []ArtistFull
	for _, artist := range fullArtists {
		if artistMatchesQuery(artist, query) {
			filtered = append(filtered, artist)
		}
	}
	return filtered
}

func artistMatchesQuery(artist ArtistFull, query string) bool {
	query = normalizeSearchText(query)
	if query == "" {
		return false
	}

	if strings.Contains(normalizeSearchText(artist.Name), query) {
		return true
	}
	if strings.Contains(strconv.Itoa(artist.CreationDate), query) {
		return true
	}
	if strings.Contains(normalizeSearchText(artist.FirstAlbum), query) {
		return true
	}
	for _, member := range artist.Members {
		if strings.Contains(normalizeSearchText(member), query) {
			return true
		}
	}
	for _, location := range artist.Locations {
		if strings.Contains(normalizeSearchText(location), query) {
			return true
		}
	}
	return false
}

func normalizeSearchText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer("_", " ", "-", " ").Replace(value)
	return strings.Join(strings.Fields(value), " ")
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
