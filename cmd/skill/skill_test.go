package skill

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInstallCmd(t *testing.T) {
	t.Run("fails when git not found", func(t *testing.T) {
		// Override PATH to hide git
		t.Setenv("PATH", "/nonexistent")
		dir := t.TempDir()

		err := runInstall(dir)
		if err == nil {
			t.Fatal("expected error when git not found")
		}
		if err.Error() != "validation error: git is required to install skills" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails when already installed", func(t *testing.T) {
		dir := t.TempDir()
		skillDir := filepath.Join(dir, skillName)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatal(err)
		}

		err := runInstall(dir)
		if err == nil {
			t.Fatal("expected error when already installed")
		}
		if err.Error() != "validation error: already installed, use 'conoha skill update'" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("clones successfully", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git not available")
		}
		dir := t.TempDir()

		err := runInstall(dir)
		if err != nil {
			// Skip if remote repo is not accessible (e.g., not yet created)
			t.Skipf("skipping: remote repo not accessible: %v", err)
		}

		skillDir := filepath.Join(dir, skillName)
		if _, err := os.Stat(filepath.Join(skillDir, ".git")); os.IsNotExist(err) {
			t.Error("expected .git directory after install")
		}
	})
}

func TestUpdateCmd(t *testing.T) {
	t.Run("fails when not installed", func(t *testing.T) {
		dir := t.TempDir()

		err := runUpdate(dir)
		if err == nil {
			t.Fatal("expected error when not installed")
		}
		if err.Error() != "validation error: not installed, use 'conoha skill install'" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails when not a git repo", func(t *testing.T) {
		dir := t.TempDir()
		skillDir := filepath.Join(dir, skillName)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatal(err)
		}

		err := runUpdate(dir)
		if err == nil {
			t.Fatal("expected error when not a git repo")
		}
		if err.Error() != "validation error: not a git repository, remove and reinstall" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("pulls successfully", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git not available")
		}
		dir := t.TempDir()

		// Install first
		if err := runInstall(dir); err != nil {
			t.Skipf("install failed (remote repo may not exist): %v", err)
		}

		err := runUpdate(dir)
		if err != nil {
			t.Fatalf("update failed: %v", err)
		}
	})
}
