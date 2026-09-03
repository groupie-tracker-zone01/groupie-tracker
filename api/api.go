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

type ArtistFull struct {
	Id           int                 `json:"id"`
	Image        string              `json:"image"`
	Name         string              `json:"name"`
	Members      []string            `json:"members"`
	CreationDate int                 `json:"creationDate"`
	FirstAlbum   string              `json:"firstAlbum"`
	Locations    []string            `json:"locations"`
	Dates        []string            `json:"dates"`
	Relations    map[string][]string `json:"relations"`
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

func GetFullArtists(data *AppData) []ArtistFull {
	var fullArtists []ArtistFull

	for _, artist := range data.Artists {
		full := ArtistFull{
			Id:           artist.Id,
			Image:        artist.Image,
			Name:         artist.Name,
			Members:      artist.Members,
			CreationDate: artist.CreationDate,
			FirstAlbum:   artist.FirstAlbum,
		}

		for _, loc := range data.Locations.LocWrapper {
			if loc.Id == artist.Id {
				full.Locations = loc.Locations
				break
			}
		}

		for _, d := range data.Dates.DatWrapper {
			if d.Id == artist.Id {
				full.Dates = d.Dates
				break
			}
		}

		for _, rel := range data.Relations.RelWrapper {
			if rel.Id == artist.Id {
				full.Relations = rel.DatesLocations
				break
			}
		}

		fullArtists = append(fullArtists, full)
	}

	return fullArtists
}

func GetArtistFull(data *AppData, artistId int) *ArtistFull {
	for _, artist := range data.Artists {
		if artist.Id == artistId {
			full := &ArtistFull{
				Id:           artist.Id,
				Image:        artist.Image,
				Name:         artist.Name,
				Members:      artist.Members,
				CreationDate: artist.CreationDate,
				FirstAlbum:   artist.FirstAlbum,
			}

			for _, loc := range data.Locations.LocWrapper {
				if loc.Id == artistId {
					full.Locations = loc.Locations
					break
				}
			}

			for _, d := range data.Dates.DatWrapper {
				if d.Id == artistId {
					full.Dates = d.Dates
					break
				}
			}

			for _, rel := range data.Relations.RelWrapper {
				if rel.Id == artistId {
					full.Relations = rel.DatesLocations
					break
				}
			}

			return full
		}
	}
	return nil
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
