# v0.3.2 Design: App Management Commands

## Overview

Add app environment variable management, app destruction, and app listing commands. Builds on Phase 1-2 infrastructure (`internal/ssh/`, `cmd/app/connectToApp`).

Target user: 1-person developer, 1 VM, same as Phase 1-2.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Env storage | `/opt/conoha/{app-name}.env.server` | Outside work dir — survives `rm -rf` during clean deploy |
| Env → .env copy | `app deploy` only (not hook) | Keep hook simple; deploy is the primary path |
| Env subcommands | set/get/list/unset | Heroku/Dokku-style, intuitive and extensible |
| Env file format | `KEY=VALUE` per line | Standard `.env` format, docker compose compatible |
| Destroy scope | Full: containers + work dir + git repo + .env.server | Clean slate for 1-person developer |
| Destroy confirmation | `prompt.Confirm` | Destructive action |
| List detection | `/opt/conoha/*.git` scan | Bare repo presence = app exists |
| List status | `docker compose ps` per app | Shows running/stopped/no containers |
| List SSH | Separate from `connectToApp` (no `--app-name`) | `app list` operates on all apps, not one |

## 1. Environment Variable Storage

**Path:** `/opt/conoha/{app-name}.env.server`

**Format:**
```
DB_HOST=localhost
DB_PORT=5432
APP_SECRET=my-secret-value
```

**Why `.env.server`:**
- `app deploy` does `rm -rf /opt/conoha/{app-name}/` before tar extract
- `.env.server` lives at `/opt/conoha/{app-name}.env.server` (sibling, not inside work dir)
- `app deploy` copies `.env.server` → `/opt/conoha/{app-name}/.env` after tar extract
- Docker Compose auto-reads `.env` from the project directory

## 2. Command: `app env set`

### Usage

```
conoha app env set <server> --app-name myapp KEY=VALUE [KEY=VALUE...]

Flags:
  --app-name <name>     Application name (default: prompted)
  -l, --user <user>     SSH user (default: root)
  -p, --port <port>     SSH port (default: 22)
  -i, --identity <key>  SSH private key path
```

### Implementation: `cmd/app/env.go`

```go
var envCmd = &cobra.Command{
    Use:   "env",
    Short: "Manage app environment variables",
}

func init() {
    envCmd.AddCommand(envSetCmd)
    envCmd.AddCommand(envGetCmd)
    envCmd.AddCommand(envListCmd)
    envCmd.AddCommand(envUnsetCmd)
}
```

**`env set` logic:**
1. Parse KEY=VALUE args using `strings.Cut(arg, "=")` — first `=` splits key from value (value may contain `=`). Validate keys with `ValidateEnvKey`.
2. SSH: read existing `.env.server` (or empty if not exists)
3. Merge: update existing keys, append new keys
4. SSH: write merged content back to `.env.server`
5. Print updated variables to stderr

**Remote script for set:**
```bash
ENV_FILE="/opt/conoha/${APP_NAME}.env.server"
touch "$ENV_FILE"
# For each KEY=VALUE: remove old KEY line, append new
grep -v "^KEY=" "$ENV_FILE" > "$ENV_FILE.tmp" || true
echo "KEY=VALUE" >> "$ENV_FILE.tmp"
mv "$ENV_FILE.tmp" "$ENV_FILE"
```

This is executed as a generated bash script via `RunScript`, with one grep+echo block per KEY=VALUE pair.

## 3. Command: `app env get`

### Usage

```
conoha app env get <server> --app-name myapp KEY
```

**Logic:**
1. Validate KEY with `ValidateEnvKey` (prevents shell injection in grep pattern)
2. SSH: `grep "^KEY=" /opt/conoha/{app-name}.env.server`
2. Extract value (cut -d= -f2-)
3. Print value to stdout (no newline decoration — scripting friendly)
4. If not found: exit with error `environment variable "KEY" not set`

**Remote command:**
```bash
grep "^KEY=" /opt/conoha/${APP_NAME}.env.server | cut -d= -f2-
```

## 4. Command: `app env list`

### Usage

```
conoha app env list <server> --app-name myapp
```

**Logic:**
1. SSH: `cat /opt/conoha/{app-name}.env.server 2>/dev/null || true`
2. Print to stdout
3. Empty file or missing file → empty output (no error)

## 5. Command: `app env unset`

### Usage

```
conoha app env unset <server> --app-name myapp KEY [KEY...]
```

**Logic:**
1. Validate keys with `ValidateEnvKey`
2. SSH: for each KEY, `grep -v "^KEY=" .env.server`
3. Print removed keys to stderr

