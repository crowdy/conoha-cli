# conoha-cli Roadmap

## v0.1.5: Confirmation Prompts (Safety)

### Context

All destructive commands (delete, rebuild, resize) execute immediately without confirmation.
Add confirmation prompts to prevent accidental operations, plus `--yes`/`-y` flag for batch/script usage.

### `--yes`/`-y` Global Flag

Add persistent flag in `cmd/root.go`:
- `flagYes bool` variable, `--yes`/`-y` flag
- Set `CONOHA_YES=1` env var in `PersistentPreRun`

Add to `internal/config/env.go`:
- `EnvYes = "CONOHA_YES"` constant
- `IsYes() bool` function

### `prompt.Confirm()` Behavior Change

Modify `internal/prompt/prompt.go`:
1. `config.IsYes()` -> `return true, nil` (auto-confirm)
2. `config.IsNoInput()` -> error with hint: `"use --yes to auto-confirm"`
3. Otherwise -> existing interactive prompt

### Commands Requiring Confirmation

| Command | Confirmation Message |
|---------|---------------------|
| `server create` | Show creation summary then confirm (name, flavor, image, volume, key) |
| `server delete` | `Delete server "name" (ID)? This cannot be undone.` |
| `server rebuild` | `Rebuild server? All data will be lost.` |
| `server resize` | `Resize server? This may cause downtime.` |
| `volume delete` | `Delete volume "name" (ID)?` |
| `dns domain delete` | `Delete domain and all its records?` |
| `lb delete` | `Delete load balancer (ID)?` |

### Files to Modify

| File | Change |
|------|--------|
| `cmd/root.go` | `--yes`/`-y` flag, env var setup |
| `internal/config/env.go` | `EnvYes`, `IsYes()` |
| `internal/prompt/prompt.go` | `Confirm()` logic change |
| `cmd/server/server.go` | create/delete/rebuild/resize confirmation |
| `cmd/volume/volume.go` | delete confirmation |
| `cmd/dns/dns.go` | domain delete confirmation |
| `cmd/lb/lb.go` | delete confirmation |
| `internal/prompt/prompt_test.go` | Confirm() tests (--yes, --no-input) |

### Verification

1. `conoha server delete <id>` -> confirmation prompt, N cancels
2. `conoha server delete <id> --yes` -> delete immediately
3. `conoha server delete <id> --no-input` -> error + `--yes` hint
4. `conoha server create --name test` -> summary + confirm
5. `golangci-lint run ./...` + `go test ./...`

---

## v0.1.6: Remaining Confirmations + Code Splitting

### Additional Delete Confirmations

- keypair delete, image delete
- network delete, subnet delete, security-group delete
- dns record delete
- storage container delete, storage rm
- identity credential/subuser delete

### `server create` Keypair Selection

Currently `--key-name` flag exists but there is no interactive prompt when omitted.
- If `--key-name` not specified, list keypairs and let user select interactively
- Skip if no keypairs exist (proceed without key)

### `keypair create` Private Key Save

Currently private key is only returned in the create response but not saved.
- Save private key to file on creation (default: `~/.ssh/conoha_<name>`)
- `--output` / `-o` flag to specify output path
- Set file permissions to 0600
- Print saved path to stderr

### `server create` Startup Script (`user_data`)

ConoHa VPS3 supports startup scripts via the `user_data` parameter in server create API.
No API exists for managing/listing saved scripts -- users provide their own script files.

API details:
- Parameter: `server.user_data` (base64-encoded)
- Max size: 16 KiB (before encoding)
- Supported headers: `#!`, `#cloud-config`, `#cloud-boothook`, `#include`, `#include-once`
- Linux only (Windows Server not supported)

CLI flags:
- `--user-data <file>`: read file, validate size, base64-encode, send as `user_data`
- `--user-data-raw <string>`: encode string directly (for simple one-liners)
- `--user-data-url <url>`: wrap as `#include` directive and encode
- Error if > 16 KiB, warn if Windows flavor selected

Control panel supports 3 methods (CLI should match):
1. File (`--user-data`) -- equivalent to "テキスト入力"
2. Raw string (`--user-data-raw`) -- for one-liners
3. URL (`--user-data-url`) -- equivalent to "URL指定", wraps as `#include <url>`

References:
- https://support.conoha.jp/v/startupscript/
- https://vps.conoha.jp/function/startupscript/
- https://doc.conoha.jp/products/vps-v3/startupscripts-v3/

Model change: add `UserData string` to `ServerCreateRequest.Server`

