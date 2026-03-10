package prompt

import (
	"testing"

	"golang.org/x/term"
	"os"

	"github.com/crowdy/conoha-cli/internal/config"
)

func TestSelect_NoInput(t *testing.T) {
	t.Setenv(config.EnvNoInput, "1")
	items := []SelectItem{{Label: "test", Value: "v1"}}
	_, err := Select("pick", items)
	if err == nil {
		t.Fatal("expected error when --no-input is set")
	}
	if got := err.Error(); got != "selection required but --no-input is set" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestSelect_NonTTY(t *testing.T) {
	t.Setenv(config.EnvNoInput, "")
	if term.IsTerminal(int(os.Stdin.Fd())) {
		t.Skip("stdin is a terminal, skipping non-TTY test")
	}
	items := []SelectItem{{Label: "test", Value: "v1"}}
	_, err := Select("pick", items)
	if err == nil {
		t.Fatal("expected error when stdin is not a TTY")
	}
	expected := "interactive selection requires a TTY; use flags to specify values"
	if got := err.Error(); got != expected {
		t.Errorf("unexpected error: %q, want %q", got, expected)
	}
}
