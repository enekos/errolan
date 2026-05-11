// Package ratelimit is a tiny in-memory token bucket keyed by arbitrary strings.
//
// Each bucket is independent: per-IP global, per-IP-per-route-family, per-email
// auth attempts, etc. Cleanup runs lazily on Allow when the map grows past a
// threshold, avoiding a background goroutine.
package ratelimit

import (
	"sync"
	"time"
)

type bucket struct {
	tokens   float64
	last     time.Time
}

type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens per second
	burst    float64
}

// New returns a limiter that refills `rate` tokens per second up to `burst`.
func New(rate float64, burst int) *Limiter {
	return &Limiter{
		buckets: make(map[string]*bucket),
		rate:    rate,
		burst:   float64(burst),
	}
}

// Allow consumes one token from the bucket identified by key. Returns true if
// allowed; false if the caller should be rejected (e.g. 429).
func (l *Limiter) Allow(key string) bool {
	return l.AllowN(key, 1)
}

func (l *Limiter) AllowN(key string, n float64) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens < n {
		l.maybeGC(now)
		return false
	}
	b.tokens -= n
	l.maybeGC(now)
	return true
}

// maybeGC drops idle buckets. Called under lock.
func (l *Limiter) maybeGC(now time.Time) {
	if len(l.buckets) < 4096 {
		return
	}
	cutoff := 10 * time.Minute
	for k, b := range l.buckets {
		if now.Sub(b.last) > cutoff {
			delete(l.buckets, k)
		}
	}
}
