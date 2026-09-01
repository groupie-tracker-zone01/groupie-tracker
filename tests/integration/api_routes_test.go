package integration_test

// Integration tests for the MVP.
//
// Unit tests check components separately.
// Here, several components work together.
// All handlers use the same AppData through one HTTP router.

import (
	"api/api"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIHandlersIntegration(t *testing.T) {
	// Create one small and consistent data set.
	// It contains an artist, a location, a date and a relation.
	// This lets us test the integration without using the Internet.
	data := &api.AppData{
		Artists: []api.Artist{
			{
				Id:   1,
				Name: "Queen",
			},
		},
		Locations: api.LocationWrapper{
			LocWrapper: []api.Location{
				{
					Id:        1,
					Locations: []string{"london-uk"},
				},
			},
		},
		Dates: api.DateWrapper{
			DatWrapper: []api.Date{
				{
					Id:    1,
					Dates: []string{"01-01-2020"},
				},
			},
		},
		Relations: api.RelationWrapper{
			RelWrapper: []api.Relation{
				{
					Id: 1,
					DatesLocations: map[string][]string{
						"london-uk": []string{"01-01-2020"},
					},
				},
			},
		},
	}

	// Create a test router with all API handlers.
	// Every handler uses the same AppData.
	mux := http.NewServeMux()
	mux.HandleFunc("/artists", api.ArtistsHandler(data))
	mux.HandleFunc("/locations", api.LocationsHandler(data))
	mux.HandleFunc("/dates", api.DatesHandler(data))
	mux.HandleFunc("/relations", api.RelationsHandler(data))

	// Test every MVP API route with the same logic.
	tests := []struct {
		name string
		path string
	}{
		{"artists", "/artists"},
		{"locations", "/locations"},
		{"dates", "/dates"},
		{"relations", "/relations"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			res := httptest.NewRecorder()

			mux.ServeHTTP(res, req)

			// Every route must return HTTP 200 OK.
			if res.Code != http.StatusOK {
				t.Fatalf("%s returned %d, expected %d", tt.path, res.Code, http.StatusOK)
			}

			// HTTP 200 with an empty body is not enough.
			// The handler must also return data.
			if res.Body.Len() == 0 {
				t.Fatalf("%s returned an empty body", tt.path)
			}
		})
	}
}
