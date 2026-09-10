package server

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestFetchJSONSuccess(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":1,"name":"Queen"}]`))
	}))
	defer testServer.Close()

	var artists []Artist
	if err := fetchJSON(testServer.URL, &artists); err != nil {
		t.Fatalf("fetchJSON returned an unexpected error: %v", err)
	}
	if len(artists) != 1 || artists[0].Name != "Queen" {
		t.Fatalf("unexpected decoded artists: %+v", artists)
	}
}

func TestFetchJSONStatusError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}))
	defer testServer.Close()

	var target any
	err := fetchJSON(testServer.URL, &target)
	if err == nil {
		t.Fatal("fetchJSON should return an error for a non-200 response")
	}
	if !strings.Contains(err.Error(), "502") && !strings.Contains(err.Error(), "Bad Gateway") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchJSONInvalidJSON(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"broken":`))
	}))
	defer testServer.Close()

	var target any
	if err := fetchJSON(testServer.URL, &target); err == nil {
		t.Fatal("fetchJSON should return an error for invalid JSON")
	}
}

func TestSortDates(t *testing.T) {
	input := []string{"31-12-2024", "01-01-2023", "15-06-2024"}
	original := append([]string(nil), input...)

	got := sortDates(input)
	want := []string{"01-01-2023", "15-06-2024", "31-12-2024"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sortDates() = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(input, original) {
		t.Fatalf("sortDates modified its input: got %v, want %v", input, original)
	}
}

func TestGetFullArtistsBuildsAndSortsData(t *testing.T) {
	data := &AppData{
		Artists: []Artist{
			{Id: 2, Name: "ZZ Top", Members: []string{"B", "A"}},
			{Id: 1, Name: "Queen", Members: []string{"Freddie", "Brian"}},
		},
		Locations: LocationWrapper{LocWrapper: []Location{
			{Id: 1, Locations: []string{"paris-france", "london-uk"}},
		}},
		Dates: DateWrapper{DatWrapper: []Date{
			{Id: 1, Dates: []string{"31-12-2024", "01-01-2023"}},
		}},
		Relations: RelationWrapper{RelWrapper: []Relation{
			{Id: 1, DatesLocations: map[string][]string{"london-uk": {"31-12-2024", "01-01-2023"}}},
		}},
	}

	got := GetFullArtists(data)
	if len(got) != 2 {
		t.Fatalf("GetFullArtists returned %d artists, want 2", len(got))
	}
	if got[0].Name != "Queen" || got[1].Name != "ZZ Top" {
		t.Fatalf("artists are not sorted by name: %+v", got)
	}
	if !reflect.DeepEqual(got[0].Members, []string{"Brian", "Freddie"}) {
		t.Fatalf("members are not sorted: %v", got[0].Members)
	}
	if !reflect.DeepEqual(got[0].Locations, []string{"london-uk", "paris-france"}) {
		t.Fatalf("locations are not sorted: %v", got[0].Locations)
	}
	if !reflect.DeepEqual(got[0].Dates, []string{"01-01-2023", "31-12-2024"}) {
		t.Fatalf("dates are not sorted: %v", got[0].Dates)
	}
	if len(got[0].LastConcerts) != 1 || got[0].LastConcerts[0].City != "london-uk" {
		t.Fatalf("unexpected last concerts: %+v", got[0].LastConcerts)
	}
	if !reflect.DeepEqual(got[0].LastConcerts[0].Dates, []string{"01-01-2023", "31-12-2024"}) {
		t.Fatalf("concert dates are not sorted: %v", got[0].LastConcerts[0].Dates)
	}
}

func TestGetArtistFull(t *testing.T) {
	data := &AppData{
		Artists:   []Artist{{Id: 1, Name: "Queen"}},
		Locations: LocationWrapper{LocWrapper: []Location{{Id: 1, Locations: []string{"london-uk"}}}},
		Dates:     DateWrapper{DatWrapper: []Date{{Id: 1, Dates: []string{"01-01-2020"}}}},
		Relations: RelationWrapper{RelWrapper: []Relation{{Id: 1, DatesLocations: map[string][]string{"london-uk": {"01-01-2020"}}}}},
	}

	artist := GetArtistFull(data, 1)
	if artist == nil || artist.Name != "Queen" {
		t.Fatalf("unexpected artist: %+v", artist)
	}
	if missing := GetArtistFull(data, 999); missing != nil {
		t.Fatalf("GetArtistFull returned %+v for an unknown id", missing)
	}
}
