package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"unicode"
	"time"
)

type Artist struct {
	Id           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
}

// structure type for wrapping the index property that wraps other json properties
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

// Structure type to store all the data from the API
type AppData struct {
	Artists   []Artist
	Locations LocationWrapper
	Dates     DateWrapper
	Relations RelationWrapper
}

// Relations
type LastConcert struct {
	City        string
	DisplayCity string
	Dates       []string
}

// Structure type to store all the data of an artist
type ArtistFull struct {
	Id           int                 `json:"id"`
	Image        string              `json:"image"`
	Name         string              `json:"name"`
	Members      []string            `json:"members"`
	CreationDate int                 `json:"creationDate"`
	FirstAlbum   string              `json:"firstAlbum"`
	Locations        []string            `json:"locations"`
	DisplayLocations []string            `json:"-"`
	Dates            []string            `json:"dates"`
	Relations    map[string][]string `json:"relations"`
	LastConcerts []LastConcert       `json:"lastConcerts"`
}

const (
	baseURL           = "https://groupietrackers.herokuapp.com/api"
	apiRequestTimeout = 5 * time.Second
)

var apiHTTPClient = &http.Client{Timeout: apiRequestTimeout}

func LoadData() (*AppData, error) {
	return loadDataContext(context.Background(), apiHTTPClient)
}

func loadDataContext(ctx context.Context, client *http.Client) (*AppData, error) {
	data := &AppData{}
	if err := fetchJSONContext(ctx, client, baseURL+"/artists", &data.Artists); err != nil {
		return nil, fmt.Errorf("chargement artists: %w", err)
	}
	if err := fetchJSONContext(ctx, client, baseURL+"/locations", &data.Locations); err != nil {
		return nil, fmt.Errorf("chargement locations: %w", err)
	}
	if err := fetchJSONContext(ctx, client, baseURL+"/dates", &data.Dates); err != nil {
		return nil, fmt.Errorf("chargement dates: %w", err)
	}
	if err := fetchJSONContext(ctx, client, baseURL+"/relation", &data.Relations); err != nil {
		return nil, fmt.Errorf("chargement relation: %w", err)
	}
	return data, nil
}

func fetchJSON(url string, target any) error {
	return fetchJSONContext(context.Background(), apiHTTPClient, url, target)
}

func fetchJSONContext(ctx context.Context, client *http.Client, url string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status: %s", response.Status)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func sortDates(dates []string) []string {
	sorted := make([]string, len(dates))
	copy(sorted, dates)

	sort.Slice(sorted, func(i, j int) bool {
		t1, _ := time.Parse("02-01-2006", sorted[i])
		t2, _ := time.Parse("02-01-2006", sorted[j])
		return t1.Before(t2)
	})
	return sorted
}

func formatLocations(locations []string) []string {
	formatted := make([]string, 0, len(locations))
	for _, location := range locations {
		formatted = append(formatted, formatLocation(location))
	}
	return formatted
}

func formatLocation(location string) string {
	location = strings.TrimSpace(strings.ReplaceAll(location, "_", " "))
	if location == "" {
		return ""
	}

	city := location
	country := ""
	if separator := strings.LastIndex(location, "-"); separator > 0 && separator < len(location)-1 {
		city = location[:separator]
		country = location[separator+1:]
	}

	city = titleWords(city)
	if country == "" {
		return city
	}

	country = strings.TrimSpace(country)
	switch strings.ToLower(country) {
	case "usa", "uk":
		country = strings.ToUpper(country)
	default:
		country = titleWords(country)
	}

	return city + ", " + country
}

func titleWords(value string) string {
	words := strings.Fields(value)
	for i, word := range words {
		parts := strings.Split(word, "-")
		for j, part := range parts {
			parts[j] = capitalizeWord(part)
		}
		words[i] = strings.Join(parts, "-")
	}
	return strings.Join(words, " ")
}

func capitalizeWord(value string) string {
	runes := []rune(strings.ToLower(value))
	if len(runes) == 0 {
		return ""
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
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
		sort.Strings(full.Members)
		for _, loc := range data.Locations.LocWrapper {
			if loc.Id == artist.Id {
				full.Locations = loc.Locations
				sort.Strings(full.Locations)
				full.DisplayLocations = formatLocations(full.Locations)
				break
			}
		}
		for _, d := range data.Dates.DatWrapper {
			if d.Id == artist.Id {
				full.Dates = sortDates(d.Dates)
				break
			}
		}
		for _, rel := range data.Relations.RelWrapper {
			if rel.Id == artist.Id {
				full.Relations = rel.DatesLocations
				break
			}
		}
		for city, dates := range full.Relations {
			if len(dates) > 0 {
				sortedDates := sortDates(dates)
				full.LastConcerts = append(full.LastConcerts, LastConcert{
					City:        city,
					DisplayCity: formatLocation(city),
					Dates:       sortedDates,
				})
			}
		}
		sort.Slice(full.LastConcerts, func(i, j int) bool {
			li := full.LastConcerts[i].Dates[len(full.LastConcerts[i].Dates)-1]
			lj := full.LastConcerts[j].Dates[len(full.LastConcerts[j].Dates)-1]
			return li > lj
		})
		fullArtists = append(fullArtists, full)
	}
	sort.Slice(fullArtists, func(i, j int) bool {
		return fullArtists[i].Name < fullArtists[j].Name
	})
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
			sort.Strings(full.Members)
			for _, loc := range data.Locations.LocWrapper {
				if loc.Id == artistId {
					full.Locations = loc.Locations
					sort.Strings(full.Locations)
					full.DisplayLocations = formatLocations(full.Locations)
					break
				}
			}
			for _, d := range data.Dates.DatWrapper {
				if d.Id == artistId {
					full.Dates = sortDates(d.Dates)
					break
				}
			}
			for _, rel := range data.Relations.RelWrapper {
				if rel.Id == artistId {
					full.Relations = rel.DatesLocations
					break
				}
			}
			for city, dates := range full.Relations {
				if len(dates) > 0 {
					sortedDates := sortDates(dates)
					full.LastConcerts = append(full.LastConcerts, LastConcert{
						City:        city,
						DisplayCity: formatLocation(city),
						Dates:       sortedDates,
					})
				}
			}
			sort.Slice(full.LastConcerts, func(i, j int) bool {
				li := full.LastConcerts[i].Dates[len(full.LastConcerts[i].Dates)-1]
				lj := full.LastConcerts[j].Dates[len(full.LastConcerts[j].Dates)-1]
				return li > lj
			})
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
