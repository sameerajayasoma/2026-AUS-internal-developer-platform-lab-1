package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	titleRe   = regexp.MustCompile(`(?i)<title[^>]*>(.*?)</title>`)
	faviconRe = regexp.MustCompile(`(?i)<link[^>]+rel=["'](?:shortcut )?icon["'][^>]+href=["']([^"']+)["']`)
)

var metadataClient = &http.Client{Timeout: 5 * time.Second}

func fetchMetadata(ctx context.Context, shortCode, originalURL string, store *Store) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", originalURL, nil)
	if err != nil {
		slog.Warn("metadata: bad request", "url", originalURL, "error", err)
		return
	}
	req.Header.Set("User-Agent", "snip-bot/1.0")

	resp, err := metadataClient.Do(req)
	if err != nil {
		slog.Warn("metadata: fetch failed", "url", originalURL, "error", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return
	}
	html := string(body)
	title := extractTitle(html)
	favicon := extractFavicon(html, originalURL)
	if title != "" || favicon != "" {
		store.UpdateMetadata(shortCode, title, favicon)
	}
}

func extractTitle(html string) string {
	m := titleRe.FindStringSubmatch(html)
	if len(m) <= 1 {
		return ""
	}
	t := strings.TrimSpace(m[1])
	if len(t) > 200 {
		t = t[:200]
	}
	return t
}

func extractFavicon(html, baseURL string) string {
	m := faviconRe.FindStringSubmatch(html)
	if len(m) <= 1 {
		return ""
	}
	href := strings.TrimSpace(m[1])
	if strings.HasPrefix(href, "http") {
		return href
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	parts := strings.SplitN(baseURL, "//", 2)
	if len(parts) != 2 {
		return ""
	}
	host := strings.SplitN(parts[1], "/", 2)[0]
	scheme := parts[0]
	if strings.HasPrefix(href, "/") {
		return scheme + "//" + host + href
	}
	return scheme + "//" + host + "/" + href
}
