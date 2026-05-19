package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

type ShortenRequest struct {
	URL        string `json:"url"`
	Username   string `json:"username"`
	CustomSlug string `json:"custom_slug,omitempty"`
}

type ShortenResponse struct {
	ShortCode string `json:"short_code"`
	URL       *URL   `json:"url"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type AnalyticsResponse struct {
	ShortCode    string        `json:"short_code"`
	OriginalURL  string        `json:"original_url"`
	Title        string        `json:"title"`
	FaviconURL   string        `json:"favicon_url"`
	ClickCount   int64         `json:"click_count"`
	CreatedAt    string        `json:"created_at"`
	RecentClicks []ClickRecord `json:"recent_clicks"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func generateShortCode() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// POST /api/shorten
func handleShorten(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ShortenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
			return
		}
		if req.URL == "" || req.Username == "" {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "url and username are required"})
			return
		}
		if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
			req.URL = "https://" + req.URL
		}
		parsed, err := url.ParseRequestURI(req.URL)
		if err != nil || parsed.Host == "" {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid URL"})
			return
		}

		shortCode := req.CustomSlug
		if shortCode != "" {
			if len(shortCode) < 3 || len(shortCode) > 20 {
				writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "custom slug must be 3-20 characters"})
				return
			}
			if store.Exists(shortCode) {
				writeJSON(w, http.StatusConflict, ErrorResponse{Error: "slug already taken"})
				return
			}
		} else {
			for {
				shortCode = generateShortCode()
				if !store.Exists(shortCode) {
					break
				}
			}
		}

		u := store.Insert(shortCode, req.URL, req.Username)

		go fetchMetadata(context.Background(), shortCode, req.URL, store)

		writeJSON(w, http.StatusCreated, ShortenResponse{ShortCode: shortCode, URL: u})
	}
}

// GET /r/{code}
func handleRedirect(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := strings.TrimPrefix(r.URL.Path, "/r/")
		if code == "" {
			http.NotFound(w, r)
			return
		}
		u, err := store.Get(code)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		go store.RecordClick(code)
		http.Redirect(w, r, u.OriginalURL, http.StatusFound)
	}
}

// GET /api/urls?username=X
func handleListURLs(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		if username == "" {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "username query parameter required"})
			return
		}
		writeJSON(w, http.StatusOK, store.List(username))
	}
}

// DELETE /api/urls/{code}
func handleDeleteURL(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := strings.TrimPrefix(r.URL.Path, "/api/urls/")
		if code == "" {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "short code required"})
			return
		}
		if err := store.Delete(code); err != nil {
			if errors.Is(err, ErrNotFound) {
				writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// GET /api/analytics/{code}
func handleGetAnalytics(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := strings.TrimPrefix(r.URL.Path, "/api/analytics/")
		if code == "" || strings.Contains(code, "/") {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "short code required"})
			return
		}
		u, err := store.Get(code)
		if err != nil {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "not found"})
			return
		}
		recent := store.RecentClicks(code, 50)
		writeJSON(w, http.StatusOK, AnalyticsResponse{
			ShortCode:    u.ShortCode,
			OriginalURL:  u.OriginalURL,
			Title:        u.Title,
			FaviconURL:   u.FaviconURL,
			ClickCount:   u.ClickCount,
			CreatedAt:    u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			RecentClicks: recent,
		})
	}
}

// GET /api/analytics/user/{username}
func handleGetUserAnalytics(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := strings.TrimPrefix(r.URL.Path, "/api/analytics/user/")
		if username == "" {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "username required"})
			return
		}
		urls := store.UserURLs(username)
		var totalClicks int64
		for _, u := range urls {
			totalClicks += u.ClickCount
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"username":     username,
			"total_urls":   len(urls),
			"total_clicks": totalClicks,
			"urls":         urls,
		})
	}
}

// GET /api/analytics/top
func handleGetTopURLs(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		urls := store.Top(50)
		var totalClicks int64
		for _, u := range urls {
			totalClicks += u.ClickCount
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"total_urls":   len(urls),
			"total_clicks": totalClicks,
			"urls":         urls,
		})
	}
}

// GET /health
func handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
