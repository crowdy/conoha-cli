# ConoHa VPS3 CLI

## Overview
Go CLI tool for ConoHa VPS3 (OpenStack-based Japanese VPS service).
Single binary, agent-friendly design with structured output.

## Build & Test
```bash
make build      # Build binary
make test       # Run tests
make lint       # Run linter (see note below)
make clean      # Clean artifacts
go test ./...   # Run all tests
```

**Note**: `golangci-lint` is installed at `~/.asdf/installs/golang/1.26.1/packages/bin/golangci-lint`.
Add it to PATH before running: `export PATH="$HOME/.asdf/installs/golang/1.26.1/packages/bin:$PATH"`

## Project Structure
```
cmd/           - Cobra command definitions (one package per resource)
internal/api/  - HTTP client and API implementations
internal/config/ - Profile, credentials, token management
internal/model/  - API response/request structs
internal/output/ - Formatters (table, json, yaml, csv)
internal/prompt/ - Interactive prompts
internal/errors/ - Error types and exit codes
```

## Conventions
- Use `cobra` for CLI commands, `viper` for config
- All output formatters implement `output.Formatter` interface
- `--format json` outputs only JSON to stdout; progress/warnings go to stderr
- Exit codes: 0=ok, 1=general, 2=auth, 3=not-found, 4=validation, 5=api, 6=network, 10=cancelled
- API base URLs follow pattern: `https://{service}.{region}.conoha.io`
- Token auto-refresh 5 min before expiry
- Config dir: `~/.config/conoha/` (config.yaml, credentials.yaml, tokens.yaml)
- Credentials file must be 0600
- Environment variables override config: CONOHA_PROFILE, CONOHA_TENANT_ID, etc.
- CONOHA_ENDPOINT overrides API base URL, CONOHA_DEBUG enables debug logging (1 or api)
