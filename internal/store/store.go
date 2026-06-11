package store

import (
	"sync"

	"github.com/SatarYaghoubi/sentinel/internal/scanner"
)

// Record pairs a scan result with the original filename it came from.
type Record struct {
	ID       string         `json:"id"`
	Filename string         `json:"filename"`
	Result   scanner.Result `json:"result"`
}

// Store is a thread-safe in-memory collection of scan records. It stands in for
// a database so the service runs without external dependencies.
type Store struct {
	mu      sync.RWMutex
	records map[string]Record
}

// New returns an empty Store.
func New() *Store {
	return &Store{records: make(map[string]Record)}
}

// Save inserts or replaces a record.
func (s *Store) Save(r Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[r.ID] = r
}

// Get returns the record for an ID and whether it was found.
func (s *Store) Get(id string) (Record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.records[id]
	return r, ok
}

// List returns every stored record.
func (s *Store) List() []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Record, 0, len(s.records))
	for _, r := range s.records {
		out = append(out, r)
	}
	return out
}

// Count returns the number of stored records.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}
