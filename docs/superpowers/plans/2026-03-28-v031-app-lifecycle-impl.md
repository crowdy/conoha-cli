# v0.3.1 App Lifecycle Commands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 5 app lifecycle commands (deploy, logs, status, stop, restart) with shared helpers and tar-based deploy.

**Architecture:** Shared `connectToApp` helper extracts SSH connection boilerplate. `app deploy` creates tar.gz locally (respecting `.dockerignore`), transfers via `RunWithStdin`, then runs docker compose. Other 4 commands are thin `RunCommand` wrappers. Refactor existing `app init` to use shared helper.

**Tech Stack:** Go 1.26, cobra, `golang.org/x/crypto/ssh`, `archive/tar`, `compress/gzip`

**Spec:** `docs/superpowers/specs/2026-03-28-v031-app-lifecycle-design.md`

---

### Task 1: Add `RunWithStdin` to `internal/ssh/exec.go` (TDD)

**Files:**
- Modify: `internal/ssh/exec.go`
- Modify: `internal/ssh/exec_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/ssh/exec_test.go`:

```go
func TestRunWithStdinNilClient(t *testing.T) {
	_, err := RunWithStdin(nil, "cat", &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/ssh/ -v -run TestRunWithStdinNilClient
```

Expected: FAIL — `RunWithStdin` not defined.

- [ ] **Step 3: Write implementation**

Add to `internal/ssh/exec.go`:

```go
// RunWithStdin executes a command, piping stdinData to its stdin.
// Stdout/stderr are streamed to the provided writers.
// Returns the remote exit code.
func RunWithStdin(client *ssh.Client, command string, stdinData io.Reader, stdout, stderr io.Writer) (int, error) {
	if client == nil {
		return -1, fmt.Errorf("SSH client is nil")
	}

	session, err := client.NewSession()
	if err != nil {
		return -1, err
	}
	defer func() { _ = session.Close() }()

	session.Stdin = stdinData
	session.Stdout = stdout
	session.Stderr = stderr

	if err := session.Run(command); err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			return exitErr.ExitStatus(), nil
		}
		return -1, err
	}
	return 0, nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/ssh/ -v
```

Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ssh/exec.go internal/ssh/exec_test.go
git commit -m "Add RunWithStdin to internal/ssh for stdin piping"
```

---

### Task 2: Add shared `connectToApp` helper and `addAppFlags`

**Files:**
- Create: `cmd/app/connect.go`

- [ ] **Step 1: Create `cmd/app/connect.go`**

```go
package app

