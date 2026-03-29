# v0.3.2 App Management Commands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add app env management (set/get/list/unset), app destroy, and app list commands, plus .env.server → .env copy in deploy.

**Architecture:** `app env` is a cobra command group with 4 subcommands that generate bash scripts to manipulate `/opt/conoha/{app-name}.env.server` via SSH. `app destroy` runs a cleanup script. `app list` scans `/opt/conoha/*.git` with inline SSH connection (no `connectToApp`). `app deploy` is modified to copy `.env.server` → `.env` before compose up.

**Tech Stack:** Go 1.26, cobra, `golang.org/x/crypto/ssh`

**Spec:** `docs/superpowers/specs/2026-03-28-v032-app-management-design.md`

---

### Task 1: `app env` command group with set/get/list/unset (TDD)

**Files:**
- Create: `cmd/app/env.go`
- Create: `cmd/app/env_test.go`
- Modify: `cmd/app/app.go`

- [ ] **Step 1: Write tests**

Create `cmd/app/env_test.go`:

```go
package app

import (
	"strings"
	"testing"
)

func TestGenerateEnvSetScript(t *testing.T) {
	script := generateEnvSetScript("myapp", map[string]string{
		"DB_HOST": "localhost",
		"DB_PORT": "5432",
	})
	s := string(script)

	if !strings.Contains(s, `ENV_FILE="/opt/conoha/myapp.env.server"`) {
		t.Error("missing ENV_FILE path")
	}
	if !strings.Contains(s, "touch") {
		t.Error("missing touch command")
	}
	if !strings.Contains(s, "DB_HOST=localhost") {
		t.Error("missing DB_HOST=localhost")
	}
	if !strings.Contains(s, "DB_PORT=5432") {
		t.Error("missing DB_PORT=5432")
	}
}

func TestGenerateEnvUnsetScript(t *testing.T) {
	script := generateEnvUnsetScript("myapp", []string{"DB_HOST", "DB_PORT"})
	s := string(script)

	if !strings.Contains(s, `ENV_FILE="/opt/conoha/myapp.env.server"`) {
		t.Error("missing ENV_FILE path")
	}
	if !strings.Contains(s, `grep -v "^DB_HOST="`) {
		t.Error("missing grep for DB_HOST")
	}
	if !strings.Contains(s, `grep -v "^DB_PORT="`) {
		t.Error("missing grep for DB_PORT")
	}
}

func TestGenerateEnvGetCommand(t *testing.T) {
	cmd := generateEnvGetCommand("myapp", "DB_HOST")
	if !strings.Contains(cmd, `grep "^DB_HOST="`) {
		t.Error("missing grep for DB_HOST")
	}
	if !strings.Contains(cmd, "cut -d= -f2-") {
		t.Error("missing cut command")
	}
}

func TestGenerateEnvListCommand(t *testing.T) {
	cmd := generateEnvListCommand("myapp")
	if !strings.Contains(cmd, "/opt/conoha/myapp.env.server") {
		t.Error("missing env file path")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./cmd/app/ -v -run "TestGenerateEnv"
```

Expected: FAIL — functions not defined.

- [ ] **Step 3: Write implementation**

Create `cmd/app/env.go`:

