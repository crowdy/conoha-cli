# v0.3.1 Design: App Lifecycle Commands

## Overview

Add 5 app lifecycle commands to manage deployed applications on ConoHa VPS. All commands use SSH to execute docker compose operations on the remote server. Builds on Phase 1 (v0.3.0) infrastructure: `internal/ssh/`, `cmd/app/`.

Target user: 1-person developer, 1 VM, same as Phase 1.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Deploy method | tar over SSH (not git push) | Works without git, more flexible |
| tar exclusion | `.dockerignore` patterns | Consistent with Docker build context |
| tar transfer | `RunWithStdin` (stdin pipe) | No temp files, single SSH session for tar, second for compose |
| Always excluded | `.git/` | Never needed on server |
| Logs follow | `--follow` flag + default tail | Both streaming and snapshot use cases |
| Stop confirmation | `prompt.Confirm` | Destructive action, matches existing pattern |
| Restart confirmation | None | Non-destructive (containers restart) |
| Post-action output | `docker compose ps` | Consistent with post-receive hook pattern |
| Boilerplate | Shared `connectToApp` helper | 5 commands share identical SSH connection setup |
| `.dockerignore` parser | Simple glob only | `filepath.Match` sufficient; `!` and `**` not supported |

## 1. New Function: `RunWithStdin`

**File:** `internal/ssh/exec.go`

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

## 2. Common Helper: `cmd/app/connect.go`

Shared boilerplate for all 5 commands + existing `app init`.

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

## 3. Command: `app deploy`

### Usage

```
conoha app deploy <server> --app-name myapp

Flags:
  --app-name <name>     Application name (default: prompted)
  -l, --user <user>     SSH user (default: root)
  -p, --port <port>     SSH port (default: 22)
  -i, --identity <key>  SSH private key path
```

### Flow

1. `connectToApp()` — resolve server, app name, SSH connection
2. Load `.dockerignore` patterns from current directory
3. Create tar.gz in memory (current directory, excluding `.git/` + `.dockerignore` patterns)
4. `RunWithStdin(client, "tar xzf - -C /opt/conoha/{app-name}", tarData, stdout, stderr)` — transfer and extract
5. `RunCommand(client, "cd /opt/conoha/{app-name} && docker compose up -d --build --remove-orphans && docker compose ps", stdout, stderr)` — build and start

### Implementation: `cmd/app/deploy.go`

```go
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

        // Transfer tar
        workDir := "/opt/conoha/" + ctx.AppName
        tarCmd := fmt.Sprintf("mkdir -p %s && tar xzf - -C %s", workDir, workDir)
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
```

## 4. `.dockerignore` Parser: `cmd/app/dockerignore.go`

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
        patterns = append(patterns, line)
    }
    return patterns, scanner.Err()
}

// shouldExclude checks if a path matches any ignore pattern.
func shouldExclude(path string, patterns []string) bool {
    base := filepath.Base(path)
    for _, p := range patterns {
        // Match against full path
        if matched, _ := filepath.Match(p, path); matched {
            return true
        }
        // Match against basename
        if matched, _ := filepath.Match(p, base); matched {
            return true
        }
    }
    return false
}
```

### Tar Archive: `cmd/app/tar.go`

```go
package app

