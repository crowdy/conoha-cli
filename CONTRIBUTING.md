# Contributing

## Secret-handling rules (must read)

This repository ships a CLI that talks to live ConoHa infrastructure.
To prevent leaking real credentials or internal endpoints, contributors
**must** follow these rules:

1. **Never commit a real secret.** No API tokens, passwords, application
   credentials, or session cookies — anywhere in the repo, including
   tests, fixtures, design docs, and CLI output examples.

2. **Mask infrastructure identifiers in committed text.** Replace real
   server IPs, VM IDs, hostnames, internal DNS names, and account IDs
   with placeholders such as `<SERVER_IP>`, `<TENANT_ID>`. Use the
   reserved RFC 5737 example ranges (`192.0.2.0/24`,
   `198.51.100.0/24`, `203.0.113.0/24`) for documentation IPs and
   `*.example.test` for hostnames.

3. **Test fixtures must use obviously-fake values.** Use `password: s3cr3t`,
   `token: dummy`, etc. Never copy a real response — even from a sandbox
   account — into a test fixture.

4. **Pre-publish drafts (Qiita, blog posts) belong outside git.**
   `docs/articles/qiita/` is gitignored so we don't accidentally publish
   work-in-progress drafts that still contain real IPs or identifiers.
   Mask values before sending to Qiita.

5. **Internal endpoints stay out of public docs and tests.** Anything
   matching `*.internal.*` or `*.gmo.*` is an internal endpoint and
   must be replaced with `https://staging.example.test` or similar in
   committed code.

6. **`credentials.yaml` and `tokens.yaml` are gitignored.** They live in
   `~/.config/conoha-cli/` on the developer's machine. If the CLI ever
   needs to write a sample of one of these files, name it
   `credentials.example.yaml`.

## How we enforce this

* **`gitleaks` runs on every PR** via GitHub Actions
  (`.github/workflows/gitleaks.yml`). PRs with detected secrets are
  blocked from merge.
* **`.gitignore` blocks the obvious patterns**: `credentials.yaml`,
  `tokens.yaml`, `.env`, `.env.*`, `*.pem`, `*.key`, `id_rsa*`,
  `docs/memory/`, `docs/articles/qiita/`.
* **Reviewers should grep PRs for IPs, hostnames, and known token
  prefixes** (`Bearer `, public IP ranges, `internal.`).

## If you accidentally commit a secret

1. **Treat it as already public.** Rotate it (revoke + reissue) at the
   source service before doing anything else.
2. After rotating, remove the value from `HEAD` in a follow-up commit.
3. To purge it from git history, use `git filter-repo` and force-push,
   then notify other contributors so they re-clone. Do this only after
   rotation.
