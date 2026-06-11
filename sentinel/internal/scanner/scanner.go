package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// Violation is a single rule match found in a scanned file.
type Violation struct {
	RuleID      string   `json:"rule_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Severity    Severity `json:"severity"`
}

// Result is the outcome of scanning one file.
type Result struct {
	SHA256     string      `json:"sha256"`
	SizeBytes  int         `json:"size_bytes"`
	Clean      bool        `json:"clean"`
	Violations []Violation `json:"violations"`
	ScannedAt  time.Time   `json:"scanned_at"`
	DurationMS int64       `json:"duration_ms"`
}

// job is an internal unit of work pushed onto the worker pool.
type job struct {
	data   []byte
	result chan Result
}

// Scanner inspects file content against a rule set using a fixed pool of
// worker goroutines. It is safe for concurrent use by multiple goroutines.
type Scanner struct {
	rules   []Rule
	jobs    chan job
	wg      sync.WaitGroup
	once    sync.Once
	closing chan struct{}
}

// New creates a Scanner backed by workers goroutines. If workers <= 0 it
// defaults to a single worker.
func New(rules []Rule, workers int) *Scanner {
	if workers <= 0 {
		workers = 1
	}
	s := &Scanner{
		rules:   rules,
		jobs:    make(chan job),
		closing: make(chan struct{}),
	}
	for i := 0; i < workers; i++ {
		s.wg.Add(1)
		go s.worker()
	}
	return s
}

// worker pulls jobs off the queue until the queue is closed.
func (s *Scanner) worker() {
	defer s.wg.Done()
	for j := range s.jobs {
		j.result <- s.scan(j.data)
	}
}

// scan performs the actual signature matching on a single payload.
func (s *Scanner) scan(data []byte) Result {
	start := time.Now()

	sum := sha256.Sum256(data)
	violations := make([]Violation, 0)

	for _, rule := range s.rules {
		if rule.Match(data) {
			violations = append(violations, Violation{
				RuleID:      rule.ID,
				Name:        rule.Name,
				Description: rule.Description,
				Severity:    rule.Severity,
			})
		}
	}

	return Result{
		SHA256:     hex.EncodeToString(sum[:]),
		SizeBytes:  len(data),
		Clean:      len(violations) == 0,
		Violations: violations,
		ScannedAt:  start.UTC(),
		DurationMS: time.Since(start).Milliseconds(),
	}
}

// Scan submits data to the worker pool and waits for the result. It respects
// context cancellation so slow or abandoned requests do not block forever.
func (s *Scanner) Scan(ctx context.Context, data []byte) (Result, error) {
	res := make(chan Result, 1)

	select {
	case s.jobs <- job{data: data, result: res}:
	case <-ctx.Done():
		return Result{}, ctx.Err()
	case <-s.closing:
		return Result{}, context.Canceled
	}

	select {
	case r := <-res:
		return r, nil
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}
}

// Shutdown stops accepting new work and waits for in-flight scans to finish.
func (s *Scanner) Shutdown() {
	s.once.Do(func() {
		close(s.closing)
		close(s.jobs)
	})
	s.wg.Wait()
}
