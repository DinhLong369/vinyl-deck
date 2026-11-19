// Package store keeps the record shelf in a JSON file guarded by a mutex.
package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type Record struct {
	ID        string    `json:"id"`
	VideoID   string    `json:"videoId"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Thumbnail string    `json:"thumbnail"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"createdAt"`
}

var (
	ErrNotFound  = errors.New("record not found")
	ErrDuplicate = errors.New("record already on the shelf")
)

type Store struct {
	mu      sync.RWMutex
	path    string
	records []Record
}

func Open(path string) (*Store, error) {
	s := &Store{path: path}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if len(b) > 0 {
		if err := json.Unmarshal(b, &s.records); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}

func (s *Store) List() []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Record, len(s.records))
	copy(out, s.records)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Position < out[j].Position })
	return out
}

func (s *Store) Add(r Record) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	maxPos := -1
	for _, e := range s.records {
		if e.VideoID == r.VideoID {
			return Record{}, ErrDuplicate
		}
		if e.Position > maxPos {
			maxPos = e.Position
		}
	}
	r.Position = maxPos + 1
	r.CreatedAt = time.Now().UTC()
	s.records = append(s.records, r)
	return r, s.persist()
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, e := range s.records {
		if e.ID == id {
			s.records = append(s.records[:i], s.records[i+1:]...)
			return s.persist()
		}
	}
	return ErrNotFound
}

// Reorder sets positions following the given id order; unknown ids are ignored.
func (s *Store) Reorder(ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pos := make(map[string]int, len(ids))
	for i, id := range ids {
		pos[id] = i
	}
	next := len(ids)
	for i := range s.records {
		if p, ok := pos[s.records[i].ID]; ok {
			s.records[i].Position = p
		} else {
			s.records[i].Position = next
			next++
		}
	}
	return s.persist()
}

func (s *Store) persist() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.records, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
