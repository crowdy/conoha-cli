# ConoHa CLI Specification

**Version**: 0.1.0
**Status**: Initial Release

## Scope

ConoHa VPS3 API の全エンドポイントに対応する CLI ツール。

## Supported APIs

| Service | Endpoint Prefix | Commands |
|---------|----------------|----------|
| Identity | `identity.{region}.conoha.io/v3` | auth, identity |
| Compute | `compute.{region}.conoha.io/v2.1` | server, flavor, keypair |
| Block Storage | `block-storage.{region}.conoha.io/v3` | volume |
| Image | `image.{region}.conoha.io/v2` | image |
| Network | `networking.{region}.conoha.io/v2.0` | network, lb |
| DNS | `dns-service.{region}.conoha.io/v1` | dns |
| Object Storage | `object-storage.{region}.conoha.io/v1` | storage |

## Auth Flow

1. `POST /v3/auth/tokens` → X-Subject-Token (24h TTL)
2. Token cached in `~/.config/conoha/tokens.yaml`
3. Auto-refresh 5 min before expiry

## Output Contract

- `--format json`: stdout = JSON only, stderr = progress/warnings
- Exit codes: 0=ok, 1=general, 2=auth, 3=not-found, 4=validation, 5=api, 6=network, 10=cancelled

## Version History

| Version | Date | Description |
|---------|------|-------------|
| 0.1.4 | TBD | Version output branding (see below) |
| 0.1.3 | 2026-03-10 | flavor list UX, CONOHA_ENDPOINT, CONOHA_ENDPOINT_MODE=int, debug logging (see below) |
| 0.1.2 | 2026-03-10 | Bug fixes and feature improvements (see below) |
| 0.1.1 | 2026-03-10 | UX improvements (see below) |
| 0.1.0 | 2026-03-10 | Initial implementation - all API endpoints |

### 0.1.4 Changes (planned)

#### 1. `version`: Add author and unofficial notice

**Current**:
```
conoha version v0.1.3
```

**After**:
```
conoha version v0.1.4 by crowdy@gmail.com
This is an unofficial tool and is not affiliated with or endorsed by ConoHa/GMO Internet Group.
```

**Modified files**:
- `cmd/version.go` — update version output format

### 0.1.3 Changes

#### 1. `flavor list`: Sort by CPU/memory instead of ID

**Current**: Sorted by flavor ID (UUID), which is meaningless to users.

**After**: Sort by VCPUS (ascending), then RAM (ascending) as secondary key.

**Modified files**:
- `cmd/flavor/flavor.go` — `listCmd` に sort ロジック追加

#### 2. `flavor list`: Human-readable memory display

**Current**: RAM is displayed in raw MB (e.g., `32768`, `131072`).

**After**: RAM is displayed in human-readable format.

| Raw (MB) | Display |
|----------|---------|
| 512 | 512M |
| 1024 | 1G |
| 2048 | 2G |
| 16384 | 16G |
| 32768 | 32G |
| 98304 | 96G |
| 131072 | 128G |

Conversion rule: if `MB % 1024 == 0` → display as `{MB/1024}G`, else display as `{MB}M`.

**Modified files**:
- `cmd/flavor/flavor.go` — `listCmd` の RAM 表示をフォーマット

#### 3. `flavor list`: Add DISK column with explanation

**Current**: DISK column shows raw number. Value `0` is confusing — users don't know what it means.

**After**: DISK column displays human-readable value with context:
- `0` → `0` (VPS storage is network-attached, not local disk)
- Non-zero values (e.g., `40`, `250`) → `40G`, `250G` (local disk, `g2d-*` dedicated disk flavors only)

The DISK column represents **local disk** size in GB. Most flavors (`g2l-*`, `g2w-*`) use network-attached
block storage (Cinder volumes) and show `0` for local disk. Only `g2d-*` (dedicated disk) flavors have
local disk attached.

**Modified files**:
- `cmd/flavor/flavor.go` — DISK カラムのフォーマット

#### 4. `flavor list`: Abuse restriction notice footer

After the table output, print a notice to stderr about potential flavor restrictions:

```
Note: Some flavors may be restricted to prevent abuse. If you cannot use a flavor,
please contact ConoHa support: https://www.conoha.jp/conoha/contact/
```

