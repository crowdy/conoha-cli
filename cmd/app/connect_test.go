package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestAddAppFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	addAppFlags(cmd)

	flags := []string{"app-name", "user", "port", "identity"}
	for _, name := range flags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag %q to be registered", name)
		}
	}

	// Check shorthand
	shorthands := map[string]string{"user": "l", "port": "p", "identity": "i"}
	for name, short := range shorthands {
		f := cmd.Flags().Lookup(name)
		if f.Shorthand != short {
			t.Errorf("flag %q shorthand: got %q, want %q", name, f.Shorthand, short)
		}
	}
}

func newAppCmdForTest() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	addAppFlags(cmd)
	return cmd
}

func TestResolveAppName_FlagWins(t *testing.T) {
	// Even with conoha.yml present in cwd, an explicit --app-name beats it.
	t.Chdir(t.TempDir())
	if err := os.WriteFile("conoha.yml", []byte("name: from-yaml\nhosts:\n  - example.local\nweb:\n  service: web\n  port: 80\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := newAppCmdForTest()
	if err := cmd.Flags().Set("app-name", "from-flag"); err != nil {
		t.Fatal(err)
	}
	got, err := resolveAppName(cmd)
	if err != nil {
		t.Fatalf("resolveAppName: %v", err)
	}
	if got != "from-flag" {
		t.Errorf("got %q, want %q", got, "from-flag")
	}
}

// Regression for #169: status (and any other command going through
// connectToApp) must read the project name from conoha.yml when --app-name
// isn't given and one exists in cwd. Before this fallback, --no-input runs
// errored with "validation error on App name: input required but --no-input
// is set" even from inside the project dir, while init/deploy/rollback
// happily read the same file.
//
// CONOHA_NO_INPUT=1 is set deliberately: the pre-fix code would fall through
// to prompt.String("App name"), which under --no-input returns the standard
// ValidationError. So this exact assertion (no error, name == "from-yaml")
// would have FAILED on the buggy code — making this a real regression
// guard, not a tautology.
func TestResolveAppName_FallsBackToProjectFile(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("conoha.yml", []byte("name: from-yaml\nhosts:\n  - example.local\nweb:\n  service: web\n  port: 80\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONOHA_NO_INPUT", "1")
	got, err := resolveAppName(newAppCmdForTest())
	if err != nil {
		t.Fatalf("resolveAppName: %v", err)
	}
	if got != "from-yaml" {
		t.Errorf("got %q, want %q", got, "from-yaml")
	}
}

func TestResolveAppName_NoFileNoInputErrors(t *testing.T) {
	// No conoha.yml in cwd, --app-name empty, --no-input set: prompt must
	// surface the standard ValidationError rather than silently returning "".
	t.Chdir(t.TempDir())
	t.Setenv("CONOHA_NO_INPUT", "1")
	_, err := resolveAppName(newAppCmdForTest())
	if err == nil {
		t.Fatal("want error under --no-input with no conoha.yml; got nil")
	}
	if !strings.Contains(err.Error(), "input required but --no-input is set") {
		t.Errorf("want 'input required' validation error, got: %v", err)
	}
}

// A conoha.yml that parses but fails Validate() for a NON-name reason
// (here: missing hosts) must still fall through to prompt — we don't want
// to smuggle in pf.Name when the rest of the project is malformed and a
// later code path would error with a vaguer message. Picking a non-name
// validation failure (rather than name: "") is what makes this a real
// guard for the Validate()-required gate: the pre-fix code went to the
// prompt unconditionally regardless of pf state, so the only way to
// distinguish working-as-intended fallthrough from accidental fallthrough
// is to show that a *valid name* still doesn't leak through when the
// project as a whole isn't valid.
func TestResolveAppName_InvalidProjectFileFallsThroughToPrompt(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "conoha.yml"), []byte("name: looks-valid\nweb:\n  service: web\n  port: 80\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONOHA_NO_INPUT", "1")
	_, err := resolveAppName(newAppCmdForTest())
	if err == nil {
		t.Fatal("want error when project file is invalid + --no-input")
	}
	if !strings.Contains(err.Error(), "input required") {
		t.Errorf("expected fallthrough to prompt (no project leaked through); got: %v", err)
	}
}
