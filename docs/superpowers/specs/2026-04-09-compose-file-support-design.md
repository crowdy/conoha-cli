# Compose File Support for app deploy

**Issue:** [#75](https://github.com/crowdy/conoha-cli/issues/75)
**Date:** 2026-04-09

## Problem

`conoha app deploy` always uses whichever compose file `docker compose` auto-detects in the current directory. Projects that use `docker-compose.yml` for local dev or other cloud providers must manually swap files before deploying to ConoHa.

## Solution

### 1. Auto-detect `conoha-docker-compose.yml`

Refactor `hasComposeFile()` in `deploy.go` to `detectComposeFile(dir string) (string, error)` that returns the first matching filename in priority order:

1. `conoha-docker-compose.yml`
2. `conoha-docker-compose.yaml`
3. `docker-compose.yml`
4. `docker-compose.yaml`
5. `compose.yml`
6. `compose.yaml`

Returns an error if none found.

### 2. `--compose-file` flag

Add `--compose-file` / `-f` (string, optional) to `addAppFlags()` in `connect.go` so all app subcommands receive it. Only `deploy` reads it initially.

When specified:
- Validate the file exists locally before deploying
- Use that filename for the remote `docker compose -f <file>` command

When not specified:
- Auto-detect using the 6-file priority order

In both cases, the remote command becomes:

```bash
docker compose -f <file> up -d --build --remove-orphans
docker compose -f <file> ps
```

### 3. Post-receive hook update

Update the hook template in `init.go` to match the new 6-file priority order using shell if/elif chain:

```bash
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
    docker compose -f "$COMPOSE_FILE" ps
fi
```

### 4. Error handling

- `--compose-file` points to non-existent file: error `compose file not found: <path>` (exit code 4)
- No compose file detected and no flag: error `no compose file found in current directory (checked conoha-docker-compose.yml/yaml, docker-compose.yml/yaml, compose.yml/yaml)` (exit code 4)

### 5. Testing

- **`deploy_test.go`**: Test `detectComposeFile()` with various file combinations (conoha priority, fallback, no file)
- **Existing deploy tests**: Update remote command assertions to include `-f <file>`
- **Flag tests**: Verify `--compose-file` flag is registered on all app subcommands
- **`init_test.go`**: Update post-receive hook assertions to match new template

## Files to modify

| File | Change |
|------|--------|
| `cmd/app/deploy.go` | Refactor `hasComposeFile()` to `detectComposeFile()`, update `deployApp()` to use detected/specified file, pass `-f` to remote commands |
| `cmd/app/connect.go` | Add `--compose-file` / `-f` flag to `addAppFlags()` |
| `cmd/app/init.go` | Update post-receive hook template with new detection order |
| `cmd/app/deploy_test.go` | Add `detectComposeFile()` tests, update deploy flow tests |
| `cmd/app/init_test.go` | Update hook template assertions |

## Out of scope

- Config persistence (saving compose file choice in app config)
- Archive filtering (excluding non-selected compose files from tar)
- `--compose-file` support baked into post-receive hook
