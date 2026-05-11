// Package lockout tracks failed login attempts and temporarily blocks accounts
// that exceed the threshold, defeating online password guessing.
package lockout

import (
	"sync"
	"time"
)

type entry struct {
	failures   int
	lockedUntil time.Time
	lastSeen   time.Time
}

type Tracker struct {
	mu        sync.Mutex
	entries   map[string]*entry
	threshold int
	window    time.Duration
	lock      time.Duration
}

// New returns a tracker that locks a key for `lock` after `threshold` failures
// observed within `window`.
func New(threshold int, window, lock time.Duration) *Tracker {
	return &Tracker{
		entries:   make(map[string]*entry),
		threshold: threshold,
		window:    window,
		lock:      lock,
	}
}

// Allowed reports whether the key is currently allowed to attempt.
func (t *Tracker) Allowed(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	e := t.entries[key]
	if e == nil {
		return true
	}
	now := time.Now()
	if !e.lockedUntil.IsZero() && now.Before(e.lockedUntil) {
		return false
	}
	// Reset failures if the window has elapsed since the last seen failure.
	if now.Sub(e.lastSeen) > t.window {
		delete(t.entries, key)
	}
	return true
}

// Failure records a failed attempt; returns true if the key is now locked.
func (t *Tracker) Failure(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	e := t.entries[key]
	if e == nil || now.Sub(e.lastSeen) > t.window {
		e = &entry{}
		t.entries[key] = e
	}
	e.failures++
	e.lastSeen = now
	if e.failures >= t.threshold {
		e.lockedUntil = now.Add(t.lock)
		return true
	}
	return false
}

// Success clears the counter for a successful attempt.
func (t *Tracker) Success(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, key)
}
