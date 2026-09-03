package main

import (
	"api/api"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Regression test:
// The home route must stay connected to the API data
// and display artists on the home page.
func TestHome(t *testing.T) {
	data := &api.AppData{
		Artists: []api.Artist{
			{
				Id:    1,
				Name:  "Queen",
				Image: "https://example.com/queen.jpg",
			},
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	routes(data).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", response.Code, http.StatusOK)
	}

	body := response.Body.String()

	if !strings.Contains(body, "Queen") {
		t.Fatal("home page does not contain artist data")
	}

	if !strings.Contains(body, "https://example.com/queen.jpg") {
		t.Fatal("home page does not contain artist image")
	}
}

// Regression test:
// An unknown route must continue to return HTTP 404.
func TestNotFound(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	response := httptest.NewRecorder()

	routes(&api.AppData{}).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("got status %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestServerPort(t *testing.T) {
	t.Run("default port", func(t *testing.T) {
		os.Unsetenv(portEnv)

		if got := serverPort(); got != defaultPort {
			t.Fatalf("got port %q, want %q", got, defaultPort)
		}
	})

	t.Run("environment port", func(t *testing.T) {
		t.Setenv(portEnv, "9090")

		if got := serverPort(); got != "9090" {
			t.Fatalf("got port %q, want %q", got, "9090")
		}
	})
}