import (
    "archive/tar"
    "compress/gzip"
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

## 5. Command: `app logs`

### Usage

```
conoha app logs <server> --app-name myapp [--follow/-f] [--tail N] [--service svc]

Flags:
  --follow, -f          Stream logs in real-time
  --tail N              Number of lines (default: 100, int type prevents injection)
  --service <name>      Specific service name
```

### Implementation: `cmd/app/logs.go`

```go
var logsCmd = &cobra.Command{
    Use:   "logs <id|name>",
    Short: "Show app container logs",
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

## 6. Command: `app status`

```go
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

## 7. Command: `app stop`

```go
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
        composeCmd := fmt.Sprintf("cd %s && docker compose stop && docker compose ps", workDir)
        exitCode, err := internalssh.RunCommand(ctx.Client, composeCmd, os.Stdout, os.Stderr)
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

## 8. Command: `app restart`

```go
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
        composeCmd := fmt.Sprintf("cd %s && docker compose restart && docker compose ps", workDir)
        exitCode, err := internalssh.RunCommand(ctx.Client, composeCmd, os.Stdout, os.Stderr)
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

## 9. Refactor: `app init` to use `connectToApp`

`cmd/app/init.go`의 보일러플레이트를 `connectToApp`으로 교체. `generateInitScript` 로직은 그대로 유지.

## 10. File Summary

| File | Action | Description |
|------|--------|-------------|
| `internal/ssh/exec.go` | **Modify** | Add `RunWithStdin` |
| `internal/ssh/exec_test.go` | **Modify** | Add `TestRunWithStdinNilClient` |
| `cmd/app/connect.go` | **New** | `connectToApp`, `addAppFlags` helpers |
| `cmd/app/connect_test.go` | **New** | Unit tests for appContext |
| `cmd/app/deploy.go` | **New** | `app deploy` command |
| `cmd/app/tar.go` | **New** | `createTarGz` |
| `cmd/app/tar_test.go` | **New** | tar creation + exclusion tests |
| `cmd/app/dockerignore.go` | **New** | `.dockerignore` parser |
| `cmd/app/dockerignore_test.go` | **New** | Pattern matching tests |
| `cmd/app/logs.go` | **New** | `app logs` command |
| `cmd/app/status.go` | **New** | `app status` command |
| `cmd/app/stop.go` | **New** | `app stop` command |
| `cmd/app/restart.go` | **New** | `app restart` command |
| `cmd/app/init.go` | **Modify** | Refactor to use `connectToApp` |
| `cmd/app/app.go` | **Modify** | Register 5 new commands |

## 11. Verification Plan

### Unit Tests

- `internal/ssh/exec_test.go`: `RunWithStdin` nil client test
- `cmd/app/tar_test.go`: create tar, verify contents, verify exclusion works
- `cmd/app/dockerignore_test.go`: pattern loading, `.git/` always excluded, comment/blank lines skipped, negation ignored
- `cmd/app/connect_test.go`: `addAppFlags` registers expected flags

### Integration Tests (manual)

1. `conoha app deploy <server> --app-name testapp` — tar transfers, compose starts
2. `conoha app logs <server> --app-name testapp --tail 10` — shows recent logs
3. `conoha app logs <server> --app-name testapp -f` — streams, Ctrl+C stops
4. `conoha app status <server> --app-name testapp` — shows container status
5. `conoha app stop <server> --app-name testapp` — prompts, stops containers
6. `conoha app restart <server> --app-name testapp` — restarts, shows status

### CI

- `go test ./...` passes
- `golangci-lint run ./...` passes
- `go build ./...` compiles

## 12. Commit Strategy

| # | Scope | Description |
|---|-------|-------------|
| 1 | `internal/ssh/exec.go` | Add `RunWithStdin` function |
| 2 | `cmd/app/connect.go` | Add shared `connectToApp` helper |
| 3 | `cmd/app/init.go` | Refactor to use `connectToApp` |
| 4 | `cmd/app/dockerignore.go`, `cmd/app/tar.go` | Add tar + dockerignore support |
| 5 | `cmd/app/deploy.go` | Add `app deploy` command |
| 6 | `cmd/app/logs.go` | Add `app logs` command |
| 7 | `cmd/app/status.go`, `stop.go`, `restart.go` | Add status/stop/restart commands |
| 8 | docs | Update roadmap with v0.3.1 |

## 13. Out of Scope (Phase 3+)

- `app env` — remote environment variable management
- `app destroy` — delete deployment (remove repo + working dir + containers)
- `app list` — list deployed apps on a server
- `.dockerignore` negation patterns (`!`)
- `.dockerignore` recursive patterns (`**`)
- Symlink handling in tar
- Multi-service deploy (deploy specific services only)