- Message is always in English (international users: Korean, Indian, etc.)
- Output to **stderr** (not stdout) so it doesn't break `--format json` output
- Only shown in `table` format (not json/yaml/csv)

**Modified files**:
- `cmd/flavor/flavor.go` — `listCmd` の出力後に stderr へ注意メッセージ追加

#### Expected output after changes

```
VCPUS  RAM    DISK  ID                                    NAME
1      2G     40G   1488a6c7-6c8f-4e39-a522-ccab7d3acc84  g2d-t-c1m2d30
2      1G     0     09efe5d4-725a-4027-a28a-acd85286d87c  g2w-t-c2m1
4      4G     0     3053c9d0-890a-4b2c-ae2c-19c2ac86da25  g2l-p-c4m4
4      16G    0     1ff846c5-72d2-41d5-8da9-7d1cb3353022  g2l-t-c4m16g1-l4
6      8G     0     2e60a683-1f84-4f12-a3a9-7caf4bdb5e21  g2w-t-c6m8
6      16G    250G  32ec2250-4123-4d73-b13e-985771208a2e  g2d-t-c6m16d240
8      24G    0     2b55bd82-0deb-42ca-9cbb-9568d90f094b  g2l-t-c8m24
12     32G    0     3229329c-4e4f-4152-98df-4728a8308b56  g2l-p-c12m32
12     32G    0     09ef51a1-139c-439f-9f10-81bae5d845f5  g2w-p-c12m32
12     48G    0     2cac1412-c4ab-4126-b0b0-3f984bbc1a65  g2l-t-c12m48
20     128G   0     214643c8-4f8f-4aa9-a3d6-a28c504c0de4  g2l-p-c20m128g1-l4
24     96G    0     13100f00-46a3-4786-bb4a-0ef56be04134  g2l-p-c24m96
40     128G   0     0f1ea1d0-8c3c-4034-8caa-f6d3113dcca2  g2l-t-c40m128

Note: Some flavors may be restricted to prevent abuse. If you cannot use a flavor,
please contact ConoHa support: https://www.conoha.jp/conoha/contact/
```

#### 5. `CONOHA_ENDPOINT` environment variable — API endpoint override

**Background**: GMO internal developers and staging/testing environments need to point the CLI
at a different API endpoint (e.g., internal staging servers) instead of the production
`https://{service}.{region}.conoha.io`.

**Current behavior**: All API endpoints are constructed from the region:
```
Client.BaseURL(service) → https://{service}.{region}.conoha.io
Authenticate()          → https://identity.{region}.conoha.io/v3/auth/tokens
```

**After**: If `CONOHA_ENDPOINT` is set, it replaces the base URL scheme+host entirely.

```bash
# Example: point to staging environment
export CONOHA_ENDPOINT=https://staging-api.internal.gmo.jp

# This makes all API calls go to:
#   https://staging-api.internal.gmo.jp/v2.1/servers/detail    (compute)
#   https://staging-api.internal.gmo.jp/v3/auth/tokens         (identity)
#   https://staging-api.internal.gmo.jp/v3/{tenant}/volumes    (block-storage)
#   etc.
```

**Design**:
- `CONOHA_ENDPOINT` value is the base URL **without** trailing slash and **without** service prefix
- When set, `Client.BaseURL(service)` ignores service name and region, returns `CONOHA_ENDPOINT` directly
- `Authenticate()` also checks `CONOHA_ENDPOINT` and uses it instead of constructing the identity URL
- This assumes the override endpoint routes all services under one host (reverse proxy pattern)
- Add `EnvEndpoint = "CONOHA_ENDPOINT"` to `internal/config/env.go`

**Modified files**:

| File | Change |
|------|--------|
| `internal/config/env.go` | Add `EnvEndpoint` constant |
| `internal/api/client.go` | `BaseURL()` checks `CONOHA_ENDPOINT`, returns it if set |
| `internal/api/auth.go` | `Authenticate()` checks `CONOHA_ENDPOINT` for identity URL |
| `internal/api/client_test.go` | Test `BaseURL()` with `CONOHA_ENDPOINT` override |
| `README.md` | Add `CONOHA_ENDPOINT` to environment variables table |
| `README-en.md` | Same |
| `README-ko.md` | Same |
| `CLAUDE.md` | Add `CONOHA_ENDPOINT` to env var list |

