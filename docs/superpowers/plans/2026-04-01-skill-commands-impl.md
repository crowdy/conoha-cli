# Skill Commands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `conoha skill install|update|remove` commands to manage the conoha-cli-skill for Claude Code.

**Architecture:** Single file `cmd/skill/skill.go` with parent command + 3 subcommands. Uses `os/exec` for git operations, `os` for filesystem checks, `prompt.Confirm` for destructive remove. Registered in `cmd/root.go`.

**Tech Stack:** Go, cobra, os/exec, internal/errors, internal/prompt

---

### Task 1: Create `cmd/skill/skill.go` with install command + test

**Files:**
- Create: `cmd/skill/skill.go`
- Create: `cmd/skill/skill_test.go`

- [ ] **Step 1: Write the test file with install tests**

Create `cmd/skill/skill_test.go`:

```go
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
			t.Fatalf("install failed: %v", err)
		}

		skillDir := filepath.Join(dir, skillName)
		if _, err := os.Stat(filepath.Join(skillDir, ".git")); os.IsNotExist(err) {
			t.Error("expected .git directory after install")
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails (skill.go doesn't exist yet)**

Run: `go test ./cmd/skill/ -v -run TestInstallCmd 2>&1`
Expected: compilation error — package does not exist

- [ ] **Step 3: Create `cmd/skill/skill.go` with install command**

Create `cmd/skill/skill.go`:

```go
package skill

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	cerrors "github.com/crowdy/conoha-cli/internal/errors"
	"github.com/crowdy/conoha-cli/internal/prompt"
)

const (
	skillRepo = "https://github.com/crowdy/conoha-cli-skill.git"
	skillName = "conoha-cli-skill"
)

// Cmd is the parent command for skill management.
var Cmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage Claude Code skills",
}

func init() {
	Cmd.AddCommand(installCmd)
}

func defaultSkillBase() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "skills")
}

func runInstall(baseDir string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return &cerrors.ValidationError{Message: "git is required to install skills"}
	}

	skillDir := filepath.Join(baseDir, skillName)
	if _, err := os.Stat(skillDir); err == nil {
		return &cerrors.ValidationError{Message: "already installed, use 'conoha skill update'"}
	}

	cmd := exec.Command("git", "clone", skillRepo, skillDir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return &cerrors.NetworkError{Err: fmt.Errorf("git clone failed: %w", err)}
	}

	fmt.Fprintln(os.Stderr, "Installed conoha-cli-skill successfully.")
	return nil
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install conoha-cli-skill for Claude Code",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInstall(defaultSkillBase())
	},
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/skill/ -v -run TestInstallCmd 2>&1`
Expected: all 3 subtests PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/skill/skill.go cmd/skill/skill_test.go
git commit -m "Add skill install command with tests (#47)"
```

---

### Task 2: Add update command + test

**Files:**
- Modify: `cmd/skill/skill.go`
- Modify: `cmd/skill/skill_test.go`

- [ ] **Step 1: Add update tests to `cmd/skill/skill_test.go`**

Append to `cmd/skill/skill_test.go`:

```go
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
			t.Fatalf("install failed: %v", err)
		}

		err := runUpdate(dir)
		if err != nil {
			t.Fatalf("update failed: %v", err)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/skill/ -v -run TestUpdateCmd 2>&1`
Expected: compilation error — `runUpdate` not defined

- [ ] **Step 3: Add update command to `cmd/skill/skill.go`**

Add to `init()`:

```go
Cmd.AddCommand(updateCmd)
```

Add functions:

```go
func runUpdate(baseDir string) error {
	skillDir := filepath.Join(baseDir, skillName)
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return &cerrors.ValidationError{Message: "not installed, use 'conoha skill install'"}
	}

	gitDir := filepath.Join(skillDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return &cerrors.ValidationError{Message: "not a git repository, remove and reinstall"}
	}

	cmd := exec.Command("git", "-C", skillDir, "pull")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return &cerrors.NetworkError{Err: fmt.Errorf("git pull failed: %w", err)}
	}

	fmt.Fprintln(os.Stderr, "Updated conoha-cli-skill successfully.")
	return nil
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update conoha-cli-skill to latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate(defaultSkillBase())
	},
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/skill/ -v -run TestUpdateCmd 2>&1`
Expected: all 3 subtests PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/skill/skill.go cmd/skill/skill_test.go
git commit -m "Add skill update command with tests (#47)"
```

---

### Task 3: Add remove command + test

**Files:**
- Modify: `cmd/skill/skill.go`
- Modify: `cmd/skill/skill_test.go`

- [ ] **Step 1: Add remove tests to `cmd/skill/skill_test.go`**

Append to `cmd/skill/skill_test.go`:

```go
func TestRemoveCmd(t *testing.T) {
	t.Run("fails when not installed", func(t *testing.T) {
		dir := t.TempDir()

		err := runRemove(dir)
		if err == nil {
			t.Fatal("expected error when not installed")
		}
		if err.Error() != "validation error: not installed" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("removes successfully with --yes", func(t *testing.T) {
		dir := t.TempDir()
		skillDir := filepath.Join(dir, skillName)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("CONOHA_YES", "1")

		err := runRemove(dir)
		if err != nil {
			t.Fatalf("remove failed: %v", err)
		}

		if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
			t.Error("expected skill directory to be removed")
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/skill/ -v -run TestRemoveCmd 2>&1`
Expected: compilation error — `runRemove` not defined

- [ ] **Step 3: Add remove command to `cmd/skill/skill.go`**

Add to `init()`:

```go
Cmd.AddCommand(removeCmd)
```

Add functions:

```go
func runRemove(baseDir string) error {
	skillDir := filepath.Join(baseDir, skillName)
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return &cerrors.ValidationError{Message: "not installed"}
	}

	ok, err := prompt.Confirm("Remove conoha-cli-skill?")
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "Cancelled.")
		return nil
	}

	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("failed to remove: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Removed conoha-cli-skill successfully.")
	return nil
}

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove conoha-cli-skill",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRemove(defaultSkillBase())
	},
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/skill/ -v -run TestRemoveCmd 2>&1`
Expected: all 2 subtests PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/skill/skill.go cmd/skill/skill_test.go
git commit -m "Add skill remove command with tests (#47)"
```

---

### Task 4: Register in root.go + full test + lint

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: Add skill import and registration to `cmd/root.go`**

Add import:

```go
"github.com/crowdy/conoha-cli/cmd/skill"
```

Add after `rootCmd.AddCommand(app.Cmd)` (line 89):

```go
rootCmd.AddCommand(skill.Cmd)
```

- [ ] **Step 2: Run full test suite**

Run: `go test ./... -v 2>&1`
Expected: all tests PASS

- [ ] **Step 3: Run linter**

Run: `golangci-lint run ./... 2>&1`
Expected: 0 issues

- [ ] **Step 4: Commit**

```bash
git add cmd/root.go
git commit -m "Register skill command group in root (#47)"
```

---

### Task 5: Create PR

- [ ] **Step 1: Create branch, push, and open PR**

```bash
git checkout -b feature/skill-commands
git push -u origin feature/skill-commands
gh pr create --title "Add skill install/update/remove commands" --body "Closes #47"
```
