package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testTemplates(t *testing.T) *template.Template {
	t.Helper()

	const source = `
{{define "home"}}HOME{{range .Artists}} {{.Name}} {{.Image}}{{end}}{{end}}
{{define "artists"}}ARTISTS{{range .Artists}} {{.Name}}{{end}}{{end}}
{{define "400"}}BAD REQUEST{{end}}
{{define "403"}}FORBIDDEN{{end}}
{{define "404"}}NOT FOUND{{end}}
{{define "405"}}METHOD NOT ALLOWED{{end}}
{{define "500"}}SERVER ERROR{{end}}
`

	tmpl, err := template.New("tests").Parse(source)
	if err != nil {
		t.Fatalf("cannot create test templates: %v", err)
	}
	return tmpl
}

func testAppData() (*AppData, []ArtistFull) {
	data := &AppData{
		Artists: []Artist{{Id: 1, Name: "Queen"}},
		Locations: LocationWrapper{LocWrapper: []Location{{
			Id:        1,
			Locations: []string{"london-uk"},
		}}},
		Dates: DateWrapper{DatWrapper: []Date{{
			Id:    1,
			Dates: []string{"01-01-2020"},
		}}},
		Relations: RelationWrapper{RelWrapper: []Relation{{
			Id:             1,
			DatesLocations: map[string][]string{"london-uk": {"01-01-2020"}},
		}}},
	}
	return data, GetFullArtists(data)
}

func TestRoutesHomeAndErrors(t *testing.T) {
	data, fullArtists := testAppData()
	handler := Routes(testTemplates(t), data, fullArtists)

	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{name: "home", method: http.MethodGet, path: "/", want: http.StatusOK},
		{name: "unknown route", method: http.MethodGet, path: "/does-not-exist", want: http.StatusNotFound},
		{name: "home wrong method", method: http.MethodPost, path: "/", want: http.StatusMethodNotAllowed},
		{name: "artists empty query", method: http.MethodGet, path: "/artists", want: http.StatusBadRequest},
		{name: "artists wrong method", method: http.MethodPost, path: "/artists?q=Queen", want: http.StatusMethodNotAllowed},
		{name: "api search empty query", method: http.MethodGet, path: "/api/search", want: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			res := httptest.NewRecorder()

			handler.ServeHTTP(res, req)

			if res.Code != tt.want {
				t.Fatalf("%s %s returned %d, want %d", tt.method, tt.path, res.Code, tt.want)
			}
		})
	}
}

func TestArtistsSearch(t *testing.T) {
	data := &AppData{}
	fullArtists := []ArtistFull{
		{Id: 1, Name: "Queen"},
		{Id: 2, Name: "Queens of the Stone Age"},
		{Id: 3, Name: "Metallica"},
	}
	handler := Routes(testTemplates(t), data, fullArtists)

	t.Run("partial query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/artists?q=queen", nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", res.Code, http.StatusOK)
		}
		body := res.Body.String()
		if !strings.Contains(body, "Queen") || !strings.Contains(body, "Queens of the Stone Age") {
			t.Fatalf("partial search did not return both matches: %q", body)
		}
		if strings.Contains(body, "Metallica") {
			t.Fatalf("partial search returned an unrelated artist: %q", body)
		}
	})

	t.Run("exact query has priority", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/artists?q=Queen", nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		body := res.Body.String()
		if !strings.Contains(body, "Queen") {
			t.Fatalf("exact search did not return Queen: %q", body)
		}
		if strings.Contains(body, "Queens of the Stone Age") {
			t.Fatalf("exact search returned additional matches: %q", body)
		}
	})
}

func TestAPIRoutesUseControlledData(t *testing.T) {
	data, fullArtists := testAppData()
	handler := Routes(testTemplates(t), data, fullArtists)

	tests := []struct {
		name string
		path string
	}{
		{name: "artists", path: "/api/artists"},
		{name: "locations", path: "/api/locations"},
		{name: "dates", path: "/api/dates"},
		{name: "relations", path: "/api/relations"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)

			if res.Code != http.StatusOK {
				t.Fatalf("%s returned %d, want %d", tt.path, res.Code, http.StatusOK)
			}
			if !strings.Contains(res.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("%s returned Content-Type %q", tt.path, res.Header().Get("Content-Type"))
			}
			if res.Body.Len() == 0 {
				t.Fatalf("%s returned an empty body", tt.path)
			}
		})
	}
}

func TestAPISearch(t *testing.T) {
	data := &AppData{}
	fullArtists := []ArtistFull{{Id: 1, Name: "Queen"}, {Id: 2, Name: "Metallica"}}
	handler := Routes(testTemplates(t), data, fullArtists)

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=quee", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", res.Code, http.StatusOK)
	}

	var got []string
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if len(got) != 1 || got[0] != "Queen" {
		t.Fatalf("unexpected search result: %v", got)
	}
}
