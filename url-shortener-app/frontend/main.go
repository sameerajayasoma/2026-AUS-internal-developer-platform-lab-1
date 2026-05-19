package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	apiURL := os.Getenv("API_SERVICE_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	mux := http.NewServeMux()

	// Proxy routes - order matters: more specific first
	mux.HandleFunc("/api/shorten", proxyHandler(apiURL))
	mux.HandleFunc("/api/urls/", proxyHandler(apiURL))
	mux.HandleFunc("/api/urls", proxyHandler(apiURL))
	mux.HandleFunc("/api/analytics/", proxyHandler(apiURL))
	mux.HandleFunc("/r/", proxyHandler(apiURL))

	// Static files (SPA fallback)
	mux.Handle("/", staticHandler())

	handler := loggingMiddleware(mux)

	log.Printf("Frontend BFF listening on :%s", port)
	log.Printf("  API proxy -> %s", apiURL)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}
