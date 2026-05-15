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
	// Hold the lock for the non-blocking send so a concurrent cancel() can't
	// close the channel between snapshot and send (panic on send-to-closed).
	// Each send is select-with-default, so this never blocks under the lock.
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subscribers[threadID] {
		select {
		case ch <- event:
		default:
			// drop on full; subscriber will catch up on next refresh
		}
	}
}