```go
package app

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage app environment variables",
}

func init() {
	envCmd.AddCommand(envSetCmd)
	envCmd.AddCommand(envGetCmd)
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envUnsetCmd)

	addAppFlags(envSetCmd)
	addAppFlags(envGetCmd)
	addAppFlags(envListCmd)
	addAppFlags(envUnsetCmd)
}

// --- env set ---

var envSetCmd = &cobra.Command{
	Use:   "set <server> KEY=VALUE [KEY=VALUE...]",
	Short: "Set environment variables",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args[:1])
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		env := make(map[string]string)
		for _, arg := range args[1:] {
			k, v, ok := strings.Cut(arg, "=")
			if !ok {
				return fmt.Errorf("invalid format %q (expected KEY=VALUE)", arg)
			}
			if err := internalssh.ValidateEnvKey(k); err != nil {
				return err
			}
			env[k] = v
		}

		script := generateEnvSetScript(ctx.AppName, env)
		exitCode, err := internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("env set failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("env set exited with code %d", exitCode)
		}

		for k, v := range env {
			fmt.Fprintf(os.Stderr, "Set %s=%s\n", k, v)
		}
		return nil
	},
}

func generateEnvSetScript(appName string, env map[string]string) []byte {
	var b strings.Builder
	b.WriteString("#!/bin/bash\nset -euo pipefail\n")
	b.WriteString(fmt.Sprintf("ENV_FILE=\"/opt/conoha/%s.env.server\"\n", appName))
	b.WriteString("touch \"$ENV_FILE\"\n")

	// Sort keys for deterministic output
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := env[k]
		// Remove old line for this key, then append new value
		b.WriteString(fmt.Sprintf("grep -v \"^%s=\" \"$ENV_FILE\" > \"$ENV_FILE.tmp\" || true\n", k))
		// Shell-safe: single-quote value, escape embedded single quotes
		escaped := strings.ReplaceAll(v, "'", "'\\''")
		b.WriteString(fmt.Sprintf("echo '%s=%s' >> \"$ENV_FILE.tmp\"\n", k, escaped))
		b.WriteString("mv \"$ENV_FILE.tmp\" \"$ENV_FILE\"\n")
	}
	return []byte(b.String())
}

// --- env get ---

var envGetCmd = &cobra.Command{
	Use:   "get <server> KEY",
	Short: "Get an environment variable value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args[:1])
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		key := args[1]
		if err := internalssh.ValidateEnvKey(key); err != nil {
			return err
		}

		command := generateEnvGetCommand(ctx.AppName, key)
		exitCode, err := internalssh.RunCommand(ctx.Client, command, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("env get failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("environment variable %q not set", key)
		}
		return nil
	},
}

func generateEnvGetCommand(appName, key string) string {
	return fmt.Sprintf(
		`grep "^%s=" /opt/conoha/%s.env.server | cut -d= -f2-`,
		key, appName)
}

// --- env list ---

var envListCmd = &cobra.Command{
	Use:   "list <server>",
	Short: "List all environment variables",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		command := generateEnvListCommand(ctx.AppName)
		_, err = internalssh.RunCommand(ctx.Client, command, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("env list failed: %w", err)
		}
		return nil
	},
}

func generateEnvListCommand(appName string) string {
	return fmt.Sprintf(`cat /opt/conoha/%s.env.server 2>/dev/null || true`, appName)
}

// --- env unset ---

var envUnsetCmd = &cobra.Command{
	Use:   "unset <server> KEY [KEY...]",
	Short: "Remove environment variables",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args[:1])
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		keys := args[1:]
		for _, k := range keys {
			if err := internalssh.ValidateEnvKey(k); err != nil {
				return err
			}
		}

		script := generateEnvUnsetScript(ctx.AppName, keys)
		exitCode, err := internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("env unset failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("env unset exited with code %d", exitCode)
		}

		for _, k := range keys {
			fmt.Fprintf(os.Stderr, "Unset %s\n", k)
		}
		return nil
	},
}

func generateEnvUnsetScript(appName string, keys []string) []byte {
	var b strings.Builder
	b.WriteString("#!/bin/bash\nset -euo pipefail\n")
	b.WriteString(fmt.Sprintf("ENV_FILE=\"/opt/conoha/%s.env.server\"\n", appName))
	b.WriteString("[ -f \"$ENV_FILE\" ] || exit 0\n")
	b.WriteString("cp \"$ENV_FILE\" \"$ENV_FILE.tmp\"\n")
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("grep -v \"^%s=\" \"$ENV_FILE.tmp\" > \"$ENV_FILE.tmp2\" || true\n", k))
		b.WriteString("mv \"$ENV_FILE.tmp2\" \"$ENV_FILE.tmp\"\n")
	}
	b.WriteString("mv \"$ENV_FILE.tmp\" \"$ENV_FILE\"\n")
	return []byte(b.String())
}
```

- [ ] **Step 4: Register envCmd in app.go**

Add `Cmd.AddCommand(envCmd)` to `cmd/app/app.go` init().

- [ ] **Step 5: Run tests**

