package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"groupie-tracker/server"
)

func TestFinalRecipeAccessibleStructure(t *testing.T) {
	data, fullArtists := applicationTestData()
	handler := server.Routes(applicationTemplates(t), data, fullArtists)

	t.Run("home has one valid document and keyboard landmarks", func(t *testing.T) {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))

		if res.Code != http.StatusOK {
			t.Fatalf("GET / returned %d, want %d", res.Code, http.StatusOK)
		}

		body := res.Body.String()
		if got := strings.Count(body, "<!DOCTYPE html>"); got != 1 {
			t.Fatalf("home rendered %d document declarations, want exactly 1", got)
		}
		if got := strings.Count(body, "<body>"); got != 1 {
			t.Fatalf("home rendered %d body elements, want exactly 1", got)
		}

		for _, expected := range []string{
			`lang="en"`,
			`class="skip-link" href="#main-content"`,
			`aria-label="Main navigation"`,
			`id="main-content"`,
			`for="artist-query"`,
			`id="artist-query"`,
			`id="carousel-toggle"`,
			`aria-label="Previous slide"`,
			`aria-label="Next slide"`,
			`alt="MetaRock logo"`,
			`/static/css/components/accessibility.css`,
		} {
			if !strings.Contains(body, expected) {
				t.Fatalf("home page does not contain accessibility marker %q", expected)
			}
		}
	})

	t.Run("artist results keep accessible structure", func(t *testing.T) {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/artists?q=Queen", nil))

		if res.Code != http.StatusOK {
			t.Fatalf("GET /artists?q=Queen returned %d, want %d", res.Code, http.StatusOK)
		}

		body := res.Body.String()
		for _, expected := range []string{
			`class="skip-link" href="#main-content"`,
			`id="main-content"`,
			`<h1 class="title-card" id="results-title">Artist information</h1>`,
			`<h2 class="artist-name">Queen</h2>`,
			`alt="Portrait of Queen"`,
		} {
			if !strings.Contains(body, expected) {
				t.Fatalf("artists page does not contain accessibility marker %q", expected)
			}
		}
	})

	t.Run("empty search result gives a working recovery link", func(t *testing.T) {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/artists?q=Nobody", nil))

		if res.Code != http.StatusOK {
			t.Fatalf("GET /artists?q=Nobody returned %d, want %d", res.Code, http.StatusOK)
		}
		body := res.Body.String()
		if !strings.Contains(body, `href="/#artist-search"`) || !strings.Contains(body, "Try another search") {
			t.Fatalf("empty results page does not expose the recovery search link: %q", body)
		}
	})
}

func TestFinalRecipeImagesExposeAltAttributes(t *testing.T) {
	data, fullArtists := applicationTestData()
	handler := server.Routes(applicationTemplates(t), data, fullArtists)
	imagePattern := regexp.MustCompile(`<img\b[^>]*>`)

	for _, path := range []string{"/", "/artists?q=Que"} {
		t.Run(path, func(t *testing.T) {
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
			if res.Code != http.StatusOK {
				t.Fatalf("GET %s returned %d, want %d", path, res.Code, http.StatusOK)
			}

			for _, tag := range imagePattern.FindAllString(res.Body.String(), -1) {
				if !strings.Contains(tag, " alt=") {
					t.Fatalf("image is missing an alt attribute on %s: %s", path, tag)
				}
			}
		})
	}
}

func TestFinalRecipeNavigationTargetsRespond(t *testing.T) {
	data, fullArtists := applicationTestData()
	handler := server.Routes(applicationTemplates(t), data, fullArtists)

	for _, path := range []string{"/", "/artists?q=Queen", "/#artist-search"} {
		t.Run(path, func(t *testing.T) {
			reqPath := strings.Split(path, "#")[0]
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, reqPath, nil))
			if res.Code != http.StatusOK {
				t.Fatalf("navigation target %s returned %d, want %d", path, res.Code, http.StatusOK)
			}
		})
	}
}

func TestFinalRecipeAccessibilityStylesAreServed(t *testing.T) {
	data, fullArtists := applicationTestData()
	handler := server.Routes(applicationTemplates(t), data, fullArtists)

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/static/css/components/accessibility.css", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("accessibility stylesheet returned %d, want %d", res.Code, http.StatusOK)
	}
	if res.Body.Len() == 0 {
		t.Fatal("accessibility stylesheet returned an empty body")
	}
}
