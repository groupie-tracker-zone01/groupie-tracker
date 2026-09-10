package main

import (
	"fmt"
	"groupie-tracker/server"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultPort             = "8080"
	portEnv                 = "PORT"
	serverReadHeaderTimeout = 5 * time.Second
	serverReadTimeout       = 10 * time.Second
	serverWriteTimeout      = 15 * time.Second
	serverIdleTimeout       = 60 * time.Second
)

func main() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	data, err := server.LoadData()
	if err != nil {
		log.Fatal(err)
	}
	// API full data artists
	fullArtists := server.GetFullArtists(data)
	templates := template.Must(template.ParseFiles(
		filepath.Join(dir, "templates", "pages", "home.html"),
		filepath.Join(dir, "templates", "pages", "artists.html"),
		filepath.Join(dir, "templates", "base", "header.html"),
		filepath.Join(dir, "templates", "base", "footer.html"),
		filepath.Join(dir, "templates", "pages", "errors", "400.html"),
		filepath.Join(dir, "templates", "pages", "errors", "403.html"),
		filepath.Join(dir, "templates", "pages", "errors", "404.html"),
		filepath.Join(dir, "templates", "pages", "errors", "405.html"),
		filepath.Join(dir, "templates", "pages", "errors", "500.html"),
	))
	router := server.Routes(templates, data, fullArtists)
	port := serverPort()
	serverHTTP := newHTTPServer(port, router)
	fmt.Printf("Server running at http://localhost:%s\n", port)
	if err := serverHTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error starting server: %v", err)
	}
}

func serverPort() string {
	if port := os.Getenv(portEnv); port != "" {
		return port
	}
	return defaultPort
}

func newHTTPServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
	}
}
