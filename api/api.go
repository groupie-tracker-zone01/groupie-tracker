package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type API struct {
	Artists   string `json:"artists"`
	Locations string `json:"locations"`
	Dates     string `json:"dates"`
	Relations string `json:"relation"`
}

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

// Gère l'affichage des données sur la page lorsque le suffixe de l'URL est /api.
func ApiHandler(w http.ResponseWriter, r *http.Request) {
	var api API
	res := fetchJSON("https://groupietrackers.herokuapp.com/api", &api)
	if res != nil {
		http.Error(w, res.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(api)
}

// Gère l'affichage des données sur la page selon l'URL reçu.
func DataHandler(w http.ResponseWriter, r *http.Request) {
	var data any
	url := "https://groupietrackers.herokuapp.com/api" + r.URL.String()
	switch r.URL.String() {
	case "/artists":
		data = &[]Artist{}
	case "/locations":
		data = LocationWrapper{}
	case "/dates":
		data = DateWrapper{}
	case "/relation":
		data = RelationWrapper{}
	}
	res := fetchJSON(url, &data)
	if res != nil {
		http.Error(w, res.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
