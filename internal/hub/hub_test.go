package hub

import (
	"sync"
	"testing"
	"time"
)

func TestSubscribeReceives(t *testing.T) {
	h := New()
	events, cancel := h.Subscribe(1)
	defer cancel()
	h.Publish(1, "ping")
	select {
	case got := <-events:
		if got != "ping" {
			t.Fatalf("want ping, got %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestUnrelatedThreadDoesNotReceive(t *testing.T) {
	h := New()
	events, cancel := h.Subscribe(1)
	defer cancel()
	h.Publish(2, "other")
	select {
	case got := <-events:
		t.Fatalf("unexpected event %q on unrelated thread", got)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestCancelClosesChannelAndCleansMap(t *testing.T) {
	h := New()
	events, cancel := h.Subscribe(1)
	cancel()
	if _, ok := <-events; ok {
		t.Fatal("expected channel to be closed")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.subscribers[1]; ok {
		t.Fatal("thread entry should have been removed when last subscriber left")
	}
}

func TestDropsOnFullSubscriber(t *testing.T) {
	h := New()
	_, cancel := h.Subscribe(1)
	defer cancel()
	// Fill the buffered channel (size 8) and a few more — no panic, just drops.
	for i := 0; i < 32; i++ {
		h.Publish(1, "spam")
	}
}

// TestPublishCancelRace stresses the send/cancel ordering. Before the fix
// publish dropped the lock before sending, so cancel could close the channel
// in between and the next send panicked.
func TestPublishCancelRace(t *testing.T) {
	h := New()
	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Publisher goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				h.Publish(1, "evt")
			}
		}
	}()

	// Many short-lived subscribers
	for i := 0; i < 200; i++ {
		events, cancel := h.Subscribe(1)
		// drain a couple events then cancel
		go func() {
			for range events {
			}
		}()
		cancel()
	}

	close(stop)
	wg.Wait()
}
