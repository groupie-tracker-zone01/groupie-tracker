package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	templates := template.Must(template.ParseFiles(
		filepath.Join(dir, "templates", "pages", "home.html"),
		filepath.Join(dir, "templates", "base", "header.html"),
		filepath.Join(dir, "templates", "base", "footer.html"),
	))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Title string
		}{
			Title: "Home - MetaRock",
		}

		err := templates.ExecuteTemplate(w, "home.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
		}
	})

	log.Println("Servidor rodando em http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
