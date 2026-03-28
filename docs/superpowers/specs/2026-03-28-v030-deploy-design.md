# v0.3.0 Design: Deploy Feature (Phase 1)

## Overview

SSH-based deploy infrastructure for ConoHa VPS. Phase 1 delivers two commands and a shared SSH library:
- `conoha server deploy` — upload and execute a script on a remote server
- `conoha app init` — set up Docker + git bare repo + post-receive hook on a server
- `internal/ssh/` — shared SSH execution package extracted from `cmd/server/ssh.go`

Target user: 1-person developer deploying to 1 VM, Dokku-style git push workflow.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| SSH library | `golang.org/x/crypto/ssh` | Need programmatic SSH (file upload, capture exit code); `syscall.Exec` replaces process, can't stream+capture |
| SSH helper location | `internal/ssh/` | Shared by `server deploy` and `app init`; keeps cmd/ thin |
| `server ssh` refactor | Keep as-is (syscall.Exec) | Interactive SSH needs process replacement; deploy needs programmatic control — different needs |
| App command group | `cmd/app/` | Separate from server — app lifecycle is a distinct concern |
| Agent on VM | No daemon | SSH + git bare repo + post-receive hook. Zero runtime dependencies beyond Docker + git |
| Build location | VM-side `docker compose build` | Simplest for 1-person use; no registry needed |
| VM base path | `/opt/conoha/` | Standard location, non-home directory, consistent naming |
| Script upload method | SSH exec with heredoc (stdin pipe) | No SCP dependency, works through any SSH config, single connection |
| Environment passing | `--env KEY=VALUE` exported before script | Shell-safe quoting with single quotes + escaping |

## 2. New Package: `internal/ssh/`

### `internal/ssh/exec.go`

Shared SSH execution helpers used by both `server deploy` and `app init`.

```go
package ssh

import (
    "fmt"
    "io"
    "os"

    "golang.org/x/crypto/ssh"
)

// ConnectConfig holds SSH connection parameters.
type ConnectConfig struct {
    Host       string // IP or hostname
    Port       string // default "22"
    User       string // default "root"
    KeyPath    string // path to private key file
}

// Connect establishes an SSH connection.
func Connect(cfg ConnectConfig) (*ssh.Client, error)

// RunScript uploads and executes a script on the remote server.
// Environment variables are exported before the script runs.
// Stdout/stderr are streamed to the provided writers.
// Returns the remote exit code.
func RunScript(client *ssh.Client, script []byte, env map[string]string, stdout, stderr io.Writer) (int, error)

// RunCommand executes a single command on the remote server.
// Stdout/stderr are streamed to the provided writers.
// Returns the remote exit code.
func RunCommand(client *ssh.Client, command string, stdout, stderr io.Writer) (int, error)
```

### SSH Connection Logic

```go
func Connect(cfg ConnectConfig) (*ssh.Client, error) {
    if cfg.Port == "" {
        cfg.Port = "22"
    }
    if cfg.User == "" {
        cfg.User = "root"
    }

    key, err := os.ReadFile(cfg.KeyPath)
    if err != nil {
        return nil, fmt.Errorf("read key %s: %w", cfg.KeyPath, err)
    }

    signer, err := ssh.ParsePrivateKey(key)
    if err != nil {
        return nil, fmt.Errorf("parse key %s: %w", cfg.KeyPath, err)
    }

    config := &ssh.ClientConfig{
        User: cfg.User,
        Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
        HostKeyCallback: ssh.InsecureIgnoreHostKey(), // personal VPS use
    }

    addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
    return ssh.Dial("tcp", addr, config)
}
```

### Script Execution Logic

```go
func RunScript(client *ssh.Client, script []byte, env map[string]string, stdout, stderr io.Writer) (int, error) {
    session, err := client.NewSession()
    if err != nil {
        return -1, err
    }
    defer session.Close()

    session.Stdout = stdout
    session.Stderr = stderr

    // Build command: export env vars, then execute script from stdin
    var envPrefix string
    for k, v := range env {
        // Shell-safe: single-quote value, escape embedded single quotes
        escaped := strings.ReplaceAll(v, "'", "'\\''")
        envPrefix += fmt.Sprintf("export %s='%s'; ", k, escaped)
    }

    session.Stdin = bytes.NewReader(script)
    cmd := envPrefix + "bash -s"

    if err := session.Run(cmd); err != nil {
        if exitErr, ok := err.(*ssh.ExitError); ok {
            return exitErr.ExitStatus(), nil
        }
        return -1, err
    }
    return 0, nil
}
```