```bash
go test ./cmd/app/ -v -run "TestGenerateEnv"
```

Expected: All PASS.

- [ ] **Step 6: Verify build + help**

```bash
go build ./... && go run . app env --help
```

Expected: Shows get, list, set, unset subcommands.

- [ ] **Step 7: Run full tests + lint**

```bash
go test ./... && golangci-lint run ./...
```

Expected: All pass, 0 issues.

- [ ] **Step 8: Commit**

```bash
git add cmd/app/env.go cmd/app/env_test.go cmd/app/app.go
git commit -m "Add app env command group with set/get/list/unset"
```

---

### Task 2: Modify `app deploy` to copy .env.server → .env

**Files:**
- Modify: `cmd/app/deploy.go`

- [ ] **Step 1: Update deploy.go compose command**

In `cmd/app/deploy.go`, replace the compose command (line 62):

```go
		// Docker compose up (copy .env.server if exists)
		fmt.Fprintf(os.Stderr, "Building and starting containers...\n")
		composeCmd := fmt.Sprintf(
			"ENV_FILE=/opt/conoha/%s.env.server; "+
				"if [ -f \"$ENV_FILE\" ]; then cp \"$ENV_FILE\" %s/.env; fi && "+
				"cd %s && docker compose up -d --build --remove-orphans && docker compose ps",
			ctx.AppName, workDir, workDir)
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: Clean.

- [ ] **Step 3: Run full tests + lint**

```bash
go test ./... && golangci-lint run ./...
```

Expected: All pass.

- [ ] **Step 4: Commit**

```bash
git add cmd/app/deploy.go
git commit -m "Add .env.server to .env copy in app deploy"
```

---

### Task 3: `app destroy` command

**Files:**
- Create: `cmd/app/destroy.go`
- Modify: `cmd/app/app.go`

- [ ] **Step 1: Create `cmd/app/destroy.go`**

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
	addAppFlags(destroyCmd)
}

var destroyCmd = &cobra.Command{
	Use:   "destroy <id|name>",
	Short: "Destroy an app and all its data",
	Long:  "Stop containers, remove work directory, git repository, and environment file.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		ok, err := prompt.Confirm(fmt.Sprintf("Destroy app %q on %s? All data will be deleted.", ctx.AppName, ctx.Server.Name))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}

		script := generateDestroyScript(ctx.AppName)
		exitCode, err := internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("destroy failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("destroy exited with code %d", exitCode)
		}

		fmt.Fprintf(os.Stderr, "App %q destroyed.\n", ctx.AppName)
		return nil
	},
}

func generateDestroyScript(appName string) []byte {
	return []byte(fmt.Sprintf(`#!/bin/bash
set -euo pipefail

APP_NAME="%s"
WORK_DIR="/opt/conoha/${APP_NAME}"
REPO_DIR="/opt/conoha/${APP_NAME}.git"
ENV_FILE="/opt/conoha/${APP_NAME}.env.server"

echo "==> Stopping containers..."
if [ -d "$WORK_DIR" ]; then
    cd "$WORK_DIR"
    docker compose down --remove-orphans 2>/dev/null || true
fi

echo "==> Removing work directory..."
rm -rf "$WORK_DIR"

echo "==> Removing git repository..."
rm -rf "$REPO_DIR"

echo "==> Removing environment file..."
rm -f "$ENV_FILE"

echo "==> Done."
`, appName))
}
```

- [ ] **Step 2: Register in app.go**

Add `Cmd.AddCommand(destroyCmd)` to `cmd/app/app.go` init().

- [ ] **Step 3: Verify build + help**

```bash
go build ./... && go run . app --help | grep destroy
```

Expected: `destroy     Destroy an app and all its data`

- [ ] **Step 4: Run full tests + lint**

```bash
go test ./... && golangci-lint run ./...
```

Expected: All pass.

- [ ] **Step 5: Commit**

```bash
git add cmd/app/destroy.go cmd/app/app.go
git commit -m "Add app destroy command"
```

---

### Task 4: `app list` command

**Files:**
- Create: `cmd/app/list.go`
- Modify: `cmd/app/app.go`

