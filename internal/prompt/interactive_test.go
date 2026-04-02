package prompt

import (
	"testing"
)

func TestIsInteractive_NoInput(t *testing.T) {
	t.Setenv("CONOHA_NO_INPUT", "1")
	if IsInteractive() {
		t.Error("expected non-interactive when CONOHA_NO_INPUT=1")
	}
}

func TestIsInteractive_NoTTY(t *testing.T) {
	// In test environment, stdin is not a TTY
	if IsInteractive() {
		t.Error("expected non-interactive in test environment (no TTY)")
	}
}
