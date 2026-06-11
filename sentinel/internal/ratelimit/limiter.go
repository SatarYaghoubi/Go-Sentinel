package ratelimit

import (
	"sync"
	"time"
)

// bucket is a single token bucket tracking one client.
type bucket struct {
	tokens     float64
	lastRefill time.Time
}

// Limiter is a per-key token-bucket rate limiter. Each key (for example a
// client IP) gets its own bucket that refills at a steady rate. It is safe for
// concurrent use.
type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens added per second
	capacity float64 // maximum tokens a bucket can hold (burst size)
}

// New returns a Limiter that allows `rate` requests per second per key, with
// bursts of up to `burst` requests.
func New(rate float64, burst int) *Limiter {
	if rate <= 0 {
		rate = 1
	}
	if burst <= 0 {
		burst = 1
	}
	return &Limiter{
		buckets:  make(map[string]*bucket),
		rate:     rate,
		capacity: float64(burst),
	}
}

// Allow reports whether a request from the given key may proceed, consuming a
// token if so.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.buckets[key]
	if !ok {
		l.buckets[key] = &bucket{tokens: l.capacity - 1, lastRefill: now}
		return true
	}

	// Refill based on elapsed time since the last request.
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > l.capacity {
		b.tokens = l.capacity
	}
	b.lastRefill = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// Cleanup removes buckets that have been idle longer than maxIdle. Call it
// periodically to bound memory usage under many distinct keys.
func (l *Limiter) Cleanup(maxIdle time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-maxIdle)
	for key, b := range l.buckets {
		if b.lastRefill.Before(cutoff) {
			delete(l.buckets, key)
		}
	}
}