**Remote script:**
```bash
ENV_FILE="/opt/conoha/${APP_NAME}.env.server"
grep -v "^KEY1=" "$ENV_FILE" | grep -v "^KEY2=" > "$ENV_FILE.tmp" || true
mv "$ENV_FILE.tmp" "$ENV_FILE"
```

## 6. Modify: `app deploy` — .env copy

**File:** `cmd/app/deploy.go`

After tar extract, before `docker compose up`:

```bash
# Copy .env.server to .env if exists
ENV_SERVER="/opt/conoha/${APP_NAME}.env.server"
if [ -f "$ENV_SERVER" ]; then
    cp "$ENV_SERVER" "${WORK_DIR}/.env"
fi
```

This is added to the compose command string:

```go
composeCmd := fmt.Sprintf(
    "ENV_FILE=/opt/conoha/%s.env.server; "+
        "if [ -f \"$ENV_FILE\" ]; then cp \"$ENV_FILE\" %s/.env; fi && "+
        "cd %s && docker compose up -d --build --remove-orphans && docker compose ps",
    ctx.AppName, workDir, workDir)
```

## 7. Command: `app destroy`

### Usage

```
conoha app destroy <server> --app-name myapp
```

### Flow

1. `connectToApp` — resolve server, app name, SSH
2. `prompt.Confirm("Destroy app \"myapp\" on server-name? All data will be deleted.")`
3. SSH script:

```bash
#!/bin/bash
set -euo pipefail
APP_NAME="{app-name}"
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

echo "==> App destroyed."
```

## 8. Command: `app list`

### Usage

```
conoha app list <server>

Flags:
  -l, --user <user>     SSH user (default: root)
  -p, --port <port>     SSH port (default: 22)
  -i, --identity <key>  SSH private key path
```

Note: `--app-name` is NOT a flag for this command. It lists all apps.

### Flow

1. Resolve server, SSH connect (custom — no `connectToApp` since no app-name needed)
2. SSH script that scans `/opt/conoha/*.git` and checks docker compose status:

```bash
#!/bin/bash
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
```

### `app list` SSH connection

Since `app list` doesn't need `--app-name`, it can't use `connectToApp`. Instead:

```go
func connectToServer(cmd *cobra.Command, args []string) (*serverContext, error)
```

A lighter helper that resolves server + SSH but skips app name. Or inline the connection logic in `list.go` (only one command needs it).

Given only one command needs this, inline is simpler. No new helper.

## 9. File Summary

| File | Action | Description |
|------|--------|-------------|
| `cmd/app/env.go` | **New** | `app env` group + set/get/list/unset subcommands |
| `cmd/app/env_test.go` | **New** | ENV_FILE parsing/generation tests |
| `cmd/app/destroy.go` | **New** | `app destroy` command |
| `cmd/app/list.go` | **New** | `app list` command (inline SSH) |
| `cmd/app/deploy.go` | **Modify** | Add .env.server → .env copy before compose up |
| `cmd/app/app.go` | **Modify** | Register envCmd, destroyCmd, listCmd |

## 10. Verification Plan

### Unit Tests

- `cmd/app/env_test.go`: script generation for set (single/multiple keys), unset (single/multiple keys), KEY=VALUE parsing, key validation

### Integration Tests (manual)

1. `conoha app env set <server> --app-name testapp DB_HOST=localhost` — sets env var
2. `conoha app env get <server> --app-name testapp DB_HOST` — prints `localhost`
3. `conoha app env list <server> --app-name testapp` — shows all vars
4. `conoha app env unset <server> --app-name testapp DB_HOST` — removes var
5. `conoha app deploy <server> --app-name testapp` — deploys with `.env` copied
6. `conoha app destroy <server> --app-name testapp` — prompts, destroys everything
7. `conoha app list <server>` — shows apps with status

### CI

- `go test ./...` passes
- `golangci-lint run ./...` passes
- `go build ./...` compiles

## 11. Commit Strategy

| # | Scope | Description |
|---|-------|-------------|
| 1 | `cmd/app/env.go` | Add `app env` command group with set/get/list/unset |
| 2 | `cmd/app/env_test.go` | Add env script generation tests |
| 3 | `cmd/app/deploy.go` | Add .env.server → .env copy to deploy |
| 4 | `cmd/app/destroy.go` | Add `app destroy` command |
| 5 | `cmd/app/list.go` | Add `app list` command |
| 6 | docs | Update roadmap with v0.3.2 |

## 12. Out of Scope (Phase 4+)

- `app env push/pull` — sync env between local `.env` and server
- Post-receive hook `.env` copy (currently deploy-only)
- Hook upgrade command (`app upgrade` to re-generate hooks with new features)
- `app scale` — scaling docker compose services
- Env value encryption at rest
- Multi-server app management
