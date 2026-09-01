package main

import (
	"api/api"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHome(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	routes(&api.AppData{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("statut reçu %d, attendu %d", response.Code, http.StatusOK)
	}

	if !strings.Contains(response.Body.String(), "Groupie Tracker") {
		t.Fatal("la page temporaire ne contient pas le titre attendu")
	}
}

func TestNotFound(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/inconnue", nil)
	response := httptest.NewRecorder()

	routes(&api.AppData{}).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("statut reçu %d, attendu %d", response.Code, http.StatusNotFound)
	}
}

func TestServerPort(t *testing.T) {
	t.Run("port par défaut", func(t *testing.T) {
		t.Setenv(portEnv, "")

		if port := serverPort(); port != defaultPort {
			t.Fatalf("port reçu %q, attendu %q", port, defaultPort)
		}
	})

	t.Run("port configuré", func(t *testing.T) {
		const configuredPort = "9090"
		t.Setenv(portEnv, configuredPort)

		if port := serverPort(); port != configuredPort {
			t.Fatalf("port reçu %q, attendu %q", port, configuredPort)
		}
	})
}
