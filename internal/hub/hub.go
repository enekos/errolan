// Package hub is an in-memory fan-out for SSE thread updates.
//
// Subscribers receive opaque event strings; the API publishes a "ping" event
// whenever a thread mutates (new/edit/delete/vote/pin/lock).
package hub

import (
	"sync"
)

type Hub struct {
	mu          sync.Mutex
	subscribers map[int64]map[chan string]struct{}
}

func New() *Hub {
	return &Hub{subscribers: make(map[int64]map[chan string]struct{})}
}

func (h *Hub) Subscribe(threadID int64) (<-chan string, func()) {
	ch := make(chan string, 8)
	h.mu.Lock()
	subs, ok := h.subscribers[threadID]
	if !ok {
		subs = make(map[chan string]struct{})
		h.subscribers[threadID] = subs
	}
	subs[ch] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		if subs, ok := h.subscribers[threadID]; ok {
			delete(subs, ch)
			if len(subs) == 0 {
				delete(h.subscribers, threadID)
			}
		}
		close(ch)
		h.mu.Unlock()
	}
	return ch, cancel
}

func (h *Hub) Publish(threadID int64, event string) {
	h.mu.Lock()
	subs := h.subscribers[threadID]
	targets := make([]chan string, 0, len(subs))
	for ch := range subs {
		targets = append(targets, ch)
	}
	h.mu.Unlock()
	for _, ch := range targets {
		select {
		case ch <- event:
		default:
			// drop on full; subscriber will catch up on next refresh
		}
	}
}
