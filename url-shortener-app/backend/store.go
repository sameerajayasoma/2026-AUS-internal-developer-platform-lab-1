package main

import (
	"errors"
	"sort"
	"sync"
	"time"
)

type URL struct {
	ID          int64     `json:"id"`
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	Title       string    `json:"title"`
	FaviconURL  string    `json:"favicon_url"`
	Username    string    `json:"username"`
	ClickCount  int64     `json:"click_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ClickRecord struct {
	ClickedAt time.Time `json:"clicked_at"`
}

var ErrNotFound = errors.New("not found")

const recentClicksCap = 50

type Store struct {
	mu     sync.RWMutex
	urls   map[string]*URL        // shortCode -> URL
	clicks map[string][]time.Time // shortCode -> timestamps, newest first
	nextID int64
}

func NewStore() *Store {
	return &Store{
		urls:   make(map[string]*URL),
		clicks: make(map[string][]time.Time),
	}
}

func (s *Store) Insert(shortCode, originalURL, username string) *URL {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	now := time.Now().UTC()
	u := &URL{
		ID:          s.nextID,
		ShortCode:   shortCode,
		OriginalURL: originalURL,
		Username:    username,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.urls[shortCode] = u
	cp := *u
	return &cp
}

func (s *Store) Get(shortCode string) (*URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.urls[shortCode]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (s *Store) Exists(shortCode string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.urls[shortCode]
	return ok
}

func (s *Store) List(username string) []URL {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []URL{}
	for _, u := range s.urls {
		if u.Username == username {
			out = append(out, *u)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

func (s *Store) Delete(shortCode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.urls[shortCode]; !ok {
		return ErrNotFound
	}
	delete(s.urls, shortCode)
	delete(s.clicks, shortCode)
	return nil
}

func (s *Store) RecordClick(shortCode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.urls[shortCode]
	if !ok {
		return
	}
	now := time.Now().UTC()
	u.ClickCount++
	u.UpdatedAt = now
	clicks := append([]time.Time{now}, s.clicks[shortCode]...)
	if len(clicks) > recentClicksCap {
		clicks = clicks[:recentClicksCap]
	}
	s.clicks[shortCode] = clicks
}

func (s *Store) RecentClicks(shortCode string, limit int) []ClickRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ts := s.clicks[shortCode]
	if limit > 0 && len(ts) > limit {
		ts = ts[:limit]
	}
	out := make([]ClickRecord, len(ts))
	for i, t := range ts {
		out[i] = ClickRecord{ClickedAt: t}
	}
	return out
}

func (s *Store) Top(limit int) []URL {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]URL, 0, len(s.urls))
	for _, u := range s.urls {
		out = append(out, *u)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ClickCount != out[j].ClickCount {
			return out[i].ClickCount > out[j].ClickCount
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (s *Store) UserURLs(username string) []URL {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []URL{}
	for _, u := range s.urls {
		if u.Username == username {
			out = append(out, *u)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ClickCount > out[j].ClickCount
	})
	return out
}

func (s *Store) UpdateMetadata(shortCode, title, favicon string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.urls[shortCode]
	if !ok {
		return
	}
	if title != "" {
		u.Title = title
	}
	if favicon != "" {
		u.FaviconURL = favicon
	}
	u.UpdatedAt = time.Now().UTC()
}
