package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	store := NewStore()

	mux := http.NewServeMux()
	// api-service surface
	mux.HandleFunc("POST /api/shorten", handleShorten(store))
	mux.HandleFunc("GET /r/", handleRedirect(store))
	mux.HandleFunc("GET /api/urls", handleListURLs(store))
	mux.HandleFunc("DELETE /api/urls/", handleDeleteURL(store))
	// analytics-service surface
	mux.HandleFunc("GET /api/analytics/top", handleGetTopURLs(store))
	mux.HandleFunc("GET /api/analytics/user/", handleGetUserAnalytics(store))
	mux.HandleFunc("GET /api/analytics/", handleGetAnalytics(store))
	// shared
	mux.HandleFunc("GET /health", handleHealth())

	handler := loggingMiddleware(corsMiddleware(mux))

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("snip-backend listening", "port", port)
	log.Fatal(srv.ListenAndServe())
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Tracing-Enabled")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		slog.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
