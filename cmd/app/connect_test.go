package app

import (
	"os"
	"path/filepath"
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

func TestHasComposeFile(t *testing.T) {
	// Empty dir — no compose file
	dir := t.TempDir()
	if hasComposeFile(dir) {
		t.Error("expected false for empty dir")
	}

	// Each valid compose file name
	names := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
	for _, name := range names {
		d := t.TempDir()
		if err := os.WriteFile(filepath.Join(d, name), []byte("version: '3'"), 0644); err != nil {
			t.Fatal(err)
		}
		if !hasComposeFile(d) {
			t.Errorf("expected true for %s", name)
		}
	}
}