import (
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/model"
	"github.com/crowdy/conoha-cli/internal/prompt"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

// appContext holds resolved SSH connection and app info.
type appContext struct {
	Client  *ssh.Client
	AppName string
	Server  *model.Server
	IP      string
	User    string
}

// connectToApp resolves server, app name, and SSH connection from common flags.
// Caller must defer ctx.Client.Close().
func connectToApp(cmd *cobra.Command, args []string) (*appContext, error) {
	client, err := cmdutil.NewClient(cmd)
	if err != nil {
		return nil, err
	}
	compute := api.NewComputeAPI(client)

	s, err := compute.FindServer(args[0])
	if err != nil {
		return nil, err
	}

	ip, err := internalssh.ServerIP(s)
	if err != nil {
		return nil, err
	}

	appName, _ := cmd.Flags().GetString("app-name")
	if appName == "" {
		appName, err = prompt.String("App name")
		if err != nil {
			return nil, err
		}
	}
	if err := internalssh.ValidateAppName(appName); err != nil {
		return nil, err
	}

	user, _ := cmd.Flags().GetString("user")
	port, _ := cmd.Flags().GetString("port")
	identity, _ := cmd.Flags().GetString("identity")

	if identity == "" {
		identity = internalssh.ResolveKeyPath(s.KeyName)
	}
	if identity == "" {
		return nil, fmt.Errorf("no SSH key found; specify --identity or ensure ~/.ssh/conoha_<keyname> exists")
	}

	sshClient, err := internalssh.Connect(internalssh.ConnectConfig{
		Host:    ip,
		Port:    port,
		User:    user,
		KeyPath: identity,
	})
	if err != nil {
		return nil, fmt.Errorf("SSH connect: %w", err)
	}

	return &appContext{
		Client:  sshClient,
		AppName: appName,
		Server:  s,
		IP:      ip,
		User:    user,
	}, nil
}

// addAppFlags adds the common SSH + app-name flags to a command.
func addAppFlags(cmd *cobra.Command) {
	cmd.Flags().String("app-name", "", "application name")
	cmd.Flags().StringP("user", "l", "root", "SSH user")
	cmd.Flags().StringP("port", "p", "22", "SSH port")
	cmd.Flags().StringP("identity", "i", "", "SSH private key path")
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add cmd/app/connect.go
git commit -m "Add shared connectToApp helper for app commands"
```

---

### Task 3: Refactor `app init` to use `connectToApp`

**Files:**
- Modify: `cmd/app/init.go`
- Modify: `cmd/app/app.go`

- [ ] **Step 1: Refactor `cmd/app/init.go`**

Replace the entire file. Keep `generateInitScript` unchanged, replace boilerplate with `connectToApp`:

```go
package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init <id|name>",
	Short: "Initialize app deployment on a server",
	Long:  "Install Docker, create git bare repo with post-receive hook for git-push deploys.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		fmt.Fprintf(os.Stderr, "Initializing app %q on %s (%s)...\n", ctx.AppName, ctx.Server.Name, ctx.IP)

		script := generateInitScript(ctx.AppName)
		exitCode, err := internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("init failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("init script exited with code %d", exitCode)
		}

		fmt.Fprintf(os.Stderr, "\nApp %q initialized on %s (%s).\n\n", ctx.AppName, ctx.Server.Name, ctx.IP)
		fmt.Fprintf(os.Stderr, "Add the remote and deploy:\n")
		fmt.Fprintf(os.Stderr, "  git remote add conoha %s@%s:/opt/conoha/%s.git\n", ctx.User, ctx.IP, ctx.AppName)
		fmt.Fprintf(os.Stderr, "  git push conoha main\n")

		return nil
	},
}

