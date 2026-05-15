package lockout

import (
	"testing"
	"time"
)

func TestAllowsBeforeThreshold(t *testing.T) {
	tr := New(3, time.Minute, time.Minute)
	if !tr.Allowed("a") {
		t.Fatal("fresh key should be allowed")
	}
	for i := 0; i < 2; i++ {
		if locked := tr.Failure("a"); locked {
			t.Fatalf("failure %d should not lock yet", i+1)
		}
	}
	if !tr.Allowed("a") {
		t.Fatal("still under threshold; should be allowed")
	}
}

func TestLocksAtThreshold(t *testing.T) {
	tr := New(3, time.Minute, time.Minute)
	for i := 0; i < 3; i++ {
		tr.Failure("a")
	}
	if tr.Allowed("a") {
		t.Fatal("expected lock after threshold failures")
	}
}

func TestSuccessClears(t *testing.T) {
	tr := New(3, time.Minute, time.Minute)
	tr.Failure("a")
	tr.Failure("a")
	tr.Success("a")
	if !tr.Allowed("a") {
		t.Fatal("success should clear failures")
	}
	// And future failures should start fresh.
	if locked := tr.Failure("a"); locked {
		t.Fatal("post-success first failure should not relock")
	}
}

func TestWindowResetsCounter(t *testing.T) {
	tr := New(3, 10*time.Millisecond, time.Minute)
	tr.Failure("a")
	tr.Failure("a")
	time.Sleep(15 * time.Millisecond)
	// Allowed call triggers the window-elapsed cleanup path.
	if !tr.Allowed("a") {
		t.Fatal("should be allowed after window elapses")
	}
	// Next failure should start a fresh counter — one failure alone doesn't lock.
	if locked := tr.Failure("a"); locked {
		t.Fatal("first failure after window reset should not lock")
	}
}

func TestLockoutExpires(t *testing.T) {
	tr := New(2, time.Minute, 15*time.Millisecond)
	tr.Failure("a")
	tr.Failure("a")
	if tr.Allowed("a") {
		t.Fatal("expected locked")
	}
	time.Sleep(25 * time.Millisecond)
	if !tr.Allowed("a") {
		t.Fatal("lock should have expired")
	}
}

func TestKeysAreIndependent(t *testing.T) {
	tr := New(2, time.Minute, time.Minute)
	tr.Failure("a")
	tr.Failure("a")
	if tr.Allowed("a") {
		t.Fatal("a should be locked")
	}
	if !tr.Allowed("b") {
		t.Fatal("b should be unaffected")
	}
}