### RunCommand Logic

```go
func RunCommand(client *ssh.Client, command string, stdout, stderr io.Writer) (int, error) {
    session, err := client.NewSession()
    if err != nil {
        return -1, err
    }
    defer session.Close()

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

## 3. Command: `conoha server deploy`

### Usage

```
conoha server deploy <id|name> --script <file> [--env KEY=VALUE]...

Flags:
  --script <file>       Local script file to upload and execute (required)
  --env KEY=VALUE       Environment variable (repeatable)
  -l, --user <user>     SSH user (default: root)
  -p, --port <port>     SSH port (default: 22)
  -i, --identity <key>  SSH private key path (overrides auto-detection)
```

### Examples

```bash
# Simple deploy
conoha server deploy my-server --script deploy.sh

# With environment variables
conoha server deploy my-server --script deploy.sh \
  --env APP_ENV=production \
  --env DB_HOST=localhost

# Non-root user, custom port
conoha server deploy my-server --script deploy.sh -l ubuntu -p 2222
```

### Implementation: `cmd/server/deploy.go`

```go
package server

import (
    "fmt"
    "os"
    "strings"

    "github.com/spf13/cobra"

    internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
    deployCmd.Flags().String("script", "", "local script file to upload and execute")
    deployCmd.Flags().StringArray("env", nil, "environment variables (KEY=VALUE, repeatable)")
    deployCmd.Flags().StringP("user", "l", "root", "SSH user")
    deployCmd.Flags().StringP("port", "p", "22", "SSH port")
    deployCmd.Flags().StringP("identity", "i", "", "SSH private key path")
    _ = deployCmd.MarkFlagRequired("script")
}

var deployCmd = &cobra.Command{
    Use:   "deploy <id|name>",
    Short: "Deploy a script to a server via SSH",
    Long:  "Upload and execute a local script on a remote server. Streams output in real-time.",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        compute, err := getComputeAPI(cmd)
        if err != nil {
            return err
        }

        s, err := compute.FindServer(args[0])
        if err != nil {
            return err
        }

        ip, err := getServerIP(s)
        if err != nil {
            return err
        }

        scriptPath, _ := cmd.Flags().GetString("script")
        envFlags, _ := cmd.Flags().GetStringArray("env")
        user, _ := cmd.Flags().GetString("user")
        port, _ := cmd.Flags().GetString("port")
        identity, _ := cmd.Flags().GetString("identity")

        if identity == "" {
            identity = resolveKeyPath(s.KeyName)
        }
        if identity == "" {
            return fmt.Errorf("no SSH key found; specify --identity or ensure ~/.ssh/conoha_<keyname> exists")
        }

        // Read script file
        script, err := os.ReadFile(scriptPath)
        if err != nil {
            return fmt.Errorf("read script %s: %w", scriptPath, err)
        }

        // Parse --env flags
        env := make(map[string]string)
        for _, e := range envFlags {
            k, v, ok := strings.Cut(e, "=")
            if !ok {
                return fmt.Errorf("invalid --env format %q (expected KEY=VALUE)", e)
            }
            env[k] = v
        }

        // Connect
        client, err := internalssh.Connect(internalssh.ConnectConfig{
            Host:    ip,
            Port:    port,
            User:    user,
            KeyPath: identity,
        })
        if err != nil {
            return fmt.Errorf("SSH connect: %w", err)
        }
        defer client.Close()

        fmt.Fprintf(os.Stderr, "Deploying %s to %s (%s)...\n", scriptPath, s.Name, ip)

        // Execute
        exitCode, err := internalssh.RunScript(client, script, env, os.Stdout, os.Stderr)
        if err != nil {
            return fmt.Errorf("deploy failed: %w", err)
        }

        if exitCode != 0 {
            return fmt.Errorf("script exited with code %d", exitCode)
        }

        fmt.Fprintf(os.Stderr, "Deploy complete.\n")
        return nil
    },
}
```

### Validation

| Check | Condition | Error |
|-------|-----------|-------|
| Script file exists | `os.ReadFile` fails | `read script <path>: <err>` |
| SSH key available | No `--identity` and no auto-detected key | `no SSH key found; specify --identity...` |
| Env format | Missing `=` | `invalid --env format "FOO" (expected KEY=VALUE)` |
| Server exists | `FindServer` fails | API error (existing handling) |
| Server has IP | No IPv4 | `no IPv4 address found...` (existing) |

## 4. Command: `conoha app init`

### Usage

```
conoha app init <id|name> [--app-name <name>]

