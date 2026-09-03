package unit_test

// Unit tests for the API handlers.
//
// Goal: test each handler separately with controlled data.
// These tests do not call the real Groupie Tracker API.
// This makes the tests fast and independent from the Internet.

import (
	"api/api"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestArtistsHandler(t *testing.T) {
	// Create simple test data.
	// We know exactly what the handler should return.
	data := &api.AppData{
		Artists: []api.Artist{
			{
				Id:   1,
				Name: "Queen",
			},
		},
	}

	// httptest creates a fake HTTP request and response.
	// We do not need to start a real server.
	req := httptest.NewRequest(http.MethodGet, "/artists", nil)
	res := httptest.NewRecorder()

	api.ArtistsHandler(data).ServeHTTP(res, req)

	// The handler must return HTTP 200 OK.
	if res.Code != http.StatusOK {
		t.Fatalf("got status %d, expected %d", res.Code, http.StatusOK)
	}

	// A 200 status is not enough.
	// Decode the JSON to check the returned artist data.
	var artists []api.Artist
	if err := json.NewDecoder(res.Body).Decode(&artists); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(artists) != 1 || artists[0].Name != "Queen" {
		t.Fatalf("unexpected artist data: %+v", artists)
	}
}

func TestLocationsHandler(t *testing.T) {
	// Create simple data to test only the locations handler.
	data := &api.AppData{
		Locations: api.LocationWrapper{
			LocWrapper: []api.Location{
				{
					Id:        1,
					Locations: []string{"london-uk"},
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/locations", nil)
	res := httptest.NewRecorder()

	api.LocationsHandler(data).ServeHTTP(res, req)

	// The handler must return HTTP 200 OK.
	if res.Code != http.StatusOK {
		t.Fatalf("got status %d, expected %d", res.Code, http.StatusOK)
	}
}

func TestDatesHandler(t *testing.T) {
	// Create simple data to test only the dates handler.
	data := &api.AppData{
		Dates: api.DateWrapper{
			DatWrapper: []api.Date{
				{
					Id:    1,
					Dates: []string{"01-01-2020"},
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/dates", nil)
	res := httptest.NewRecorder()

	api.DatesHandler(data).ServeHTTP(res, req)

	// The handler must return HTTP 200 OK.
	if res.Code != http.StatusOK {
		t.Fatalf("got status %d, expected %d", res.Code, http.StatusOK)
	}
}

func TestRelationsHandler(t *testing.T) {
	// Create simple data to test only the relations handler.
	data := &api.AppData{
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

	req := httptest.NewRequest(http.MethodGet, "/relations", nil)
	res := httptest.NewRecorder()

	api.RelationsHandler(data).ServeHTTP(res, req)

	// The handler must return HTTP 200 OK.
	if res.Code != http.StatusOK {
		t.Fatalf("got status %d, expected %d", res.Code, http.StatusOK)
	}
}