- [ ] **Step 1: Create `cmd/app/list.go`**

```go
package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	listCmd.Flags().StringP("user", "l", "root", "SSH user")
	listCmd.Flags().StringP("port", "p", "22", "SSH port")
	listCmd.Flags().StringP("identity", "i", "", "SSH private key path")
}

var listCmd = &cobra.Command{
	Use:   "list <id|name>",
	Short: "List deployed apps on a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		compute := api.NewComputeAPI(client)

		s, err := compute.FindServer(args[0])
		if err != nil {
			return err
		}

		ip, err := internalssh.ServerIP(s)
		if err != nil {
			return err
		}

		user, _ := cmd.Flags().GetString("user")
		port, _ := cmd.Flags().GetString("port")
		identity, _ := cmd.Flags().GetString("identity")

		if identity == "" {
			identity = internalssh.ResolveKeyPath(s.KeyName)
		}
		if identity == "" {
			return fmt.Errorf("no SSH key found; specify --identity or ensure ~/.ssh/conoha_<keyname> exists")
		}

		sshClient, err := internalssh.Connect(internalssh.ConnectConfig{
			Host:    ip,
			Port:    port,
			User:    user,
			KeyPath: identity,
		})
		if err != nil {
			return fmt.Errorf("SSH connect: %w", err)
		}
		defer func() { _ = sshClient.Close() }()

		script := generateListScript()
		exitCode, err := internalssh.RunScript(sshClient, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("list failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("list exited with code %d", exitCode)
		}
		return nil
	},
}

func generateListScript() []byte {
	return []byte(`#!/bin/bash
for repo in /opt/conoha/*.git; do
    [ -d "$repo" ] || continue
    APP_NAME=$(basename "$repo" .git)
    WORK_DIR="/opt/conoha/${APP_NAME}"

    if [ -d "$WORK_DIR" ] && (cd "$WORK_DIR" && docker compose ps --status running -q 2>/dev/null | grep -q .); then
        STATUS="running"
    elif [ -d "$WORK_DIR" ] && (cd "$WORK_DIR" && docker compose ps -q 2>/dev/null | grep -q .); then
        STATUS="stopped"
    else
        STATUS="no containers"
    fi

    printf "%-30s %s\n" "$APP_NAME" "$STATUS"
done
`)
}
```

- [ ] **Step 2: Register in app.go**

Add `Cmd.AddCommand(listCmd)` to `cmd/app/app.go` init().

- [ ] **Step 3: Verify build + help**

```bash
go build ./... && go run . app --help | grep list
```

Expected: `list        List deployed apps on a server`

- [ ] **Step 4: Run full tests + lint**

```bash
go test ./... && golangci-lint run ./...
```

Expected: All pass.

- [ ] **Step 5: Commit**

```bash
git add cmd/app/list.go cmd/app/app.go
git commit -m "Add app list command"
```

---

### Task 5: Update roadmap

**Files:**
- Modify: `docs/roadmap.md`

- [ ] **Step 1: Update roadmap**

In `docs/roadmap.md`, replace the "Phase 3+ (future)" section at the end with a proper v0.3.2 section:

```markdown
---

## v0.3.2: App Management Commands (Phase 3)

App environment variable management, destruction, and listing.

### `app env`

```
conoha app env set <server> --app-name myapp KEY=VALUE [KEY=VALUE...]
conoha app env get <server> --app-name myapp KEY
conoha app env list <server> --app-name myapp
conoha app env unset <server> --app-name myapp KEY [KEY...]
```

- Environment variables stored in `/opt/conoha/{app-name}.env.server`
- `app deploy` copies `.env.server` → `.env` for docker compose

### `app destroy`

```
conoha app destroy <server> --app-name myapp
```

- Stops containers, removes work dir, git repo, and env file
- Requires confirmation

### `app list`

```
conoha app list <server>
```

- Lists all deployed apps with container status (running/stopped/no containers)
```

- [ ] **Step 2: Verify build**

```bash
go build ./... && go test ./...
```

Expected: All pass.

- [ ] **Step 3: Commit**

```bash
git add docs/roadmap.md
git commit -m "Update roadmap with v0.3.2 app management commands"
```
