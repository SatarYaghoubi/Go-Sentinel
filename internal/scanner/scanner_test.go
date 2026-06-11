package scanner

import (
	"context"
	"testing"
)

func TestScanDetectsViolation(t *testing.T) {
	s := New(DefaultRules(), 2)
	defer s.Shutdown()

	payload := []byte("harmless text EICAR-STANDARD-ANTIVIRUS-TEST-FILE more text")
	res, err := s.Scan(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Clean {
		t.Fatal("expected file to be flagged, got clean")
	}
	if len(res.Violations) == 0 {
		t.Fatal("expected at least one violation")
	}
	if res.Violations[0].RuleID != "EICAR-001" {
		t.Errorf("expected EICAR-001, got %s", res.Violations[0].RuleID)
	}
}

func TestScanCleanFile(t *testing.T) {
	s := New(DefaultRules(), 2)
	defer s.Shutdown()

	res, err := s.Scan(context.Background(), []byte("a perfectly ordinary file"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Clean {
		t.Errorf("expected clean file, got %d violations", len(res.Violations))
	}
	if res.SHA256 == "" {
		t.Error("expected a sha256 hash to be computed")
	}
}

func TestScanContextCancelled(t *testing.T) {
	s := New(DefaultRules(), 1)
	defer s.Shutdown()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before submitting

	if _, err := s.Scan(ctx, []byte("data")); err == nil {
		t.Error("expected an error from a cancelled context")
	}
}