**Example test case**:
```go
func TestBaseURLWithEndpointOverride(t *testing.T) {
    t.Setenv("CONOHA_ENDPOINT", "https://staging.internal.gmo.jp")
    client := NewClient("c3j1", "tok", "tenant1")
    url := client.BaseURL("compute")
    expected := "https://staging.internal.gmo.jp"
    if url != expected {
        t.Errorf("expected %q, got %q", expected, url)
    }
}
```

#### 6. Custom `User-Agent` header

**Current**: No User-Agent header is set. Go default `Go-http-client/1.1` is sent.

**After**: All HTTP requests include a custom User-Agent header:

```
User-Agent: crowdy/conoha-cli/{version}
```

- `{version}` is the build-time version string from `cmd.version` (e.g., `v0.1.3`)
- Set in `Client.Do()` so it applies to all API requests
- Also set in `Authenticate()` which uses its own `http.Client`

**Design**:
- Add `Version` field to `Client` struct, or pass version via a package-level variable
- Simpler approach: use a package-level `UserAgent` variable in `internal/api/client.go`,
  set from `cmd.version` at init time via `api.SetUserAgent()` or similar
- Format: `crowdy/conoha-cli/{version}` (e.g., `crowdy/conoha-cli/v0.1.3`)

**Modified files**:

| File | Change |
|------|--------|
| `internal/api/client.go` | Add `UserAgent` var, set in `Do()` |
| `internal/api/auth.go` | Set User-Agent in `Authenticate()` |
| `cmd/root.go` | Set `api.UserAgent` from `version` at init |
| `internal/api/client_test.go` | Verify User-Agent header |

#### 7. Debug logging via `CONOHA_DEBUG` and `--verbose`

**Current**: No debug/HTTP logging. `--verbose` flag is defined in README but not implemented.

**After**: Two levels of debug output, all to **stderr** (never stdout).

**Activation**:

| Method | Level | What is logged |
|--------|-------|----------------|
| `--verbose` or `CONOHA_DEBUG=1` | verbose | HTTP method, URL, status code, duration |
| `CONOHA_DEBUG=api` | api | verbose + request/response headers and bodies |

**Output format** (stderr):

```
# verbose level (CONOHA_DEBUG=1)
> POST https://identity.c3j1.conoha.io/v3/auth/tokens
< 201 Created (243ms)

> GET https://compute.c3j1.conoha.io/v2.1/servers/detail
< 200 OK (128ms)

# api level (CONOHA_DEBUG=api)
> POST https://identity.c3j1.conoha.io/v3/auth/tokens
> Content-Type: application/json
> User-Agent: crowdy/conoha-cli/v0.1.3
> {"auth":{"identity":{"methods":["password"],"password":{"user":{"name":"...","password":"****"}}}}}
< 201 Created (243ms)
< X-Subject-Token: ****
< Content-Type: application/json
< {"token":{"expires_at":"2026-03-11T12:00:00Z",...}}
```

**Security — sensitive data masking**:
- Passwords in request bodies → `"password":"****"`
- `X-Auth-Token` header value → `****`
- `X-Subject-Token` header value → `****`
- `Authorization` header value → `****`

**Design**:
- Add `CONOHA_DEBUG` env var to `internal/config/env.go`
- Add `internal/api/debug.go` (new) — debug logger with level check, request/response logging
- `--verbose` flag in `cmd/root.go` sets `CONOHA_DEBUG=1` equivalent internally
- Logging hooks in `Client.Do()` (before request, after response)
- `Authenticate()` also uses the same logging
- Priority: `CONOHA_DEBUG=api` > `CONOHA_DEBUG=1` > `--verbose` (same as `1`) > off

**Modified files**:

| File | Change |
|------|--------|
| `internal/config/env.go` | Add `EnvDebug` constant |
| `internal/api/debug.go` | **New** — debug logger, request/response logging, masking |
| `internal/api/client.go` | Add logging hooks in `Do()` |
| `internal/api/auth.go` | Add logging in `Authenticate()` |
| `cmd/root.go` | Wire `--verbose` flag to debug level |
| `internal/api/debug_test.go` | **New** — test masking, log output |

