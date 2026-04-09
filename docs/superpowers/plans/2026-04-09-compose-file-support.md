# Compose File Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Support `conoha-docker-compose.yml` auto-detection and `--compose-file` flag for `app deploy` (#75).

**Architecture:** Refactor `hasComposeFile()` to `detectComposeFile()` returning the detected filename. Add `--compose-file` / `-f` flag to `addAppFlags()`. Pass `-f <file>` to remote `docker compose` commands. Update post-receive hook in `init.go` to match new detection order.

**Tech Stack:** Go, cobra, `os.Stat`, shell scripting for post-receive hook

---

### File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `cmd/app/deploy.go` | Modify | Replace `hasComposeFile()` with `detectComposeFile()`, update `deployApp()` |
| `cmd/app/connect.go` | Modify | Add `--compose-file` / `-f` flag to `addAppFlags()` |
| `cmd/app/init.go` | Modify | Update post-receive hook template |
| `cmd/app/connect_test.go` | Modify | Replace `TestHasComposeFile` with `TestDetectComposeFile`, add flag test |
| `cmd/app/init_test.go` | Modify | Add assertions for new hook detection order |

---

### Task 1: Add `--compose-file` flag and update `detectComposeFile()`

**Files:**
- Modify: `cmd/app/connect.go:82-87`
- Modify: `cmd/app/deploy.go:83-91`
- Modify: `cmd/app/connect_test.go:32-50`

- [ ] **Step 1: Write tests for `detectComposeFile()` and the new flag**

Replace `TestHasComposeFile` and add `TestDetectComposeFilePriority` in `cmd/app/connect_test.go`. Replace the entire file content with:

```go
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

	flags := []string{"app-name", "user", "port", "identity", "compose-file"}
	for _, name := range flags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag %q to be registered", name)
		}
	}

	// Check shorthand
	shorthands := map[string]string{"user": "l", "port": "p", "identity": "i", "compose-file": "f"}
	for name, short := range shorthands {
		f := cmd.Flags().Lookup(name)
		if f.Shorthand != short {
			t.Errorf("flag %q shorthand: got %q, want %q", name, f.Shorthand, short)
		}
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/app/ -run "TestDetectComposeFile|TestAddAppFlags" -v`
Expected: FAIL — `detectComposeFile` undefined, `compose-file` flag not found

- [ ] **Step 3: Add `--compose-file` flag to `addAppFlags()`**

In `cmd/app/connect.go`, add one line at the end of `addAppFlags()`:

```go
func addAppFlags(cmd *cobra.Command) {
	cmd.Flags().String("app-name", "", "application name")
	cmd.Flags().StringP("user", "l", "root", "SSH user")
	cmd.Flags().StringP("port", "p", "22", "SSH port")
	cmd.Flags().StringP("identity", "i", "", "SSH private key path")
	cmd.Flags().StringP("compose-file", "f", "", "compose file path (auto-detected if not specified)")
}
```

- [ ] **Step 4: Replace `hasComposeFile()` with `detectComposeFile()` in `deploy.go`**

Replace the `hasComposeFile` function (lines 83-91) with:

```go
// composeFileNames lists compose files in detection priority order.
var composeFileNames = []string{
	"conoha-docker-compose.yml",
	"conoha-docker-compose.yaml",
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

// detectComposeFile returns the first compose file found in dir.
func detectComposeFile(dir string) (string, error) {
	for _, name := range composeFileNames {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return name, nil
		}
	}
	return "", fmt.Errorf("no compose file found in current directory (checked conoha-docker-compose.yml/yaml, docker-compose.yml/yaml, compose.yml/yaml)")
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./cmd/app/ -run "TestDetectComposeFile|TestAddAppFlags" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/app/deploy.go cmd/app/connect.go cmd/app/connect_test.go
git commit -m "feat: add detectComposeFile and --compose-file flag (#75)"
```

---

### Task 2: Update `deployApp()` to use compose file selection

**Files:**
- Modify: `cmd/app/deploy.go:33-81`

- [ ] **Step 1: Update `deployApp()` to resolve and use compose file**

Replace the `deployApp` function in `cmd/app/deploy.go` with:

```go
func deployApp(ctx *appContext) error {
	// Resolve compose file
	composeFile, err := resolveComposeFile(ctx.ComposeFile)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Using compose file: %s\n", composeFile)

	// Load .dockerignore
	patterns, err := loadIgnorePatterns(".")
	if err != nil {
		return err
	}

	// Create tar.gz
	fmt.Fprintf(os.Stderr, "Archiving current directory...\n")
	var buf bytes.Buffer
	if err := createTarGz(".", patterns, &buf); err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Uploading to %s (%s)...\n", ctx.Server.Name, ctx.IP)

	// Transfer tar (clean deploy: remove old files first)
	workDir := "/opt/conoha/" + ctx.AppName
	tarCmd := fmt.Sprintf("rm -rf %s && mkdir -p %s && tar xzf - -C %s", workDir, workDir, workDir)
	exitCode, err := internalssh.RunWithStdin(ctx.Client, tarCmd, &buf, os.Stdout, os.Stderr)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("upload exited with code %d", exitCode)
	}

	// Docker compose up (copy .env.server if exists)
	fmt.Fprintf(os.Stderr, "Building and starting containers...\n")
	composeCmd := fmt.Sprintf(
		"ENV_FILE=/opt/conoha/%s.env.server; "+
			"if [ -f \"$ENV_FILE\" ]; then cp \"$ENV_FILE\" %s/.env; fi && "+
			"cd %s && docker compose -f %s up -d --build --remove-orphans && docker compose -f %s ps",
		ctx.AppName, workDir, workDir, composeFile, composeFile)
	exitCode, err = internalssh.RunCommand(ctx.Client, composeCmd, os.Stdout, os.Stderr)
	if err != nil {
		return fmt.Errorf("deploy failed: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("deploy exited with code %d", exitCode)
	}

	fmt.Fprintf(os.Stderr, "Deploy complete.\n")
	return nil
}

// resolveComposeFile returns the compose file to use.
// If explicit is non-empty, it validates that the file exists.
// Otherwise it auto-detects using the priority order.
func resolveComposeFile(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("compose file not found: %s", explicit)
		}
		return explicit, nil
	}
	return detectComposeFile(".")
}
```

