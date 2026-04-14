package prompt

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/term"

	"github.com/crowdy/conoha-cli/internal/config"
)

func TestSelect_NoInput_WithoutHint_IncludesLabel(t *testing.T) {
	// Without a hint, the error includes the selection label so the user knows
	// which selection failed.
	t.Setenv(config.EnvNoInput, "1")
	items := []SelectItem{{Label: "test", Value: "v1"}}
	_, err := Select("Select flavor", items)
	if err == nil {
		t.Fatal("expected error when --no-input is set")
	}
	if !strings.Contains(err.Error(), "Select flavor") {
		t.Errorf("error should include label, got: %q", err.Error())
	}
}

func TestSelect_NoInput_WithHint_ShowsFlag(t *testing.T) {
	// With a hint, the error names the specific flag the user should provide.
	t.Setenv(config.EnvNoInput, "1")
	items := []SelectItem{{Label: "option A", Value: "a"}}
	_, err := Select("Select security group", items, "use --security-group <name> to specify (repeatable)")
	if err == nil {
		t.Fatal("expected error under --no-input")
	}
	if !strings.Contains(err.Error(), "--security-group") {
		t.Errorf("error should include flag hint, got: %q", err.Error())
	}
}

func TestSelect_NonTTY_IncludesLabel(t *testing.T) {
	t.Setenv(config.EnvNoInput, "")
	if term.IsTerminal(int(os.Stdin.Fd())) {
		t.Skip("stdin is a terminal, skipping non-TTY test")
	}
	items := []SelectItem{{Label: "test", Value: "v1"}}
	_, err := Select("pick", items)
	if err == nil {
		t.Fatal("expected error when stdin is not a TTY")
	}
	if !strings.Contains(err.Error(), "pick") {
		t.Errorf("error should include label, got: %q", err.Error())
	}
}