#### Modified files summary

| File | Change |
|------|--------|
| `cmd/flavor/flavor.go` | Sort, human-readable RAM/DISK, footer message |
| `cmd/auth/auth.go` | Prompt/status label "User ID" in int mode |
| `cmd/root.go` | Set `api.UserAgent` from version, wire `--verbose` to debug |
| `internal/config/env.go` | Add `EnvEndpoint`, `EnvEndpointMode`, `EnvDebug` constants |
| `internal/api/client.go` | `BaseURL()` endpoint override + int mode service path/remap, User-Agent, debug |
| `internal/api/auth.go` | `Authenticate()` endpoint override + int mode `user.id` auth, User-Agent, debug |
| `internal/api/debug.go` | **New** — debug logger, HTTP logging, sensitive data masking |
| `internal/api/client_test.go` | Endpoint override test, User-Agent test |
| `internal/api/debug_test.go` | **New** — masking test, log output test |
| `internal/prompt/prompt.go` | Fix errcheck lint for `term.Restore` |
| `README.md`, `README-en.md`, `README-ko.md` | Add `CONOHA_ENDPOINT`, `CONOHA_ENDPOINT_MODE`, `CONOHA_DEBUG` to env vars |
| `CLAUDE.md` | Add `CONOHA_ENDPOINT`, `CONOHA_ENDPOINT_MODE`, `CONOHA_DEBUG`, int/ext API differences |

#### 8. `CONOHA_ENDPOINT_MODE=int` — Internal API support

**Background**: The external (public) API uses OpenStack-compatible URL structure with service as subdomain.
The internal API (used by frontend) uses a single endpoint with service as path segment, and has
different authentication requirements.

**URL routing differences**:

| Mode | URL pattern |
|------|-------------|
| ext (default) | `https://{service}.{region}.conoha.io/{version}/...` |
| int | `{CONOHA_ENDPOINT}/{service}/{version}/...` |

**Service name remapping** (int mode):

| ext (subdomain) | int (path) |
|---|---|
| `image` | `image-service` |
| `networking` | `network` |
| others | same name |

**Authentication differences**:

| Field | ext (OpenStack) | int |
|-------|----------------|-----|
| User identifier | `user.name` + `user.domain.id` | `user.id` |
| Password | `user.password` | `user.password` |
| Project | `scope.project.id` | `scope.project.id` |

Internal API does NOT support `user.name`/`user.domain` — must use `user.id`.

**UX changes in int mode**:
- `auth login` prompt: "User ID" instead of "API Username"
- `auth status` output: "User ID" instead of "Username"

**Usage**:
```bash
CONOHA_ENDPOINT=http://int-api-host/api \
CONOHA_ENDPOINT_MODE=int \
./conoha auth login
```

**Modified files**:

| File | Change |
|------|--------|
| `internal/config/env.go` | Add `EnvEndpointMode` constant |
| `internal/api/client.go` | `BaseURL()` appends service path + remapping in int mode |
| `internal/api/auth.go` | `Authenticate()` uses `user.id` in int mode, appends `/identity` |
| `cmd/auth/auth.go` | Prompt label "User ID" / status label "User ID" in int mode |

### 0.1.2 Changes

- **Bug fix**: `volume list` タイムスタンプパース失敗を修正 — `FlexTime` カスタム型でタイムゾーンなしフォーマットに対応
- **Bug fix**: `server console` が `os-getVNCConsole` で失敗する問題を修正 — `POST /servers/{id}/remote-consoles` エンドポイントに変更
- **Feature**: `server list` に flavor 名カラムを追加 — flavor 一覧を取得して ID→名前をマッピング
- **Model**: `Server.FlavorID` → `Server.Flavor` (nested `FlavorRef` struct) に変更（OpenStack 標準の `"flavor": {"id": "xxx"}` 形式に対応）

### 0.1.1 Changes

- `auth login`: パスワード入力時にマスク表示（`*******`）、ペースト対応
- `auth login/status`: トークン有効期限にJST（日本時間）も併記
- `server show <id|name>`: ID だけでなくサーバー名でも指定可能に
- `server show`: 出力を人間が読みやすい key-value 形式に改善
- `server rename <id|name> <newname>`: サーバー名変更コマンドを追加