func generateInitScript(appName string) []byte {
	return []byte(fmt.Sprintf(`#!/bin/bash
set -euo pipefail

echo "==> Installing Docker..."
if ! command -v docker &>/dev/null; then
    curl -fsSL https://get.docker.com | sh
fi

echo "==> Installing Docker Compose plugin..."
if ! docker compose version &>/dev/null; then
    apt-get update -qq && apt-get install -y -qq docker-compose-plugin
fi

echo "==> Installing git..."
if ! command -v git &>/dev/null; then
    apt-get update -qq && apt-get install -y -qq git
fi

APP_NAME="%s"
REPO_DIR="/opt/conoha/${APP_NAME}.git"
WORK_DIR="/opt/conoha/${APP_NAME}"

echo "==> Creating directories..."
mkdir -p "$WORK_DIR"

if [ -d "$REPO_DIR" ]; then
    echo "Git repo already exists at $REPO_DIR, skipping."
else
    git init --bare "$REPO_DIR"
fi

echo "==> Installing post-receive hook..."
cat > "$REPO_DIR/hooks/post-receive" << 'HOOK'
#!/bin/bash
set -euo pipefail

APP_NAME="%s"
WORK_DIR="/opt/conoha/${APP_NAME}"
DEPLOY_BRANCH="main"

# Read pushed refs from stdin; only deploy on main branch push
while read -r oldrev newrev refname; do
    branch=$(basename "$refname")
    if [ "$branch" != "$DEPLOY_BRANCH" ]; then
        echo "Pushed to $branch — skipping deploy (only $DEPLOY_BRANCH triggers deploy)."
        continue
    fi

    echo "==> Checking out $DEPLOY_BRANCH..."
    GIT_DIR="$(dirname "$0")/.."
    git --work-tree="$WORK_DIR" --git-dir="$GIT_DIR" checkout -f "$DEPLOY_BRANCH"

    cd "$WORK_DIR"

    if [ -f docker-compose.yml ] || [ -f docker-compose.yaml ] || [ -f compose.yml ] || [ -f compose.yaml ]; then
        echo "==> Building and starting containers..."
        docker compose up -d --build --remove-orphans
        echo "==> Deploy complete!"
        docker compose ps
    else
        echo "Warning: No compose file found in $WORK_DIR"
        echo "Push a docker-compose.yml to enable auto-deploy."
    fi
done
HOOK
chmod +x "$REPO_DIR/hooks/post-receive"

echo "==> Done!"
`, appName, appName))
}
```

- [ ] **Step 2: Run existing tests**

```bash
go test ./cmd/app/ -v
```

Expected: All `TestGenerateInitScript*` tests still PASS.

- [ ] **Step 3: Run full build + lint**

```bash
go build ./... && golangci-lint run ./...
```

Expected: Clean.

- [ ] **Step 4: Commit**

```bash
git add cmd/app/init.go
git commit -m "Refactor app init to use shared connectToApp helper"
```

---

### Task 4: `.dockerignore` parser + tar archive (TDD)

**Files:**
- Create: `cmd/app/dockerignore.go`
- Create: `cmd/app/dockerignore_test.go`
- Create: `cmd/app/tar.go`
- Create: `cmd/app/tar_test.go`

- [ ] **Step 1: Write dockerignore tests**

Create `cmd/app/dockerignore_test.go`:

```go
package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIgnorePatterns_NoFile(t *testing.T) {
	dir := t.TempDir()
	patterns, err := loadIgnorePatterns(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Should have default excludes (.git)
	if len(patterns) != 1 || patterns[0] != ".git" {
		t.Errorf("expected [.git], got %v", patterns)
	}
}

func TestLoadIgnorePatterns_WithFile(t *testing.T) {
	dir := t.TempDir()
	content := "node_modules\n# comment\n\n*.log\nlogs/\n!important.log\n"
	if err := os.WriteFile(filepath.Join(dir, ".dockerignore"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	patterns, err := loadIgnorePatterns(dir)
	if err != nil {
		t.Fatal(err)
	}

	// .git (default) + node_modules + *.log + logs (trailing slash stripped) = 4
	// # comment, blank line, !important.log (negation) are skipped
	want := []string{".git", "node_modules", "*.log", "logs"}
	if len(patterns) != len(want) {
		t.Fatalf("got %v, want %v", patterns, want)
	}
	for i, p := range patterns {
		if p != want[i] {
			t.Errorf("pattern[%d]: got %q, want %q", i, p, want[i])
		}
	}
}

func TestShouldExclude(t *testing.T) {
	patterns := []string{".git", "node_modules", "*.log"}

	tests := []struct {
		path string
		want bool
	}{
		{".git", true},
		{"node_modules", true},
		{"src/app.go", false},
		{"error.log", true},
		{"src/error.log", true}, // basename match
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := shouldExclude(tt.path, patterns); got != tt.want {
				t.Errorf("shouldExclude(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./cmd/app/ -v -run "TestLoadIgnore|TestShouldExclude"
```

Expected: FAIL — functions not defined.

- [ ] **Step 3: Write dockerignore implementation**

Create `cmd/app/dockerignore.go`:

```go
package app

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// defaultExcludes are always excluded from tar archives.
var defaultExcludes = []string{".git"}

// loadIgnorePatterns reads .dockerignore from dir and returns patterns.
// Returns defaultExcludes if .dockerignore doesn't exist.
func loadIgnorePatterns(dir string) ([]string, error) {
	patterns := append([]string{}, defaultExcludes...)

	f, err := os.Open(filepath.Join(dir, ".dockerignore"))
	if err != nil {
		if os.IsNotExist(err) {
			return patterns, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Negation patterns (!) not supported
		if strings.HasPrefix(line, "!") {
			continue
		}
		// Strip trailing slash (Docker treats "logs/" and "logs" identically)
		line = strings.TrimRight(line, "/")
		patterns = append(patterns, line)
	}
	return patterns, scanner.Err()
}

// shouldExclude checks if a path matches any ignore pattern.
// Limitations: patterns match top-level directory names and file basenames only.
// Nested directory matching requires explicit paths (e.g., "src/vendor").
func shouldExclude(path string, patterns []string) bool {
	base := filepath.Base(path)
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, path); matched {
			return true
		}
		if matched, _ := filepath.Match(p, base); matched {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run dockerignore tests**

```bash
go test ./cmd/app/ -v -run "TestLoadIgnore|TestShouldExclude"
```

Expected: All PASS.

- [ ] **Step 5: Write tar tests**

Create `cmd/app/tar_test.go`:

```go
package app

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestCreateTarGz(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(dir, "app.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "compose.yml"), []byte("version: '3'"), 0644)
	os.MkdirAll(filepath.Join(dir, ".git", "objects"), 0755)
	os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("ref: refs/heads/main"), 0644)
	os.WriteFile(filepath.Join(dir, "debug.log"), []byte("log"), 0644)

	patterns := []string{".git", "*.log"}
	var buf bytes.Buffer
	if err := createTarGz(dir, patterns, &buf); err != nil {
		t.Fatal(err)
	}

	// Extract and check contents
	files := extractTarNames(t, &buf)
	sort.Strings(files)

	want := []string{"app.go", "compose.yml"}
	if len(files) != len(want) {
		t.Fatalf("got files %v, want %v", files, want)
	}
	for i, f := range files {
		if f != want[i] {
			t.Errorf("file[%d]: got %q, want %q", i, f, want[i])
		}
	}
}

func TestCreateTarGz_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	if err := createTarGz(dir, nil, &buf); err != nil {
		t.Fatal(err)
	}

	files := extractTarNames(t, &buf)
	if len(files) != 0 {
		t.Errorf("expected empty archive, got %v", files)
	}
}

func extractTarNames(t *testing.T, data *bytes.Buffer) []string {
	t.Helper()
	gr, err := gzip.NewReader(data)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if !hdr.FileInfo().IsDir() {
			names = append(names, hdr.Name)
		}
	}
	return names
}
```

- [ ] **Step 6: Write tar implementation**

Create `cmd/app/tar.go`:

```go
package app

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// createTarGz creates a gzip-compressed tar archive of dir, excluding matched patterns.
func createTarGz(dir string, patterns []string, w io.Writer) error {
	gw := gzip.NewWriter(w)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		if shouldExclude(rel, patterns) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip symlinks (avoid archiving as empty files)
		if d.Type()&os.ModeSymlink != 0 {
			fmt.Fprintf(os.Stderr, "Warning: skipping symlink %s\n", rel)
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = rel

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}
```

- [ ] **Step 7: Run all tar + dockerignore tests**

```bash
go test ./cmd/app/ -v -run "TestCreateTarGz|TestLoadIgnore|TestShouldExclude"
```

Expected: All PASS.

- [ ] **Step 8: Run full build + lint**

```bash
go build ./... && golangci-lint run ./...
```

Expected: Clean.

- [ ] **Step 9: Commit**

```bash
git add cmd/app/dockerignore.go cmd/app/dockerignore_test.go cmd/app/tar.go cmd/app/tar_test.go
git commit -m "Add .dockerignore parser and tar archive support"
```

---

### Task 5: `app deploy` command

**Files:**
- Create: `cmd/app/deploy.go`
- Modify: `cmd/app/app.go`

- [ ] **Step 1: Create `cmd/app/deploy.go`**

```go
package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(deployCmd)
}

var deployCmd = &cobra.Command{
	Use:   "deploy <id|name>",
	Short: "Deploy current directory to a server",
	Long:  "Archive current directory, upload via SSH, and run docker compose up.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		// Pre-flight: check compose file exists locally
		if !hasComposeFile(".") {
			return fmt.Errorf("no docker-compose.yml/yaml or compose.yml/yaml found in current directory")
		}

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

		// Docker compose up
		fmt.Fprintf(os.Stderr, "Building and starting containers...\n")
		composeCmd := fmt.Sprintf("cd %s && docker compose up -d --build --remove-orphans && docker compose ps", workDir)
		exitCode, err = internalssh.RunCommand(ctx.Client, composeCmd, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("deploy failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("deploy exited with code %d", exitCode)
		}

		fmt.Fprintf(os.Stderr, "Deploy complete.\n")
		return nil
	},
}

// hasComposeFile checks if a docker compose file exists in dir.
func hasComposeFile(dir string) bool {
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Register in app.go**

Replace `cmd/app/app.go`:

```go
package app

import "github.com/spf13/cobra"

// Cmd is the app command group.
var Cmd = &cobra.Command{
	Use:   "app",
	Short: "Application deployment commands",
}

func init() {
	Cmd.AddCommand(initCmd)
	Cmd.AddCommand(deployCmd)
}
```

- [ ] **Step 3: Verify build + help**

```bash
go build ./... && go run . app --help | grep deploy
```

Expected: `deploy      Deploy current directory to a server`

- [ ] **Step 4: Run full tests + lint**

```bash
go test ./... && golangci-lint run ./...
```

Expected: All pass.

- [ ] **Step 5: Commit**

```bash
git add cmd/app/deploy.go cmd/app/app.go
git commit -m "Add app deploy command with tar-over-SSH transfer"
```

---

### Task 6: `app logs` command

**Files:**
- Create: `cmd/app/logs.go`
- Modify: `cmd/app/app.go`

- [ ] **Step 1: Create `cmd/app/logs.go`**

```go
package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(logsCmd)
	logsCmd.Flags().BoolP("follow", "f", false, "stream logs in real-time")
	logsCmd.Flags().Int("tail", 100, "number of lines to show")
	logsCmd.Flags().String("service", "", "specific service name")
}

var logsCmd = &cobra.Command{
	Use:   "logs <id|name>",
	Short: "Show app container logs",
	Long:  "Show docker compose logs. Use --follow to stream in real-time (Ctrl+C to stop).",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		follow, _ := cmd.Flags().GetBool("follow")
		tail, _ := cmd.Flags().GetInt("tail")
		service, _ := cmd.Flags().GetString("service")

		workDir := "/opt/conoha/" + ctx.AppName
		composeCmd := fmt.Sprintf("cd %s && docker compose logs --tail %d", workDir, tail)
		if follow {
			composeCmd += " -f"
		}
		if service != "" {
			if err := internalssh.ValidateAppName(service); err != nil {
				return fmt.Errorf("invalid service name: %w", err)
			}
			composeCmd += " " + service
		}

		exitCode, err := internalssh.RunCommand(ctx.Client, composeCmd, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("logs failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("logs exited with code %d", exitCode)
		}
		return nil
	},
}
```

- [ ] **Step 2: Register in app.go**

Add `Cmd.AddCommand(logsCmd)` to `cmd/app/app.go` init().

- [ ] **Step 3: Verify build + help**

```bash
go build ./... && go run . app logs --help
```

Expected: Shows `--follow`, `--tail`, `--service` flags.

- [ ] **Step 4: Run full tests + lint**

```bash
go test ./... && golangci-lint run ./...
```

Expected: All pass.

- [ ] **Step 5: Commit**

```bash
git add cmd/app/logs.go cmd/app/app.go
git commit -m "Add app logs command with follow and tail support"
```

---

### Task 7: `app status`, `app stop`, `app restart` commands

**Files:**
- Create: `cmd/app/status.go`
- Create: `cmd/app/stop.go`
- Create: `cmd/app/restart.go`
- Modify: `cmd/app/app.go`

- [ ] **Step 1: Create `cmd/app/status.go`**

```go
package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status <id|name>",
	Short: "Show app container status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		workDir := "/opt/conoha/" + ctx.AppName
		exitCode, err := internalssh.RunCommand(ctx.Client, fmt.Sprintf("cd %s && docker compose ps", workDir), os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("status failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("status exited with code %d", exitCode)
		}
		return nil
	},
}
```

- [ ] **Step 2: Create `cmd/app/stop.go`**

```go
package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/internal/prompt"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(stopCmd)
}

var stopCmd = &cobra.Command{
	Use:   "stop <id|name>",
	Short: "Stop app containers",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		ok, err := prompt.Confirm(fmt.Sprintf("Stop app %q on %s?", ctx.AppName, ctx.Server.Name))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}

		workDir := "/opt/conoha/" + ctx.AppName
		fmt.Fprintf(os.Stderr, "Stopping app %q on %s...\n", ctx.AppName, ctx.Server.Name)
		exitCode, err := internalssh.RunCommand(ctx.Client, fmt.Sprintf("cd %s && docker compose stop && docker compose ps", workDir), os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("stop failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("stop exited with code %d", exitCode)
		}
		return nil
	},
}
```

- [ ] **Step 3: Create `cmd/app/restart.go`**

```go
package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(restartCmd)
}

var restartCmd = &cobra.Command{
	Use:   "restart <id|name>",
	Short: "Restart app containers",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		workDir := "/opt/conoha/" + ctx.AppName
		fmt.Fprintf(os.Stderr, "Restarting app %q on %s...\n", ctx.AppName, ctx.Server.Name)
		exitCode, err := internalssh.RunCommand(ctx.Client, fmt.Sprintf("cd %s && docker compose restart && docker compose ps", workDir), os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("restart failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("restart exited with code %d", exitCode)
		}
		return nil
	},
}
```

- [ ] **Step 4: Register all 3 in app.go**

Update `cmd/app/app.go` init() to add:

```go
Cmd.AddCommand(statusCmd)
Cmd.AddCommand(stopCmd)
Cmd.AddCommand(restartCmd)
```

- [ ] **Step 5: Verify build + help**

```bash
go build ./... && go run . app --help
```

Expected: Shows `deploy`, `init`, `logs`, `restart`, `status`, `stop` subcommands.

- [ ] **Step 6: Run full tests + lint**

```bash
go test ./... && golangci-lint run ./...
```

Expected: All pass.

- [ ] **Step 7: Commit**

```bash
git add cmd/app/status.go cmd/app/stop.go cmd/app/restart.go cmd/app/app.go
git commit -m "Add app status, stop, and restart commands"
```

---

### Task 8: Update roadmap

**Files:**
- Modify: `docs/roadmap.md`

- [ ] **Step 1: Update roadmap**

In `docs/roadmap.md`, update the Phase 2+ section under v0.3.0 to replace it with a proper v0.3.1 section. After the v0.3.0 section, add:

```markdown
---

## v0.3.1: App Lifecycle Commands (Phase 2)

App lifecycle management commands for deployed applications.

### `app deploy`

```
conoha app deploy <server> --app-name myapp
```

- Archive current directory (respects `.dockerignore`), upload via SSH
- Clean deploy: removes old files, extracts tar, runs `docker compose up -d --build`
- Pre-flight check: verifies compose file exists locally

### `app logs`

```
conoha app logs <server> --app-name myapp [--follow/-f] [--tail N] [--service svc]
```

### `app status` / `app stop` / `app restart`

```
conoha app status <server> --app-name myapp
conoha app stop <server> --app-name myapp
conoha app restart <server> --app-name myapp
```

### Shared Infrastructure

- `connectToApp` helper: shared SSH connection boilerplate for all app commands
- `RunWithStdin`: SSH command execution with stdin data piping
- `.dockerignore` parser: simple glob patterns, always excludes `.git/`

### Phase 3+ (future)

- `app env` (remote env var management)
- `app destroy` (remove deployment)
- `app list` (list deployed apps)
```

- [ ] **Step 2: Verify build**

```bash
go build ./... && go test ./...
```

Expected: All pass.

- [ ] **Step 3: Commit**

```bash
git add docs/roadmap.md
git commit -m "Update roadmap with v0.3.1 app lifecycle commands"
```