Flags:
  --app-name <name>     Application name (default: prompted interactively)
  -l, --user <user>     SSH user (default: root)
  -p, --port <port>     SSH port (default: 22)
  -i, --identity <key>  SSH private key path (overrides auto-detection)
```

### What It Does

1. SSH into the server
2. Install Docker + Docker Compose + git (if not present)
3. Create git bare repo at `/opt/conoha/{app-name}.git/`
4. Create working directory at `/opt/conoha/{app-name}/`
5. Install post-receive hook that:
   - Checks out code to working directory
   - Runs `docker compose up -d --build`
6. Print git remote URL for the user to add

### Examples

```bash
# Interactive (prompts for app name)
conoha app init my-server

# Non-interactive
conoha app init my-server --app-name myapp

# Output after success:
#   App "myapp" initialized on my-server (203.0.113.1).
#
#   Add the remote and deploy:
#     git remote add conoha root@203.0.113.1:/opt/conoha/myapp.git
#     git push conoha main
```

### Implementation: `cmd/app/app.go`

```go
package app

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
    Use:   "app",
    Short: "Application deployment commands",
}

func init() {
    Cmd.AddCommand(initCmd)
}
```

### Implementation: `cmd/app/init.go`

```go
package app

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"

    "github.com/crowdy/conoha-cli/cmd/cmdutil"
    "github.com/crowdy/conoha-cli/internal/api"
    "github.com/crowdy/conoha-cli/internal/prompt"
    internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
    initCmd.Flags().String("app-name", "", "application name")
    initCmd.Flags().StringP("user", "l", "root", "SSH user")
    initCmd.Flags().StringP("port", "p", "22", "SSH port")
    initCmd.Flags().StringP("identity", "i", "", "SSH private key path")
}

var initCmd = &cobra.Command{
    Use:   "init <id|name>",
    Short: "Initialize app deployment on a server",
    Long:  "Install Docker, create git bare repo with post-receive hook for git-push deploys.",
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

        ip, err := getServerIP(s)
        if err != nil {
            return err
        }

        appName, _ := cmd.Flags().GetString("app-name")
        if appName == "" {
            appName, err = prompt.String("App name")
            if err != nil {
                return err
            }
        }

        user, _ := cmd.Flags().GetString("user")
        port, _ := cmd.Flags().GetString("port")
        identity, _ := cmd.Flags().GetString("identity")

        if identity == "" {
            identity = resolveKeyPath(s.KeyName)
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
        defer sshClient.Close()

        fmt.Fprintf(os.Stderr, "Initializing app %q on %s (%s)...\n", appName, s.Name, ip)

        // Run init script
        script := generateInitScript(appName)
        exitCode, err := internalssh.RunScript(sshClient, script, nil, os.Stdout, os.Stderr)
        if err != nil {
            return fmt.Errorf("init failed: %w", err)
        }
        if exitCode != 0 {
            return fmt.Errorf("init script exited with code %d", exitCode)
        }

        fmt.Fprintf(os.Stderr, "\nApp %q initialized on %s (%s).\n\n", appName, s.Name, ip)
        fmt.Fprintf(os.Stderr, "Add the remote and deploy:\n")
        fmt.Fprintf(os.Stderr, "  git remote add conoha %s@%s:/opt/conoha/%s.git\n", user, ip, appName)
        fmt.Fprintf(os.Stderr, "  git push conoha main\n")

        return nil
    },
}
```

### Init Script Template

```go
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

echo "==> Checking out code..."
git --work-tree="$WORK_DIR" --git-dir="$(dirname "$0")/.." checkout -f

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
HOOK
chmod +x "$REPO_DIR/hooks/post-receive"

echo "==> Done!"
`, appName, appName))
}
```

## 5. Shared Helpers in `cmd/app/`

`cmd/app/` needs `getServerIP()` and `resolveKeyPath()` which currently live in `cmd/server/`. Two options:

| Option | Approach | Chosen |
|--------|----------|--------|
| A | Move to `internal/ssh/` as exported functions | **Yes** |
| B | Duplicate in `cmd/app/` | No — DRY violation |

