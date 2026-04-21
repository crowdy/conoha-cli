# conoha - ConoHa VPS3 CLI

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

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
- Claude Code skill integration (`conoha skill install` to add infrastructure recipes)

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

> **Tip**: If you already have [Scoop](https://scoop.sh/) installed, dropping the binary into the `shims` directory is easier than editing `%PATH%`:
>
> ```cmd
> move conoha.exe %USERPROFILE%\scoop\shims\
> ```

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
| `conoha app` | App deployment & management (init / deploy / rollback / logs / status / stop / restart / env / destroy / list) |
| `conoha proxy` | conoha-proxy reverse proxy management (boot / reboot / start / stop / restart / remove / logs / details / services) |
| `conoha config` | CLI configuration (show / set / path) |
| `conoha skill` | Claude Code skill management (install / update / remove) |

## App Deploy

`conoha app` supports two deploy modes that can coexist on the same VPS. `conoha app init` writes a marker on the server (`/opt/conoha/<name>/.conoha-mode`), and every subsequent `deploy` / `status` / `logs` / `stop` / `restart` / `destroy` / `rollback` auto-detects the mode from it. Pass `--proxy` or `--no-proxy` to override; a mismatch with the marker is an error (destroy + re-init to switch modes).

| Mode | Default | When to use | Layout | `conoha.yml` | `conoha proxy boot` | DNS / TLS |
|---|:-:|---|---|:-:|:-:|:-:|
| **proxy** (blue/green) | ✓ | Public app with a domain + Let's Encrypt TLS | `/opt/conoha/<name>/<slot>/` blue/green slots | required | required | required |
| **no-proxy** (flat) |  | Testing, internal / dev VPS, non-HTTP services, hobby apps | `/opt/conoha/<name>/` flat single dir | n/a | n/a | n/a |

### proxy mode (default): conoha-proxy blue/green

[conoha-proxy](https://github.com/crowdy/conoha-proxy) provides Let's Encrypt HTTPS, Host-header routing, and instant rollback inside the drain window.

1. Create `conoha.yml` at your repo root:

   ```yaml
   name: myapp                   # DNS-1123 label (lowercase alnum + hyphen, 1-63 chars)
   hosts:
     - app.example.com           # one or more FQDNs, no duplicates
   web:
     service: web                # must match a service in the compose file
     port: 8080                  # container-side listen port (1-65535)
   # --- optional ---
   compose_file: docker-compose.yml   # auto-detected (conoha-docker-compose.yml → docker-compose.yml → compose.yml)
   accessories: [db, redis]           # sibling services that join the same network
   health:
     path: /healthz
     interval_ms: 1000
     timeout_ms: 500
     healthy_threshold: 2
     unhealthy_threshold: 3
   deploy:
     drain_ms: 5000                   # drain window before tearing down the old slot (milliseconds; default 30000 if omitted)
   ```

2. Boot the proxy container on the VPS:

   ```bash
   conoha proxy boot my-server --acme-email ops@example.com
   ```

   Skipping this step and going straight to `app init` fails with an Admin API socket error — the proxy container is not yet running.

3. Point the DNS A record at the VPS (Let's Encrypt HTTP-01 validation needs it). DNS must resolve by the time `app init` registers the host — if it doesn't, the `app`-layer deploy itself still succeeds but the hostname serves invalid certs until ACME eventually succeeds.

4. Register with the proxy and deploy:

   ```bash
   conoha app init my-server
   conoha app deploy my-server
   ```

5. Rollback (drain window only — instant swap back to the previous slot):

   ```bash
   conoha app rollback my-server
   ```

`deploy --slot <id>` pins the slot ID (rule: `[a-z0-9][a-z0-9-]{0,63}`; default is git short SHA or timestamp). Explicitly reusing an existing slot ID purges its work dir before re-extracting. When `--slot` is omitted and the default collides with an existing compose project (e.g. a still-draining previous slot), the CLI auto-suffixes with `-2`, `-3`, ... so a collision is never destructive.

### no-proxy mode: flat single-slot

Shortest path: no `conoha.yml`, no proxy, no DNS required. A `docker-compose.yml` is enough. This is equivalent to `docker compose up -d --build` over SSH and is the right choice when you do not need TLS or Host-based routing (testing, internal tools, non-HTTP services, hobby deployments).

```bash
# Initialize (verifies Docker / Compose are installed and writes the marker; does not install anything — pre-install via e.g. `conoha server create --user-data ./install-docker.sh`)
conoha app init my-server --app-name myapp --no-proxy

# Deploy (tar current dir → upload → extract to /opt/conoha/myapp/ → docker compose up -d --build)
conoha app deploy my-server --app-name myapp --no-proxy
```

Subsequent `status` / `logs` / `stop` / `restart` / `destroy` auto-detect no-proxy mode from the server marker, so you do not need to repeat `--no-proxy` (passing it again is allowed and is a no-op):

```bash
conoha app status my-server --app-name myapp
conoha app logs my-server --app-name myapp --follow
conoha app destroy my-server --app-name myapp
```

`rollback` is not available in no-proxy mode (there is no blue/green swap; invoking it prints a "rollback is not supported in no-proxy mode" error and exits). To revert, check out the previous commit and redeploy: `git checkout <sha> && conoha app deploy --no-proxy --app-name <app> <server>`.

### Switching modes

Destroy, then re-init in the opposite mode:

```bash
conoha app destroy my-server --app-name myapp            # removes marker and work dir
conoha app init my-server --app-name myapp --no-proxy    # re-initialize in the other mode
```

Different `<app-name>`s on the same VPS can run in different modes side by side.

### Key flags

| Flag | Command | Description |
|---|---|---|
| `--app-name <name>` | Always on `destroy` / `status` / `logs` / `stop` / `restart` / `env`; required on `init` / `deploy` / `rollback` only when used with `--no-proxy` | App name. Interactively prompted when omitted on a TTY; must be specified in non-TTY environments |
| `--proxy` / `--no-proxy` | lifecycle cmds (not `list`) | on `init`, selects the mode to write into the marker; on every other cmd, overrides the marker (mismatch is an error) |
| `--slot <id>` | `deploy` | Pin the slot ID (proxy mode only) |
| `--drain-ms <ms>` | `rollback` | Override the rollback drain window (0 = proxy default) |
| `--follow` / `-f` | `logs` | Stream in real time |
| `--service <name>` | `logs` | Restrict to one service |
| `--tail <n>` | `logs` | Line count (default 100) |
| `--data-dir <path>` | proxy-facing cmds | Server-side proxy data dir (default `/var/lib/conoha-proxy`) |

### Environment variables (no-proxy mode)

Server-side env vars persist across deploys. `conoha app env set` works in both modes and writes to `/opt/conoha/<app>.env.server` on the server, **but proxy-mode deploy does not currently merge that file into the slot's `.env`** — running `app env set` against a proxy app prints `warning: app env has no effect on proxy-mode deployed slots; see #94 for the redesign` ([#94](https://github.com/crowdy/conoha-cli/issues/94) tracks the redesign). In proxy mode, pass app config via `environment:` / `env_file:` in your compose file for now.

```bash
conoha app env set my-server --app-name myapp DATABASE_URL=postgres://...
conoha app env list my-server --app-name myapp
conoha app env get my-server --app-name myapp DATABASE_URL
conoha app env unset my-server --app-name myapp DATABASE_URL
```

At deploy time (no-proxy mode only), `.env` is assembled as **repo-committed `.env` first, then `/opt/conoha/<app>.env.server` appended**, so `app env set` values win via last-occurrence semantics. Proxy mode does not perform this merge.

## Claude Code Skill

ConoHa CLI includes infrastructure automation skills for Claude Code. Once installed, you can use natural language to manage infrastructure.

### Installation

```bash
# Install the skill
conoha skill install

# Update the skill
conoha skill update

# Remove the skill
conoha skill remove
```

### Usage

Simply give instructions in Claude Code, and the skill will be triggered automatically:

```
> Create a server on ConoHa
> Set up a k8s cluster
> Deploy an app
```

### Recipe List

| Recipe | Description |
|--------|-------------|
| Docker Compose App Deploy | Deploy containerized apps via `conoha app deploy` |
| Custom Script Deploy | Server provisioning with startup scripts |
| Kubernetes Cluster | k3s cluster setup (coming soon) |
| OpenStack Platform | DevStack platform setup (coming soon) |
| Slurm HPC Cluster | Slurm HPC cluster setup (coming soon) |

See [conoha-cli-skill](https://github.com/crowdy/conoha-cli-skill) for details.

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

Apache License 2.0 - See [LICENSE](LICENSE) for details.
