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
| 0.1.3 | TBD | flavor list UX improvements (see below) |
| 0.1.2 | 2026-03-10 | Bug fixes and feature improvements (see below) |
| 0.1.1 | 2026-03-10 | UX improvements (see below) |
| 0.1.0 | 2026-03-10 | Initial implementation - all API endpoints |

### 0.1.3 Changes (planned)

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

#### Modified files summary

| File | Change |
|------|--------|
| `cmd/flavor/flavor.go` | Sort, human-readable RAM/DISK, footer message |

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
