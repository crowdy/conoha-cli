# App Reset Command Design Spec

## Problem

Redeploying an app from scratch requires three separate commands (`app destroy`, `app init`, `app deploy`). This is tedious for iterative development and error-prone in automated/agent workflows.

## Solution

Add `conoha app reset <server> --app-name <app>` that combines destroy + init + deploy in a single command with a single SSH connection.

## Command

```
conoha app reset <id|name> --app-name <app> [--yes] [--user root] [--port 22] [--identity <key>]
```

- `--yes` skips the confirmation prompt
- Inherits all standard app flags via `addAppFlags()`

## Execution Flow

1. `connectToApp()` — resolve server, SSH connect (1 connection, reused)
2. Confirmation prompt (skipped with `--yes`)
3. Run `generateDestroyScript()` via SSH — stop containers, remove work dir/git repo/env file
4. Run `generateInitScript()` via SSH — install Docker/git, create bare repo + post-receive hook
5. Run deploy logic — check local compose file, create tar.gz, upload, docker compose up

## Implementation

### Extract deploy helper

Refactor `deploy.go`: extract the RunE body into `deployApp(ctx *appContext) error`. The `deployCmd.RunE` calls this helper.

### New file: `reset.go`

- `resetCmd` cobra command with `--yes` flag
- Calls `connectToApp()`, then sequentially: destroy script, init script, `deployApp()`
- Each step prints progress to stderr

### Registration

Add `resetCmd` to `app.go` via `Cmd.AddCommand(resetCmd)`.

## Error Handling

Each step (destroy, init, deploy) fails fast. If init fails after destroy, the app is in a partially destroyed state — user can manually run `app init` + `app deploy` to recover.

## Testing

- `reset_test.go`: verify command structure, `--yes` flag registration
- Deploy helper extraction is covered by existing deploy behavior (no behavior change)
