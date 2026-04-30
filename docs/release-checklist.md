# Release checklist

Hand-run smoke before tagging `v0.2.0` (and every subsequent RC unless the
section is explicitly waived). The CI `e2e` job (spec
[`2026-04-23-e2e-tests-design.md`](superpowers/specs/2026-04-23-e2e-tests-design.md))
gates the CLI ↔ proxy contract in DinD; this checklist covers the bug
classes DinD cannot reproduce — documented in §7.1 of that spec.

> **Quick path (v0.6.0+).** The conoha-cli-skill recipe
> [`recipes/release-smoke.md`](https://github.com/crowdy/conoha-cli-skill/blob/main/recipes/release-smoke.md)
> automates §1 and §2 row 7 against a real VPS with active failure-mode
> assertions (healthy-gate, UFW status, LE issuer, status from /tmp,
> two-path rollback, residue-free teardown). It was developed during
> PR #162 row 7 verification and is the recommended way to run the
> smoke. Hand-run §2 rows 1–6 and 8 below for cases the recipe does
> not script (TOFU, ACME quota, DNS propagation).

## Prerequisites

- `gh auth status` succeeds (release is published via GitHub).
- A clean ConoHa account usable for the release: tenant ID + API user + key pair.
- A test domain you control with DNS write access (`conoha dns record add`),
  OR no domain at all — the skill recipe falls back to `<host>.<vps-ip>.sslip.io`
  to avoid LE quota and DNS propagation lag.
- Local `conoha` built from the RC commit: `go build -o bin/conoha ./`.

## 1. Fresh-VPS smoke (spec §7 golden path)

Run on the RC commit, from a workstation with the built binary. Either
follow [`recipes/release-smoke.md`](https://github.com/crowdy/conoha-cli-skill/blob/main/recipes/release-smoke.md)
end-to-end (recommended), or run the steps below by hand.

1. **Create a VPS.** `bin/conoha server create --no-input --yes --wait --flavor g2l-t-c3m2 --image vmi-docker-29.2-ubuntu-24.04-amd64 --name rc-smoke-$(date +%s) --key-name <yours> --security-group default --security-group IPv4v6-SSH --security-group IPv4v6-Web --security-group IPv4v6-ICMP`.
   **v0.7.1+:** the four flags above are equivalent to `--for proxy` (preset
   added in #184).
2. **Point DNS** *(optional — skip if using sslip.io fallback).*
   `bin/conoha dns record add <zone> <host> A <vps-ip>` and wait until
   `dig +short <host>.<zone>` returns the IP (1–5 min).
3. **Boot the proxy.** `bin/conoha proxy boot --acme-email you@example.com <server>`
   → confirm **stderr** contains `==> Waiting for proxy to become healthy`
   (the #177 healthy-gate; if absent, the CLI predates v0.7.0 and is not
   the binary you intend to release) followed by `Boot complete.` Both
   lines are written to stderr — capture with `2>&1` if you want them in
   a single log.
4. **Init + deploy a fixture app** (reuse `tests/e2e/fixtures/` or a real
   project with a real `Host:`). `bin/conoha app init <server>` then
   `bin/conoha app deploy <server>`.
5. **Hit the site over HTTPS in a browser.** Expect HTTP 200 and a valid
   Let's Encrypt **production** certificate (issuer `R10`–`R14` for RSA
   chains or `E5`–`E8` for ECDSA — the LE prod intermediates rotate, so
   any current production issuer is fine). An issuer containing
   `staging.letsencrypt.org` (e.g. `(STAGING) Counterfeit Cas E1`)
   indicates the proxy is in staging mode and the row fails.
6. **Tear down.** `bin/conoha app destroy --yes <server>`,
   `bin/conoha proxy remove --purge <server>`, then
   `bin/conoha server delete --yes --delete-boot-volume --wait <server>`.
   `--delete-boot-volume` is the **opt-in** flag added in #88 — without
   it the boot volume is retained after the server is gone and will
   block future server creation with the same name on quota-tight
   tenants. Then remove the DNS record if step 2 was used.

If any step fails, do **not** tag. File the bug against the RC commit and
iterate on an `rcN+1` tag.

## 2. VPS-only bug checks (spec §7.1)

For each row: what you observe on this RC, compared to what the spec says
should happen. Tick the `Done?` column (`[x]` = passed) for every row;
leave `[ ]` and note the reason in the release notes if you waive one
(e.g. ACME is rate-limited today) or it fails.

The **Coverage** column says where the check lives:
- *recipe* — covered by [`recipes/release-smoke.md`](https://github.com/crowdy/conoha-cli-skill/blob/main/recipes/release-smoke.md). One end-to-end run satisfies the row.
- *CLI guard* — the CLI now refuses or warns on the failure mode (regression-tested in CI). Run the recipe to confirm the guard fires.
- *hand-only* — recipe does not script this; needs deliberate setup the recipe avoids.

| # | Bug class | What to run | Pass condition | Coverage | Done? |
|---|-----------|-------------|----------------|----------|-------|
| 1 | **cloud-init timing** | `bin/conoha server create …` immediately followed by `bin/conoha proxy boot <server>` (no manual sleep). | `proxy boot` waits for cloud-init to complete before SSH-ing, then succeeds on first try. No `Connection refused`/`timeout` errors surface to the user. The healthy-gate (#177) prints `==> Waiting for proxy to become healthy` followed by `Boot complete.` to **stderr** (use `2>&1` when piping to a file). | recipe (steps 3, 5) + CLI guard (#177) | `[ ]` |
| 2 | **systemd unit race** | `ssh <server> 'systemctl status docker conoha-proxy'` after step 1.3. Confirm UFW (`ufw status verbose`) allows 80/443 and `/etc/sysctl.d/99-conoha-proxy.conf` exists. | Both units `active (running)`. `journalctl -u conoha-proxy` shows no `failed to connect to docker.sock` retries. UFW allows 80,443/tcp; sysctl `net.ipv4.ip_unprivileged_port_start=0` set. (#173/#174 apply both at boot.) | recipe (step 5) + CLI guard (#173/#174) | `[ ]` |
| 3 | **SSH known_hosts TOFU (#101)** | Connect from a fresh workstation (empty `~/.ssh/known_hosts`). Run `bin/conoha app status <server>` twice; between runs, `ssh-keygen -R <vps-ip>` to simulate a rebuild. | First run prompts for TOFU (or auto-accepts with `CONOHA_SSH_INSECURE=1`); second run (post-remove) re-prompts rather than silently pinning an old fingerprint. | hand-only — recipe uses `--insecure`, which masks TOFU semantics by design. | `[ ]` |
| 4 | **ACME rate-limit degradation** | Trigger `app deploy` against a domain already past the LE weekly quota (reuse a known-throttled zone, or skip with reason). | CLI output ends with `TLS pending — degraded` rather than a plain-looking success. Proxy keeps serving HTTP. | hand-only — recipe uses sslip.io to *avoid* quota; the throttled-quota path needs a separate setup. | `[ ]` |
| 5 | **DNS propagation** | `bin/conoha app init <server>` against a host whose A record was added <60s ago (before full NS propagation). | CLI either warns `hosts not yet reachable — deploy may fail until DNS propagates` or fails early with a clear message; no silent-succeed-then-502-later behavior. | hand-only — recipe sidesteps via sslip.io. | `[ ]` |
| 6 | **ConoHa API response drift** | `bin/conoha flavor list`, `bin/conoha image list`, `bin/conoha keypair list` all against live API. **+v0.7.0:** `bin/conoha dns domain list` and `bin/conoha dns record list <zone>` (PR #180 added `uuid` field acceptance — pre-#180 binaries fail here). | All five return non-empty tables (or empty for DNS if the tenant has no zones, but no parse error) with no `unexpected field` / `missing required` parse errors. If any parser is brittle, catch it here rather than during a `server create` mid-release. | recipe (pre-flight + step 1) + CLI guard (#180) | `[ ]` |
| 7 | **multi-host / expose blocks (v0.3.0+)** | Deploy a sample with an `expose:` block (recipe step 4 stages a minimal root + `expose: api` fixture on sslip; for hand-run reuse a real two-host project like `gitea` + `dex`). Run `init`, `deploy`, then `app status --format json <server>` and `app rollback <server>` (no `--target`). **+v0.7.0:** also run `app status --app-name <name> <server>` from a directory **without** `conoha.yml` (#178); confirm stderr warns `no conoha.yml in cwd` and stdout is valid JSON with `root` populated. | `init` upserts root + `<name>-<label>` proxy services. `deploy` swaps every block (expose first, root last) and the sub-host responds 200 over HTTPS. `status --format json` returns `{root, expose: [{label, service}]}` with both services populated. `status` from /tmp returns root only + stderr warning. `rollback` (no flag) hits `/rollback` N+1 times in reverse order; a 409 on one block prints `warning: drain window expired for <svc>; skipping` and the others still roll back. `--target=<label>` on a closed window is fatal. | recipe (steps 4, 6, 8) + CLI guard (#178 status no-yml) | `[ ]` |
| 8 | **`app env set` interpolation (#166/#179)** | Sample whose `compose.yml` uses `${KEY:-default}` interpolation in an `environment:` block. `bin/conoha app env set <server> KEY=<rand>`, then `bin/conoha app deploy <server>`, then exec into the container and `printenv KEY`. | Container env shows `<rand>`, not `default`. The CLI passes `--env-file /opt/conoha/<app>/.env.server` to compose so interpolation and `env_file:` agree (#179). Pre-#179 binaries silently get `default` and the user-set value never reaches the container. | hand-only — recipe's nginx fixture has no interpolation; needs a sample with `${KEY:-default}` to exercise. Cover during a release where samples that use interpolation (e.g. gitea) get a deploy. | `[ ]` |

Link the run output (asciinema / terminal log / screenshots / recipe report
template at `recipes/release-smoke.md` § 結果レポーティング) in the release
notes.

## 3. Promotion gate

If §1 is green and every row in §2 is either ✓ or justified-waived:

1. **Push the RC tag.** `git tag -s v0.x.0-rcN -m "<release-notes>" && git push origin v0.x.0-rcN`.
   The signed tag's annotation body becomes the editable release-notes
   draft. **`goreleaser` skips Homebrew/Scoop publishing for RC tags**
   (per `skip_upload: auto` set in #183), so the RC binary is available
   only as the GitHub pre-release.
2. **Wait for `CI / Release` to pass on the tag.** The workflow uploads
   binaries + checksums to the GitHub release; brew/scoop are deliberately
   *not* updated yet.
3. **Soak the RC.** Run against a pre-prod ConoHa account under realistic
   load for **24h** if pre-prod exists, or **1 week wall-clock** otherwise.
   Treat regressions surfaced during soak as the bar for tagging an
   `rcN+1`. If you shorten the soak for a time-sensitive release, record
   the reason in the release notes.
4. **Cut the final tag.** `git tag -s v0.x.0 -m "<final notes>" && git push origin v0.x.0`.
   The same `Release` workflow re-runs and (because the tag is now non-RC)
   publishes Homebrew tap + Scoop bucket via goreleaser.
5. **Curate the GitHub release body.** `gh release edit v0.x.0` and replace
   the auto-generated changelog with the curated body — typically the
   final-tag annotation message, plus a link back to the smoke run.
6. **Merge the banner-bump PR.** The `Release` workflow's `update-banner`
   job opens `chore(banner): bump version pill to v0.x.0` against `main`
   after goreleaser succeeds. It edits only `banner.svg` (the version
   pill rendered in the README header). Squash-merge it; no review
   needed. If you cut a release out-of-band, run `make banner-version
   V=v0.x.0` locally and commit instead.

## 4. Next release deltas

When adding a new CLI surface or VPS-side behavior, update the §2 table
**before** cutting the tag. DinD cannot catch these classes, so this
document is where the coverage lives. Recent additions:

- **v0.7.0** — row 6 picked up `dns domain/record list` (#180), row 7
  picked up `app status` from /tmp (#178), row 8 added for env-file
  interpolation (#166/#179). Rows 1 and 2 became CLI-guarded by
  #173/#174 (host-level UFW + sysctl applied at proxy boot) and #177
  (proxy boot healthy gate); the hand-run is now confirmation that the
  guard fires, not the only line of defense.
- **v0.7.1+** — `--for proxy` preset (#182/#184) collapses the four
  `--security-group` flags + UUID flavor/image into one `--for proxy`.
  No new bug class — preserves the existing §1 step 1 surface as a
  shorthand. If a future preset adds new bug-class exposure (e.g.
  `--for k8s-master` involves a private network that ACME cannot reach),
  add a row here.
