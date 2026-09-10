package main

import (
	"groupie-tracker/server"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func applicationTemplates(t *testing.T) *template.Template {
	t.Helper()

	templates, err := template.ParseFiles(
		"templates/pages/home.html",
		"templates/pages/artists.html",
		"templates/base/header.html",
		"templates/base/footer.html",
		"templates/pages/errors/400.html",
		"templates/pages/errors/403.html",
		"templates/pages/errors/404.html",
		"templates/pages/errors/405.html",
		"templates/pages/errors/500.html",
	)
	if err != nil {
		t.Fatalf("cannot parse application templates: %v", err)
	}
	return templates
}

func applicationTestData() (*server.AppData, []server.ArtistFull) {
	data := &server.AppData{
		Artists: []server.Artist{
			{Id: 1, Name: "Queen", Image: "https://example.com/queen.jpg", Members: []string{"Freddie Mercury"}, CreationDate: 1970, FirstAlbum: "14-12-1973"},
			{Id: 2, Name: "Queens of the Stone Age", Image: "https://example.com/qotsa.jpg", Members: []string{"Josh Homme"}, CreationDate: 1996, FirstAlbum: "22-09-1998"},
		},
		Locations: server.LocationWrapper{LocWrapper: []server.Location{
			{Id: 1, Locations: []string{"london-uk"}},
			{Id: 2, Locations: []string{"los_angeles-usa"}},
		}},
		Dates: server.DateWrapper{DatWrapper: []server.Date{
			{Id: 1, Dates: []string{"01-01-2020"}},
			{Id: 2, Dates: []string{"02-02-2020"}},
		}},
		Relations: server.RelationWrapper{RelWrapper: []server.Relation{
			{Id: 1, DatesLocations: map[string][]string{"london-uk": {"01-01-2020"}}},
			{Id: 2, DatesLocations: map[string][]string{"los_angeles-usa": {"02-02-2020"}}},
		}},
	}

	return data, server.GetFullArtists(data)
}

func TestApplicationUsesRealTemplates(t *testing.T) {
	data, fullArtists := applicationTestData()
	handler := server.Routes(applicationTemplates(t), data, fullArtists)

	t.Run("home renders the real page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("GET / returned %d, want %d", res.Code, http.StatusOK)
		}
		body := res.Body.String()
		for _, expected := range []string{"/artists", "/static/css/pages/home.css", "/static/JS/home.js"} {
			if !strings.Contains(body, expected) {
				t.Fatalf("home page does not contain %q", expected)
			}
		}
	})

	t.Run("partial search reaches the real artists template", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/artists?q=Que", nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("GET /artists returned %d, want %d", res.Code, http.StatusOK)
		}
		body := res.Body.String()
		for _, expected := range []string{"Queen", "Queens of the Stone Age", "/static/JS/artists.js"} {
			if !strings.Contains(body, expected) {
				t.Fatalf("artists page does not contain %q", expected)
			}
		}
	})

	t.Run("unknown route renders the real 404 page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		if res.Code != http.StatusNotFound {
			t.Fatalf("unknown route returned %d, want %d", res.Code, http.StatusNotFound)
		}
		if res.Body.Len() == 0 {
			t.Fatal("404 page is empty")
		}
	})
}

func TestApplicationServesStaticAssets(t *testing.T) {
	data, fullArtists := applicationTestData()
	handler := server.Routes(applicationTemplates(t), data, fullArtists)

	for _, path := range []string{
		"/static/css/pages/home.css",
		"/static/JS/home.js",
		"/static/JS/artists.js",
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)

			if res.Code != http.StatusOK {
				t.Fatalf("GET %s returned %d, want %d", path, res.Code, http.StatusOK)
			}
			if res.Body.Len() == 0 {
				t.Fatalf("GET %s returned an empty body", path)
			}
		})
	}
}
