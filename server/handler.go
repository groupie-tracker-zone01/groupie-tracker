package server

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type paginationLink struct {
	Number  int
	URL     string
	Current bool
}

type artistsPageData struct {
	Title          string
	Query          string
	Suggestions    []ArtistFull
	Artists        []ArtistFull
	SelectedArtist *ArtistFull
	Limit          int
	TotalResults   int
	Pages          []paginationLink
	PrevURL        string
	NextURL        string
}

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
	mux.HandleFunc("/artist", func(w http.ResponseWriter, r *http.Request) {
		artistDetails(w, r, templates, fullArtists)
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

	filtered := filterArtists(fullArtists, query)
	limit := resultLimit(r.URL.Query().Get("limit"))
	page := positiveInt(r.URL.Query().Get("page"), 1)
	totalResults := len(filtered)
	totalPages := 0
	if totalResults > 0 {
		totalPages = (totalResults + limit - 1) / limit
		if page > totalPages {
			page = totalPages
		}
	}

	start := (page - 1) * limit
	end := start + limit
	if end > totalResults {
		end = totalResults
	}

	visibleArtists := []ArtistFull{}
	if start >= 0 && start < totalResults {
		visibleArtists = filtered[start:end]
	}

	data := artistsPageData{
		Title:        "Artists - MetaRock",
		Query:        query,
		Suggestions:  fullArtists,
		Artists:      visibleArtists,
		Limit:        limit,
		TotalResults: totalResults,
	}

	for pageNumber := 1; pageNumber <= totalPages; pageNumber++ {
		data.Pages = append(data.Pages, paginationLink{
			Number:  pageNumber,
			URL:     resultsURL(query, limit, pageNumber),
			Current: pageNumber == page,
		})
	}
	if page > 1 {
		data.PrevURL = resultsURL(query, limit, page-1)
	}
	if page < totalPages {
		data.NextURL = resultsURL(query, limit, page+1)
	}

	renderPage(w, templates, "artists", data)
}

func artistDetails(w http.ResponseWriter, r *http.Request, templates *template.Template, fullArtists []ArtistFull) {
	if r.Method != http.MethodGet {
		renderErrors(w, http.StatusMethodNotAllowed, templates)
		return
	}

	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id <= 0 {
		renderErrors(w, http.StatusBadRequest, templates)
		return
	}

	var selected *ArtistFull
	for i := range fullArtists {
		if fullArtists[i].Id == id {
			selected = &fullArtists[i]
			break
		}
	}
	if selected == nil {
		renderErrors(w, http.StatusNotFound, templates)
		return
	}

	data := artistsPageData{
		Title:          selected.Name + " - MetaRock",
		Suggestions:    fullArtists,
		SelectedArtist: selected,
		Limit:          5,
	}
	renderPage(w, templates, "artists", data)
}

func resultLimit(value string) int {
	switch value {
	case "10":
		return 10
	case "25":
		return 25
	default:
		return 5
	}
}

func positiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func resultsURL(query string, limit, page int) string {
	values := url.Values{}
	values.Set("q", query)
	values.Set("limit", strconv.Itoa(limit))
	values.Set("page", strconv.Itoa(page))
	return "/artists?" + values.Encode()
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
	value = strings.NewReplacer("_", " ", "-", " ", ",", " ").Replace(value)
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
