# conoha - ConoHa VPS3 CLI

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[日本語](README.md) | [한국어](README-ko.md)

A command-line interface for the ConoHa VPS3 API. Written in Go as a single binary with an agent-friendly design.

**[Documentation](https://crowdy.github.io/conoha-cli-pages/)** — Guides, deployment examples, and command reference

> **Note**: This tool is for VPS3 only. It is not compatible with legacy VPS2 CLIs (hironobu-s/conoha-vps, miyabisun/conoha-cli).

## Features

- Single binary, cross-platform (Linux / macOS / Windows)
- Multiple profile support (`gh auth` style)
- Structured output (`--format json/yaml/csv/table`)
- Agent-friendly design (`--no-input`, deterministic exit codes, stderr/stdout separation)
- Automatic token refresh (re-authenticates 5 minutes before expiry)

## Installation

### Scoop (Windows)

```powershell
scoop bucket add crowdy https://github.com/crowdy/crowdy-bucket
scoop install conoha
```

### Build from source

```bash
go install github.com/crowdy/conoha-cli@latest
```

### Release binaries

Download from the [Releases](https://github.com/crowdy/conoha-cli/releases) page, or use the commands below:

**Linux (amd64)**

```bash
VERSION=$(curl -s https://api.github.com/repos/crowdy/conoha-cli/releases/latest | grep tag_name | cut -d '"' -f4)
curl -Lo conoha.tar.gz "https://github.com/crowdy/conoha-cli/releases/download/${VERSION}/conoha-cli_${VERSION#v}_linux_amd64.tar.gz"
tar xzf conoha.tar.gz conoha
sudo mv conoha /usr/local/bin/
rm conoha.tar.gz
```

**macOS (Apple Silicon)**

```bash
VERSION=$(curl -s https://api.github.com/repos/crowdy/conoha-cli/releases/latest | grep tag_name | cut -d '"' -f4)
curl -Lo conoha.tar.gz "https://github.com/crowdy/conoha-cli/releases/download/${VERSION}/conoha-cli_${VERSION#v}_darwin_arm64.tar.gz"
tar xzf conoha.tar.gz conoha
sudo mv conoha /usr/local/bin/
rm conoha.tar.gz
```

**Windows (amd64)**

```powershell
$version = (Invoke-RestMethod https://api.github.com/repos/crowdy/conoha-cli/releases/latest).tag_name
$v = $version -replace '^v', ''
Invoke-WebRequest -Uri "https://github.com/crowdy/conoha-cli/releases/download/$version/conoha-cli_${v}_windows_amd64.zip" -OutFile conoha.zip
Expand-Archive conoha.zip -DestinationPath .
Remove-Item conoha.zip
```

## Quick Start

```bash
# Login (enter tenant ID, username, password)
conoha auth login

# Check authentication status
conoha auth status

# List servers
conoha server list

# Output in JSON format
conoha server list --format json

# Show server details (by ID or name)
conoha server show <server-id-or-name>

# Rename a server
conoha server rename <server-id-or-name> new-name
```

## Commands

| Command | Description |
|---------|-------------|
| `conoha auth` | Authentication (login / logout / status / list / switch / token / remove) |
| `conoha server` | Server management (list / show / create / delete / start / stop / reboot / resize / rebuild / rename / console / ips / metadata / ssh / deploy / attach-volume / detach-volume) |
| `conoha flavor` | Flavor listing (list / show) |
| `conoha keypair` | SSH keypair management (list / create / delete) |
| `conoha volume` | Block storage (list / show / create / delete / types / backup) |
| `conoha image` | Image management (list / show / delete) |
| `conoha network` | Network management (network / subnet / port / security-group / qos) |
| `conoha lb` | Load balancer (lb / listener / pool / member / healthmonitor) |
| `conoha dns` | DNS management (domain / record) |
| `conoha storage` | Object storage (container / ls / cp / rm / publish) |
| `conoha identity` | Identity management (credential / subuser / role) |
| `conoha app` | App deployment & management (init / deploy / logs / status / stop / restart / env / destroy / list) |
| `conoha config` | CLI configuration (show / set / path) |

## Configuration

Configuration files are stored in `~/.config/conoha/`:

| File | Description | Permission |
|------|-------------|------------|
| `config.yaml` | Profile settings | 0600 |
| `credentials.yaml` | Passwords | 0600 |
| `tokens.yaml` | Token cache | 0600 |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `CONOHA_PROFILE` | Profile name to use |
| `CONOHA_TENANT_ID` | Tenant ID |
| `CONOHA_USERNAME` | API username |
| `CONOHA_PASSWORD` | API password |
| `CONOHA_TOKEN` | Auth token (direct) |
| `CONOHA_FORMAT` | Output format |
| `CONOHA_CONFIG_DIR` | Config directory path |
| `CONOHA_NO_INPUT` | Non-interactive mode (`1` or `true`) |
| `CONOHA_ENDPOINT` | API endpoint override |
| `CONOHA_ENDPOINT_MODE` | `int` for internal API mode (appends service to path) |
| `CONOHA_DEBUG` | Debug logging (`1` or `api`) |

Priority: environment variables > flags > profile config > defaults

### Global Flags

```
--profile    Profile to use
--format     Output format (table / json / yaml / csv)
--no-input   Disable interactive prompts
--quiet      Suppress non-essential output
--verbose    Verbose output
--no-color   Disable color output
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Authentication failure |
| 3 | Resource not found |
| 4 | Validation error |
| 5 | API error |
| 6 | Network error |
| 10 | User cancelled |

## Agent Integration

This CLI is designed for use by scripts and AI agents:

```bash
# Non-interactive mode with JSON output
conoha server list --format json --no-input

# Get token for scripting
TOKEN=$(conoha auth token)

# Error handling via exit codes
conoha server show abc123 || echo "Exit code: $?"
```

## Development

```bash
make build     # Build binary
make test      # Run tests
make lint      # Run linter
make clean     # Clean artifacts
```

## API Documentation

- [ConoHa VPS3 API Reference](https://doc.conoha.jp/reference/api-vps3/)

## License

MIT License - See [LICENSE](LICENSE) for details.
