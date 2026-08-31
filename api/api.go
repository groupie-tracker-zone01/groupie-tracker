package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Artist struct {
	Id           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
}

// Type de structure enveloppe pour mémoriser la propriété Index qui englobe les autres propriétés
type LocationWrapper struct {
	LocWrapper []Location `json:"index"`
}

type Location struct {
	Id        int      `json:"id"`
	Locations []string `json:"locations"`
	Dates     string   `json:"dates"`
}

type DateWrapper struct {
	DatWrapper []Date `json:"index"`
}

type Date struct {
	Id    int      `json:"id"`
	Dates []string `json:"dates"`
}

type RelationWrapper struct {
	RelWrapper []Relation `json:"index"`
}

type Relation struct {
	Id             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

type AppData struct {
	Artists   []Artist
	Locations LocationWrapper
	Dates     DateWrapper
	Relations RelationWrapper
}

const baseURL = "https://groupietrackers.herokuapp.com/api"

func LoadData() (*AppData, error) {
	data := &AppData{}
	if err := fetchJSON(baseURL+"/artists", &data.Artists); err != nil {
		return nil, fmt.Errorf("chargement artists: %w", err)
	}
	if err := fetchJSON(baseURL+"/locations", &data.Locations); err != nil {
		return nil, fmt.Errorf("chargement locations: %w", err)
	}
	if err := fetchJSON(baseURL+"/dates", &data.Dates); err != nil {
		return nil, fmt.Errorf("chargement dates: %w", err)
	}
	if err := fetchJSON(baseURL+"/relation", &data.Relations); err != nil {
		return nil, fmt.Errorf("chargement relation: %w", err)
	}
	return data, nil
}

func fetchJSON(url string, target any) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status: %s", response.Status)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func ArtistsHandler(data *AppData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(data.Artists); err != nil {
			http.Error(w, "erreur de réponse", http.StatusInternalServerError)
		}
	}
}

func LocationsHandler(data *AppData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data.Locations)
	}
}

func DatesHandler(data *AppData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data.Dates)
	}
}

func RelationsHandler(data *AppData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data.Relations)
	}
}
