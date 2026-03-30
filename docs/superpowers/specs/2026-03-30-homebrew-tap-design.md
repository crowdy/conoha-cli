# Homebrew Tap Setup Design

## Date: 2026-03-30

## Goal

Enable `brew install crowdy/tap/conoha` for macOS and Linux users by setting up a Homebrew tap backed by GoReleaser automation.

## Problem

The docs site at `crowdy.github.io/conoha-cli-pages/guide/getting-started` references `brew install crowdy/tap/conoha`, but the `crowdy/homebrew-tap` repository does not exist, causing installation to fail.

## Design

### Components

1. **`crowdy/homebrew-tap` GitHub repo** (public) — Hosts Homebrew formula files. Created empty; GoReleaser populates it on first release.

2. **`.goreleaser.yaml` brews section** — Tells GoReleaser to generate a formula (`conoha.rb`) and push it to `crowdy/homebrew-tap` on each tagged release. Covers macOS (arm64, amd64) and Linux (amd64, arm64).

3. **`release.yml` workflow update** — Uses `HOMEBREW_TAP_TOKEN` secret (Fine-grained PAT) instead of the default `GITHUB_TOKEN`, since the default token cannot push to a different repository.

### Authentication

- **Fine-grained PAT** scoped to `crowdy/homebrew-tap` only
  - Permission: Contents → Read and write
  - Stored as repository secret `HOMEBREW_TAP_TOKEN` in `crowdy/conoha-cli`

### User Experience

```
brew install crowdy/tap/conoha
conoha --version
```

No PAT or authentication needed for end users — the tap repo and GitHub Releases are both public.

### Formula Details

- Name: `conoha`
- Description: CLI tool for ConoHa VPS3 API
- Homepage: https://crowdy.github.io/conoha-cli-pages/
- License: MIT
- Platforms: macOS (arm64, amd64), Linux (amd64, arm64)

### Manual Steps After Merge

1. Create Fine-grained PAT at GitHub Settings → Developer settings → Personal access tokens → Fine-grained tokens
   - Repository access: Only select `crowdy/homebrew-tap`
   - Permissions: Contents → Read and write
2. Add PAT as secret `HOMEBREW_TAP_TOKEN` in `crowdy/conoha-cli` → Settings → Secrets and variables → Actions
3. Create a new tagged release (e.g., `v0.3.3`) to trigger the first formula push
