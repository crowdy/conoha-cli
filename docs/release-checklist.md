# Release checklist

Hand-run smoke before tagging `v0.2.0` (and every subsequent RC unless the
section is explicitly waived). The CI `e2e` job (spec
[`2026-04-23-e2e-tests-design.md`](superpowers/specs/2026-04-23-e2e-tests-design.md))
gates the CLI ↔ proxy contract in DinD; this checklist covers the bug
classes DinD cannot reproduce — documented in §7.1 of that spec.

## Prerequisites

- `gh auth status` succeeds (release is published via GitHub).
- A clean ConoHa account usable for the release: tenant ID + API user + key pair.
- A test domain you control with DNS write access (`conoha dns record add`).
- Local `conoha` built from the RC commit: `go build -o bin/conoha ./`.

## 1. Fresh-VPS smoke (spec §7 golden path)

Run on the RC commit, from a workstation with the built binary.

1. **Create a VPS.** `bin/conoha server create --flavor <smallest> --image vmi-docker-29.2-ubuntu-24.04-amd64 --name rc-smoke-$(date +%s)`
   Record the server name.
2. **Point DNS.** `bin/conoha dns record add <zone> <host> A <vps-ip>` and
   wait until `dig +short <host>.<zone>` returns the IP (can take 1-5 min).
3. **Boot the proxy.** `bin/conoha proxy boot --acme-email you@example.com <server>`
   → confirm stdout reports `proxy ready` and `/readyz` returns 200 over SSH.
4. **Init + deploy a fixture app** (reuse `tests/e2e/fixtures/` or a real
   project with a real `Host:`). `bin/conoha app init <server>` then
   `bin/conoha app deploy <server>`.
5. **Hit the site over HTTPS in a browser.** Expect HTTP 200 and a valid
   Let's Encrypt certificate (issuer: `R3` / `E1` or the current LE chain).
6. **Tear down.** `bin/conoha app destroy --yes <server>`,
   `bin/conoha proxy remove --purge <server>`, then
   `bin/conoha server delete <server>` and
   `bin/conoha dns record remove <zone> <host> A`.

If any step fails, do **not** tag. File the bug against the RC commit and
iterate on an `rcN+1` tag.

## 2. VPS-only bug checks (spec §7.1)

For each row: what you observe on this RC, compared to what the spec says
should happen. Tick the `Done?` column (`[x]` = passed) for every row;
leave `[ ]` and note the reason in the release notes if you waive one
(e.g. ACME is rate-limited today) or it fails.

| # | Bug class | What to run | Pass condition | Done? |
|---|-----------|-------------|----------------|-------|
| 1 | **cloud-init timing** | `bin/conoha server create …` immediately followed by `bin/conoha proxy boot <server>` (no manual sleep). | `proxy boot` waits for cloud-init to complete before SSH-ing, then succeeds on first try. No `Connection refused`/`timeout` errors surface to the user. | `[ ]` |
| 2 | **systemd unit race** | `ssh <server> 'systemctl status docker conoha-proxy'` after step 1.3. | Both units are `active (running)`. `journalctl -u conoha-proxy` shows no `failed to connect to docker.sock` retries. | `[ ]` |
| 3 | **SSH known_hosts TOFU (#101)** | Connect from a fresh workstation (empty `~/.ssh/known_hosts`). Run `bin/conoha app status <server>` twice; between runs, `ssh-keygen -R <vps-ip>` to simulate a rebuild. | First run prompts for TOFU (or auto-accepts with `CONOHA_SSH_INSECURE=1`); second run (post-remove) re-prompts rather than silently pinning an old fingerprint. | `[ ]` |
| 4 | **ACME rate-limit degradation** | Trigger `app deploy` against a domain already past the LE weekly quota (reuse a known-throttled zone, or skip with reason). | CLI output ends with `TLS pending — degraded` rather than a plain-looking success. Proxy keeps serving HTTP. | `[ ]` |
| 5 | **DNS propagation** | `bin/conoha app init <server>` against a host whose A record was added <60s ago (before full NS propagation). | CLI either warns `hosts not yet reachable — deploy may fail until DNS propagates` or fails early with a clear message; no silent-succeed-then-502-later behavior. | `[ ]` |
| 6 | **ConoHa API response drift** | `bin/conoha flavor list`, `bin/conoha image list`, `bin/conoha keypair list` all against live API. | All three return non-empty tables with no `unexpected field` / `missing required` parse errors. If any parser is brittle, catch it here rather than during a `server create` mid-release. | `[ ]` |
| 7 | **multi-host / expose blocks (v0.3.0+)** | Deploy a sample with an `expose:` block (e.g. `gitea` with a `dex` subdomain; two separate A records required). Run `bin/conoha app init`, `deploy`, then `app status --format json <server>` and `app rollback <server>` (no `--target`). | `init` upserts root + `<name>-<label>` proxy services. `deploy` swaps every block (expose first, root last) and the sub-host responds 200 over HTTPS. `status --format json` returns `{root, expose: [{label, service}]}` with both services populated. `rollback` (no flag) hits `/rollback` N+1 times in reverse order; a 409 on one block prints `warning: drain window expired for <svc>; skipping` and the others still roll back. | `[ ]` |

Link the run output (asciinema / terminal log / screenshots) in the release notes.

## 3. Promotion gate

If §1 is green and every row in §2 is either ✓ or justified-waived:

1. Push the RC tag: `git tag v0.2.0-rcN && git push origin v0.2.0-rcN`.
2. Wait for `CI / e2e` to pass on the tag.
3. Soak the RC tag against a pre-prod ConoHa account under realistic
   load for **24h** (or **1 week wall-clock** if no pre-prod environment
   is available). If no regressions emerge, cut the final tag:
   `git tag -s v0.2.0 && git push origin v0.2.0`. If you shorten the
   soak for a time-sensitive release, record the reason in the release
   notes.
4. Run `gh release create v0.2.0 --generate-notes` and edit the body to
   link back to this checklist run.

## 4. Next release deltas

When adding a new CLI surface or VPS-side behavior, update the §2 table
before cutting the tag. DinD cannot catch these classes, so this document
is where the coverage lives.