### Split server.go (~806 lines -> 5-6 files)

- `server.go` -- Cmd, init(), helpers
- `list.go` -- listCmd, showCmd
- `create.go` -- createCmd, selectFlavor, selectImage, volume helpers
- `lifecycle.go` -- delete, start, stop, reboot, resize, rebuild, console
- `metadata.go` -- metadataCmd, ipsCmd
- `volume_attach.go` -- attach/detach volume

---

## v0.1.7: List UX Improvements

- `--filter key=value` -- filtering for list commands
- `--sort-by field` -- sorting for list commands
- `--no-headers` -- remove table headers (for scripting)
- `--no-color` actual implementation (flag exists but unused)
- Command aliases: `network sg` -> `security-group`, `network sgr` -> `security-group-rule`
- Human-readable byte sizes in table output (e.g. `1.4 GB` instead of `1538800161`), applies to `storage container list` etc.
- `image list`: add visibility column (public/private) to output
- `storage publish`: show public URL after publishing (e.g. `https://object-storage.c3j1.conoha.io/v1/AUTH_{tenant}/{container}`)
- `storage cp --recursive` / `-r`: upload/download directories recursively (currently single file only)

### `server show` Enhancements

#### Volume Info

`conoha server show <id>` output should include attached volume information (at minimum: volume ID, size).
- Call volume attachment API (`GET /servers/{id}/os-volume_attachments`) to get attached volume IDs
- Call volume detail API (`GET /volumes/{id}`) for each to get size
- Display in server show output: volume ID, size (GB), device path

#### Port / IP Info

Show port and IP information alongside addresses.
- Call port list API (`GET /ports?device_id={server_id}`) to get ports attached to the server
- Display: IP address, port ID, MAC address, security groups

---

## v0.1.8: `--wait` for Async Operations

- `--wait` / `--wait-timeout` -- wait for async operation completion
- server create (until ACTIVE), delete (until 404), start/stop/reboot
- volume create (until available)
- Extract existing `waitForVolume()` pattern into `cmdutil.WaitFor()` shared helper

### `server ssh` Command

SSH into a server via system `ssh` command (like `gcloud compute ssh`, `az ssh vm`).
- Get server IP and key_name from server detail API
- Resolve private key path (`~/.ssh/conoha_<key_name>`, requires v0.1.6 keypair save)
- Execute system `ssh` via `os.Exec`

```
conoha server ssh <server>              # ssh root@<ip> -i ~/.ssh/conoha_<key>
conoha server ssh <server> "ls -la"     # remote command execution
conoha server ssh <server> -l ubuntu    # specify user
conoha server ssh <server> -p 2222      # specify port
```

Flags: `--user` / `-l`, `--port` / `-p`, `--identity` / `-i` (override key path)

---

## v0.1.9: Load Balancer CLI Completion + Image Upload

- listener create/show/delete
- pool create/show/delete
- member create/show/delete
- healthmonitor create/show/delete
- Add API/model + split cmd/lb/ files
- `image upload` command (API details TBD)

---

## v0.2.0: Testing & CI/CD

- Increase unit test coverage (target 50%+)
- GitHub Actions CI: test + lint on push/PR
- goreleaser automated releases

---

## v0.2.1: `server deploy` Command

SSH-based deployment command, built on top of `server ssh` (v0.1.8).
Designed as a simple building block that AI agents can compose for full deployment pipelines.

```
conoha server deploy <server> --script deploy.sh
conoha server deploy <server> --script deploy.sh --env APP_ENV=production
```

- Connect to server via SSH (reuse `server ssh` infrastructure)
- Upload and execute a deployment script on the remote server
- `--script <file>`: local script to upload and run
- `--env KEY=VALUE`: pass environment variables to the script (repeatable)
- Stream stdout/stderr in real-time
- Exit with remote script's exit code

### AI Agent Workflow Example

An AI agent can compose the full pipeline using existing commands:
1. `conoha server create --user-data init.sh` -- create VPS with initial setup
2. `conoha server ssh <server> "apt install -y docker.io"` -- install dependencies
3. `conoha server deploy <server> --script deploy.sh` -- deploy application

No need for built-in app detection or framework support -- the agent generates
appropriate scripts based on project type (Rails, Laravel, Django, static, etc.).

References:
- Kamal: Docker + SSH deploy (https://kamal-deploy.org/)
- Dokku: git push-based self-hosted PaaS (https://dokku.com/)
- az webapp up: simplest PaaS deploy (one command)
- gcloud run deploy --source: auto-buildpack + container