- [ ] **Step 2: Add `ComposeFile` field to `appContext` and wire the flag**

In `cmd/app/connect.go`, add the field to `appContext`:

```go
type appContext struct {
	Client      *ssh.Client
	AppName     string
	Server      *model.Server
	IP          string
	User        string
	ComposeFile string
}
```

In `connectToApp()`, read the flag and set the field. Add after the `identity` resolution block (after line 58):

```go
	composeFile, _ := cmd.Flags().GetString("compose-file")
```

And set it in the return struct:

```go
	return &appContext{
		Client:      sshClient,
		AppName:     appName,
		Server:      s,
		IP:          ip,
		User:        user,
		ComposeFile: composeFile,
	}, nil
```

- [ ] **Step 3: Run all app tests**

Run: `go test ./cmd/app/ -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/app/deploy.go cmd/app/connect.go
git commit -m "feat: use compose file selection in deploy flow (#75)"
```

---

### Task 3: Update post-receive hook in `init.go`

**Files:**
- Modify: `cmd/app/init.go:103-112`
- Modify: `cmd/app/init_test.go`

- [ ] **Step 1: Add test assertions for new hook detection order**

Add a new test in `cmd/app/init_test.go`:

```go
func TestGenerateInitScriptComposeDetection(t *testing.T) {
	script := string(generateInitScript("myapp"))

	// Verify conoha-docker-compose.yml is checked first
	checks := []string{
		"conoha-docker-compose.yml",
		"conoha-docker-compose.yaml",
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
		"COMPOSE_FILE=",
		`docker compose -f "$COMPOSE_FILE"`,
	}
	for _, want := range checks {
		if !strings.Contains(script, want) {
			t.Errorf("script missing %q", want)
		}
	}

	// Verify priority order: conoha-docker-compose.yml appears before docker-compose.yml
	conohaIdx := strings.Index(script, "conoha-docker-compose.yml")
	dockerIdx := strings.Index(script, "docker-compose.yml")
	if conohaIdx > dockerIdx {
		t.Error("conoha-docker-compose.yml should be checked before docker-compose.yml")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/app/ -run TestGenerateInitScriptComposeDetection -v`
Expected: FAIL — missing `conoha-docker-compose.yml` and `COMPOSE_FILE=` in script

- [ ] **Step 3: Update the post-receive hook template in `init.go`**

Replace lines 103-111 in `generateInitScript()` (the compose detection block inside the HOOK heredoc) with:

```go
    COMPOSE_FILE=""
    if [ -f conoha-docker-compose.yml ]; then
        COMPOSE_FILE=conoha-docker-compose.yml
    elif [ -f conoha-docker-compose.yaml ]; then
        COMPOSE_FILE=conoha-docker-compose.yaml
    elif [ -f docker-compose.yml ]; then
        COMPOSE_FILE=docker-compose.yml
    elif [ -f docker-compose.yaml ]; then
        COMPOSE_FILE=docker-compose.yaml
    elif [ -f compose.yml ]; then
        COMPOSE_FILE=compose.yml
    elif [ -f compose.yaml ]; then
        COMPOSE_FILE=compose.yaml
    fi

    if [ -n "$COMPOSE_FILE" ]; then
        echo "==> Building and starting containers with $COMPOSE_FILE..."
        docker compose -f "$COMPOSE_FILE" up -d --build --remove-orphans
        echo "==> Deploy complete!"
        docker compose -f "$COMPOSE_FILE" ps
    else
        echo "Warning: No compose file found in $WORK_DIR"
        echo "Push a docker-compose.yml to enable auto-deploy."
    fi
```

- [ ] **Step 4: Update existing init test assertion**

In `cmd/app/init_test.go`, update the `TestGenerateInitScript` check for `"docker compose up"`. The existing check `{"docker compose up", "docker compose up -d --build"}` should be updated to:

```go
{"docker compose up", `docker compose -f "$COMPOSE_FILE" up -d --build`},
```

- [ ] **Step 5: Run all tests to verify they pass**

Run: `go test ./cmd/app/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/app/init.go cmd/app/init_test.go
git commit -m "feat: update post-receive hook with compose file priority (#75)"
```

---

### Task 4: Lint and final verification

**Files:** All modified files

- [ ] **Step 1: Run linter**

Run: `make lint`
Expected: No issues

- [ ] **Step 2: Run full test suite**

Run: `make test`
Expected: All tests pass

- [ ] **Step 3: Build**

Run: `make build`
Expected: Clean build

- [ ] **Step 4: Final commit if any lint fixes needed**

```bash
git add -A
git commit -m "fix: lint issues from compose file support"
```
