package app

import (
	"strings"
	"testing"
)

func TestDetermineSlotID_Timestamp(t *testing.T) {
	id, err := determineSlotID(".", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 14 {
		t.Errorf("expected 14-char timestamp, got %q (%d chars)", id, len(id))
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			t.Errorf("non-digit %q in %q", r, id)
		}
	}
}

func TestSuffixIfTaken(t *testing.T) {
	taken := map[string]bool{"abc1234": true, "abc1234-2": true}
	got := suffixIfTaken("abc1234", func(s string) bool { return taken[s] })
	if got != "abc1234-3" {
		t.Errorf("got %q, want abc1234-3", got)
	}
	got = suffixIfTaken("fresh", func(s string) bool { return false })
	if got != "fresh" {
		t.Errorf("got %q, want fresh", got)
	}
}

func TestDetermineSlotID_GitShortSHA(t *testing.T) {
	// This test runs inside our own repo, so git IS available. If not, skip.
	id, err := determineSlotID(".", true)
	if err != nil {
		t.Skipf("git not available in test env: %v", err)
	}
	if len(id) != 7 {
		t.Errorf("expected 7-char short SHA, got %q", id)
	}
	if strings.ContainsAny(id, " \n\r") {
		t.Errorf("whitespace in %q", id)
	}
}
