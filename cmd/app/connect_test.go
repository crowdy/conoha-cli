package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	clerrors "github.com/crowdy/conoha-cli/internal/errors"
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

func TestDeployCmdHasComposeFileFlag(t *testing.T) {
	f := deployCmd.Flags().Lookup("compose-file")
	if f == nil {
		t.Fatal("expected compose-file flag to be registered on deployCmd")
	}
	if f.Shorthand != "f" {
		t.Errorf("compose-file shorthand: got %q, want %q", f.Shorthand, "f")
	}
}

func TestDetectComposeFile(t *testing.T) {
	// Empty dir — no compose file
	dir := t.TempDir()
	_, err := detectComposeFile(dir)
	if err == nil {
		t.Error("expected error for empty dir")
	}

	// Each valid compose file name
	names := []string{
		"conoha-docker-compose.yml",
		"conoha-docker-compose.yaml",
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}
	for _, name := range names {
		d := t.TempDir()
		if err := os.WriteFile(filepath.Join(d, name), []byte("version: '3'"), 0644); err != nil {
			t.Fatal(err)
		}
		got, err := detectComposeFile(d)
		if err != nil {
			t.Errorf("expected no error for %s, got %v", name, err)
		}
		if got != name {
			t.Errorf("expected %q, got %q", name, got)
		}
	}
}

func TestResolveComposeFile(t *testing.T) {
	// Explicit file that exists
	dir := t.TempDir()
	f := filepath.Join(dir, "custom-compose.yml")
	if err := os.WriteFile(f, []byte("version: '3'"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := resolveComposeFile(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != f {
		t.Errorf("expected %q, got %q", f, got)
	}

	// Explicit file that does not exist
	_, err = resolveComposeFile("/nonexistent/compose.yml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
	if err != nil && !strings.Contains(err.Error(), "compose file not found") {
		t.Errorf("expected error to contain %q, got %q", "compose file not found", err.Error())
	}
	// Should be a ValidationError (exit code 4)
	if _, ok := err.(*clerrors.ValidationError); !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestValidateComposeFilePath(t *testing.T) {
	valid := []string{
		"docker-compose.yml",
		"conoha-docker-compose.yaml",
		"compose.yml",
		"path/to/compose.yml",
		"my_app-compose.yml",
	}
	for _, p := range valid {
		if err := validateComposeFilePath(p); err != nil {
			t.Errorf("expected %q to be valid, got error: %v", p, err)
		}
	}

	invalid := []string{
		"it's-compose.yml",
		"compose;rm -rf.yml",
		"$(whoami).yml",
		"file`cmd`.yml",
		"a b.yml",
		"compose|pipe.yml",
		"compose&bg.yml",
	}
	for _, p := range invalid {
		err := validateComposeFilePath(p)
		if err == nil {
			t.Errorf("expected %q to be invalid", p)
		}
		if _, ok := err.(*clerrors.ValidationError); !ok {
			t.Errorf("expected ValidationError for %q, got %T", p, err)
		}
	}
}

func TestDetectComposeFilePriority(t *testing.T) {
	// conoha-docker-compose.yml takes priority over docker-compose.yml
	dir := t.TempDir()
	for _, name := range []string{"conoha-docker-compose.yml", "docker-compose.yml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("version: '3'"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := detectComposeFile(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "conoha-docker-compose.yml" {
		t.Errorf("expected conoha-docker-compose.yml to take priority, got %q", got)
	}
}
