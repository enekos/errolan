package ratelimit

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestAllowsBurstThenRejects(t *testing.T) {
	// 1 token/sec, burst 3. Three back-to-back allows; the fourth fails.
	l := New(1, 3)
	for i := 0; i < 3; i++ {
		if !l.Allow("k") {
			t.Fatalf("burst call %d should be allowed", i+1)
		}
	}
	if l.Allow("k") {
		t.Fatal("4th call should be rejected (burst exhausted)")
	}
}

func TestRefills(t *testing.T) {
	// 100 tokens/sec → ~10ms per token, burst 1.
	l := New(100, 1)
	if !l.Allow("k") {
		t.Fatal("first call should be allowed")
	}
	if l.Allow("k") {
		t.Fatal("immediate second call should be denied")
	}
	time.Sleep(20 * time.Millisecond)
	if !l.Allow("k") {
		t.Fatal("token should have refilled by now")
	}
}

func TestKeysAreIndependent(t *testing.T) {
	l := New(1, 1)
	if !l.Allow("a") {
		t.Fatal("first allow on a should succeed")
	}
	if !l.Allow("b") {
		t.Fatal("first allow on b should succeed independently")
	}
	if l.Allow("a") {
		t.Fatal("second allow on a should be denied")
	}
}

func TestAllowNRespectsCount(t *testing.T) {
	l := New(1, 5)
	if !l.AllowN("k", 3) {
		t.Fatal("AllowN(3) on burst 5 should succeed")
	}
	if !l.AllowN("k", 2) {
		t.Fatal("AllowN(2) should drain the remaining tokens")
	}
	if l.AllowN("k", 1) {
		t.Fatal("no tokens left; should fail")
	}
}

func TestConcurrentSafe(t *testing.T) {
	l := New(1000, 1000)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "k" + strconv.Itoa(i%5)
			for j := 0; j < 100; j++ {
				l.Allow(key)
			}
		}(i)
	}
	wg.Wait()
}
