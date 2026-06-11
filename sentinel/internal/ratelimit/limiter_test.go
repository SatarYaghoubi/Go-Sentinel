package ratelimit

import "testing"

func TestAllowWithinBurst(t *testing.T) {
	l := New(1, 3)
	for i := 0; i < 3; i++ {
		if !l.Allow("client-a") {
			t.Fatalf("request %d should have been allowed within burst", i+1)
		}
	}
	if l.Allow("client-a") {
		t.Error("request beyond burst should have been rejected")
	}
}

func TestSeparateKeysAreIndependent(t *testing.T) {
	l := New(1, 1)
	if !l.Allow("client-a") {
		t.Fatal("first request for client-a should pass")
	}
	if !l.Allow("client-b") {
		t.Fatal("first request for client-b should pass independently")
	}
}
