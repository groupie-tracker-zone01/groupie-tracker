package server

import (
	"encoding/json"
	"fmt"
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
{{define "artists"}}ARTISTS{{range .Artists}} {{.Name}}{{end}}{{with .SelectedArtist}} DETAIL {{.Name}}{{end}}{{end}}
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

func realErrorTemplates(t *testing.T) *template.Template {
	t.Helper()

	tmpl, err := template.ParseFiles(
		"../templates/base/footer.html",
		"../templates/pages/errors/400.html",
		"../templates/pages/errors/403.html",
		"../templates/pages/errors/404.html",
		"../templates/pages/errors/405.html",
		"../templates/pages/errors/500.html",
	)
	if err != nil {
		t.Fatalf("cannot parse real error templates: %v", err)
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

func TestRenderErrorsUsesRealTemplates(t *testing.T) {
	templates := realErrorTemplates(t)

	tests := []struct {
		status int
		text   string
	}{
		{status: http.StatusBadRequest, text: "Error 400 - Bad Request"},
		{status: http.StatusForbidden, text: "Error 403 - Forbidden"},
		{status: http.StatusNotFound, text: "Error 404 - Not Found"},
		{status: http.StatusMethodNotAllowed, text: "Error 405 - Method Not Allowed"},
		{status: http.StatusInternalServerError, text: "Error 500 - StatusInternalServerError"},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			res := httptest.NewRecorder()
			renderErrors(res, tt.status, templates)

			if res.Code != tt.status {
				t.Fatalf("renderErrors returned status %d, want %d", res.Code, tt.status)
			}
			if !strings.Contains(res.Body.String(), tt.text) {
				t.Fatalf("rendered error page does not contain %q: %q", tt.text, res.Body.String())
			}
		})
	}
}

func TestArtistsSearch(t *testing.T) {
	data := &AppData{}
	fullArtists := []ArtistFull{
		{
			Id:           1,
			Name:         "Queen",
			Members:      []string{"Freddie Mercury"},
			CreationDate: 1970,
			FirstAlbum:   "13-07-1973",
			Locations:    []string{"london-uk"},
		},
		{
			Id:           2,
			Name:         "Queens of the Stone Age",
			Members:      []string{"Josh Homme"},
			CreationDate: 1996,
			FirstAlbum:   "22-09-1998",
			Locations:    []string{"los_angeles-usa"},
		},
		{Id: 3, Name: "Metallica", Members: []string{"James Hetfield"}, CreationDate: 1981},
	}
	handler := Routes(testTemplates(t), data, fullArtists)

	tests := []struct {
		name      string
		query     string
		want      []string
		notWanted []string
	}{
		{
			name:      "artist name is case insensitive and keeps partial matches",
			query:     "QUEEN",
			want:      []string{"Queen", "Queens of the Stone Age"},
			notWanted: []string{"Metallica"},
		},
		{
			name:      "member",
			query:     "freddie",
			want:      []string{"Queen"},
			notWanted: []string{"Queens of the Stone Age", "Metallica"},
		},
		{
			name:      "location accepts readable spaces",
			query:     "los angeles usa",
			want:      []string{"Queens of the Stone Age"},
			notWanted: []string{"Queen", "Metallica"},
		},
		{
			name:      "creation date",
			query:     "1970",
			want:      []string{"Queen"},
			notWanted: []string{"Queens of the Stone Age", "Metallica"},
		},
		{
			name:      "first album",
			query:     "13-07-1973",
			want:      []string{"Queen"},
			notWanted: []string{"Queens of the Stone Age", "Metallica"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/artists?q="+tt.query, nil)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)

			if res.Code != http.StatusOK {
				t.Fatalf("got status %d, want %d", res.Code, http.StatusOK)
			}
			body := res.Body.String()
			for _, expected := range tt.want {
				if !strings.Contains(body, expected) {
					t.Fatalf("search %q did not contain %q: %q", tt.query, expected, body)
				}
			}
			for _, unexpected := range tt.notWanted {
				if strings.Contains(body, unexpected) {
					t.Fatalf("search %q unexpectedly contained %q: %q", tt.query, unexpected, body)
				}
			}
		})
	}
}

func TestArtistsPaginationAndDetail(t *testing.T) {
	data := &AppData{}
	fullArtists := make([]ArtistFull, 0, 12)
	for i := 1; i <= 12; i++ {
		fullArtists = append(fullArtists, ArtistFull{Id: i, Name: fmt.Sprintf("Band%02d", i)})
	}
	handler := Routes(testTemplates(t), data, fullArtists)

	t.Run("second page with five results", func(t *testing.T) {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/artists?q=Band&limit=5&page=2", nil))

		if res.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", res.Code, http.StatusOK)
		}
		body := res.Body.String()
		for _, expected := range []string{"Band06", "Band07", "Band08", "Band09", "Band10"} {
			if !strings.Contains(body, expected) {
				t.Fatalf("page 2 does not contain %q: %q", expected, body)
			}
		}
		for _, unexpected := range []string{"Band01", "Band05", "Band11", "Band12"} {
			if strings.Contains(body, unexpected) {
				t.Fatalf("page 2 unexpectedly contains %q: %q", unexpected, body)
			}
		}
	})

	t.Run("detail route selects one artist", func(t *testing.T) {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/artist?id=7", nil))

		if res.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", res.Code, http.StatusOK)
		}
		if !strings.Contains(res.Body.String(), "DETAIL Band07") {
			t.Fatalf("detail page does not contain selected artist: %q", res.Body.String())
		}
	})

	t.Run("unknown artist returns 404", func(t *testing.T) {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/artist?id=99", nil))
		if res.Code != http.StatusNotFound {
			t.Fatalf("got status %d, want %d", res.Code, http.StatusNotFound)
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
