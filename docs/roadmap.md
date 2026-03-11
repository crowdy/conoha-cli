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

---

## v0.1.8: `--wait` for Async Operations

- `--wait` / `--wait-timeout` -- wait for async operation completion
- server create (until ACTIVE), delete (until 404), start/stop/reboot
- volume create (until available)
- Extract existing `waitForVolume()` pattern into `cmdutil.WaitFor()` shared helper

---

## v0.1.9: Load Balancer CLI Completion

- listener create/show/delete
- pool create/show/delete
- member create/show/delete
- healthmonitor create/show/delete
- Add API/model + split cmd/lb/ files

---

## v0.2.0: Testing & CI/CD

- Increase unit test coverage (target 50%+)
- GitHub Actions CI: test + lint on push/PR
- goreleaser automated releases
