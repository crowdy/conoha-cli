# v0.2.0 Design: Testing & CI/CD

## Overview

Add GitHub Actions CI/CD pipelines and expand unit test coverage from 18.7% to 50%+. CI runs on push to main and PRs. Release pipeline uses goreleaser on tag push. Test coverage targets the API layer (httptest-based) as the highest-ROI area.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Coverage strategy | API layer first | Largest untested codebase area (~30% of code), httptest pattern already exists |
| CI triggers | push main + PR | Personal project, simple and sufficient |
| Release platforms | 6 builds (existing) | linux/darwin/windows × amd64/arm64, `.goreleaser.yaml` already configured |
| Spec scope | Single spec | CI/Release are small (2 YAML files), tests are the bulk |
| Implementation order | CI → Release → Tests | CI validates test additions as they land |

## 1. GitHub Actions CI

**File:** `.github/workflows/ci.yml`

Triggers: push to `main`, pull requests to `main`.

Three independent jobs:

```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - run: go test ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - uses: golangci/golangci-lint-action@v6

  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - run: go build -o /dev/null ./...
```

## 2. goreleaser Release Workflow

**File:** `.github/workflows/release.yml`

Triggers: push tags matching `v*`.

```yaml
name: Release
on:
  push:
    tags: ['v*']

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - uses: goreleaser/goreleaser-action@v6
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- `fetch-depth: 0` — goreleaser needs full git history for changelog generation
- `permissions: contents: write` — required to create GitHub Releases
- Uses existing `.goreleaser.yaml` config (6 platform builds, checksums, changelog)

## 3. Unit Tests — API Layer (httptest)

### Current State

- Overall coverage: 18.7% (99 tests, 17 test files)
- Well-tested: `internal/model` (100%), `internal/output` (82.6%), `internal/config` (68.2%)
- Untested: Most `internal/api/` endpoints (10.4%), all `cmd/` command handlers (0%)

### Test Pattern

Extend the existing `client_test.go` httptest pattern. Each API service gets its own test file.

```go
func TestListServers(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != "GET" || r.URL.Path != "/v2.1/servers/detail" {
            t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
        }
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprint(w, `{"servers":[{"id":"xxx","name":"test"}]}`)
    }))
    defer ts.Close()

    client := &Client{HTTP: ts.Client(), Token: "tok", TenantID: "tid"}
    t.Setenv("CONOHA_ENDPOINT", ts.URL)

    servers, err := NewComputeAPI(client).ListServers()
    if err != nil {
        t.Fatal(err)
    }
    if len(servers) != 1 || servers[0].Name != "test" {
        t.Errorf("unexpected: %+v", servers)
    }
}
```

Each test verifies:
- HTTP method (GET/POST/PUT/DELETE)
- URL path (correct endpoint)
- Request body (POST/PUT — JSON structure, field names)
- Response parsing (JSON → model struct mapping)
- Error cases (404 → NotFoundError, 401 → AuthError, 500 → APIError)

### Test Files to Create

| File | Tests For | Method Count | Priority |
|------|-----------|-------------|----------|
| `internal/api/compute_test.go` | ListServers, GetServer, CreateServer, DeleteServer, StartServer, StopServer, RebootServer, ResizeServer, RebuildServer, GetConsoleURL, ListAddresses, AttachVolume, DetachVolume | ~15 | 1 |
| `internal/api/loadbalancer_test.go` | All LB CRUD + sub-resources (Get/List/Create/Delete for LB, Listener, Pool, Member, HealthMonitor) | ~18 | 2 |
| `internal/api/volume_test.go` | ListVolumes, GetVolume, CreateVolume, DeleteVolume, ListVolumeTypes, ListBackups, GetBackup, RestoreBackup | ~8 | 3 |
| `internal/api/dns_test.go` | ListDomains, GetDomain, CreateDomain, DeleteDomain, ListRecords, CreateRecord, DeleteRecord | ~7 | 4 |
| `internal/api/network_test.go` | ListNetworks, ListSubnets, ListPorts, ListSecurityGroups, ListSecurityGroupRules, CreateSecurityGroupRule, DeleteSecurityGroupRule, DeleteSecurityGroup | ~8 | 5 |
| `internal/api/image_test.go` | ListImages, GetImage, DeleteImage, CreateImage, UploadImageFile | ~5 | 6 |
| `internal/api/objectstorage_test.go` | ListContainers, CreateContainer, DeleteContainer, ListObjects, UploadObject, DeleteObject | ~6 | 7 |
| `internal/api/identity_test.go` | ListCredentials, CreateCredential, DeleteCredential, ListSubUsers | ~4 | 8 |
| `internal/api/auth_test.go` | Authenticate, EnsureToken (with token caching) | ~3 | 9 |

### Coverage Target

| Area | Current | After Tests | Notes |
|------|---------|-------------|-------|
| `internal/api/` | 10.4% | ~80% | httptest for all endpoints |
| `internal/config/` | 68.2% | ~80% | Fill gaps in Set() methods |
| `internal/prompt/` | 23.0% | ~40% | Expand Confirm/Select edge cases |
| `cmd/cmdutil/` | 33.3% | ~50% | NewClient, GetFormat helpers |
| **Overall** | **18.7%** | **~50%** | Target met |

API layer is ~30% of total codebase LOC. Getting it to 80% coverage brings overall to ~40%. Supplementing with config/prompt/cmdutil gap-filling reaches 50%.

### Supplementary Test Additions

Beyond the API layer, fill gaps in already-partially-tested packages:

- `internal/config/`: Add tests for `CredentialsStore.Set()`, `TokenStore.Set()`, `Config.Set()`
- `internal/prompt/`: Add edge cases for `Confirm()` with `--yes` flag, `Select()` with `--no-input`
- `cmd/cmdutil/`: Add tests for `GetFormat()` with env/flag/config precedence

## 4. Verification Plan

1. `go test ./...` — all tests pass
2. `go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1` — overall ≥ 50%
3. `golangci-lint run ./...` — 0 issues
4. GitHub Actions: push to PR, verify CI runs test/lint/build
5. Tag + push: verify goreleaser creates release (can test with `v0.2.0-rc1`)

## 5. Commit Strategy

| # | Scope | Description |
|---|-------|-------------|
| 1 | CI | Add `.github/workflows/ci.yml` |
| 2 | Release | Add `.github/workflows/release.yml` |
| 3 | Tests | Add compute API tests |
| 4 | Tests | Add loadbalancer API tests |
| 5 | Tests | Add volume API tests |
| 6 | Tests | Add dns API tests |
| 7 | Tests | Add network API tests |
| 8 | Tests | Add image API tests |
| 9 | Tests | Add objectstorage + identity API tests |
| 10 | Tests | Add auth API tests |
| 11 | Tests | Supplement config/prompt/cmdutil tests |
| 12 | Verify | Coverage check, fix any issues |

Single PR: `v0.2.0: Testing & CI/CD`