Move `getServerIP()` → `internal/ssh/ServerIP()` and `resolveKeyPath()` → `internal/ssh/ResolveKeyPath()`.

Update `cmd/server/ssh.go` and `cmd/server/deploy.go` to call the new locations.

```go
// internal/ssh/resolve.go

package ssh

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/crowdy/conoha-cli/internal/model"
)

// ServerIP extracts the best IPv4 address from server addresses.
// Prefers floating IPv4 over fixed IPv4.
func ServerIP(s *model.Server) (string, error) {
    // ... same logic as current getServerIP()
}

// ResolveKeyPath returns the SSH private key path for a server's key name.
// Looks for ~/.ssh/conoha_<keyName>. Returns empty string if not found.
func ResolveKeyPath(keyName string) string {
    // ... same logic as current resolveKeyPath()
}
```

## 6. Command Registration

### `cmd/root.go` change

```go
import "github.com/crowdy/conoha-cli/cmd/app"

// In init():
rootCmd.AddCommand(app.Cmd)
```

### `cmd/server/server.go` change

```go
// In init():
Cmd.AddCommand(deployCmd)
```

## 7. Dependency

New Go dependency:

```
go get golang.org/x/crypto/ssh
```

This is already an indirect dependency via `golang.org/x/term`. Adding direct usage.

## 8. File Summary

| File | Action | Description |
|------|--------|-------------|
| `internal/ssh/exec.go` | **New** | Connect, RunScript, RunCommand |
| `internal/ssh/resolve.go` | **New** | ServerIP, ResolveKeyPath (moved from cmd/server/ssh.go) |
| `cmd/server/deploy.go` | **New** | `server deploy` command |
| `cmd/app/app.go` | **New** | `app` command group |
| `cmd/app/init.go` | **New** | `app init` command |
| `cmd/server/ssh.go` | **Modify** | Use `internal/ssh.ServerIP`, `internal/ssh.ResolveKeyPath` |
| `cmd/server/ssh_test.go` | **Modify** | Update test imports after function move |
| `cmd/server/server.go` | **Modify** | Register deployCmd |
| `cmd/root.go` | **Modify** | Register app.Cmd |
| `go.mod` | **Modify** | Add `golang.org/x/crypto` direct dependency |
| `docs/roadmap.md` | **Modify** | Update roadmap with v0.3.0 section |

## 9. Verification Plan

### Unit Tests

- `internal/ssh/resolve_test.go`: ServerIP (floating/fixed/none), ResolveKeyPath (exists/missing/empty)
- `cmd/server/deploy_test.go`: flag parsing, env validation (KEY=VALUE format)
- `cmd/app/init_test.go`: generateInitScript output validation

### Integration Tests (manual)

1. `conoha server deploy <server> --script test.sh` — script runs, output streams
2. `conoha server deploy <server> --script test.sh --env FOO=bar` — env var visible in script
3. `conoha server deploy <server> --script fail.sh` — non-zero exit code propagated
4. `conoha app init <server> --app-name testapp` — Docker + git installed, repo created
5. `git push conoha main` — post-receive hook triggers, containers start
6. `conoha server ssh <server>` — still works after refactor (regression)

### CI

- `go test ./...` passes
- `golangci-lint run ./...` passes
- `go build ./...` compiles

## 10. Commit Strategy

| # | Scope | Description |
|---|-------|-------------|
| 1 | `internal/ssh/` | Add SSH package: Connect, RunScript, RunCommand |
| 2 | `internal/ssh/` | Move ServerIP, ResolveKeyPath from cmd/server/ssh.go |
| 3 | `cmd/server/ssh.go` | Refactor to use internal/ssh helpers |
| 4 | `cmd/server/deploy.go` | Add `server deploy` command |
| 5 | `cmd/app/` | Add `app init` command |
| 6 | tests | Add unit tests for ssh, deploy, app init |
| 7 | docs | Update roadmap with v0.3.0 |

## 11. Out of Scope (Phase 2+)

- `app deploy` (git push wrapper) — Phase 2
- `app logs/status/stop/restart` — Phase 2
- `app env` (remote env var management) — Phase 3
- `app destroy/list` — Phase 3
- SSL/domain configuration — user's Compose responsibility
- Multi-server deploy — not in scope
- Password-based SSH auth — key-only
