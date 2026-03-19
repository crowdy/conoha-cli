package cmdutil

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestWaitForImmediateSuccess(t *testing.T) {
	err := WaitFor(WaitConfig{
		Interval: 10 * time.Millisecond,
		Timeout:  100 * time.Millisecond,
		Resource: "test",
	}, func() (bool, string, error) {
		return true, "done", nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestWaitForTimeout(t *testing.T) {
	err := WaitFor(WaitConfig{
		Interval: 10 * time.Millisecond,
		Timeout:  50 * time.Millisecond,
		Resource: "test-resource",
	}, func() (bool, string, error) {
		return false, "PENDING", nil
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout error, got: %v", err)
	}
}

func TestWaitForError(t *testing.T) {
	want := errors.New("something broke")
	err := WaitFor(WaitConfig{
		Interval: 10 * time.Millisecond,
		Timeout:  100 * time.Millisecond,
		Resource: "test",
	}, func() (bool, string, error) {
		return false, "", want
	})
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func TestWaitForEventualSuccess(t *testing.T) {
	calls := 0
	err := WaitFor(WaitConfig{
		Interval: 10 * time.Millisecond,
		Timeout:  1 * time.Second,
		Resource: "test",
	}, func() (bool, string, error) {
		calls++
		if calls >= 3 {
			return true, "ACTIVE", nil
		}
		return false, "BUILD", nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if calls < 3 {
		t.Fatalf("expected at least 3 calls, got %d", calls)
	}
}
