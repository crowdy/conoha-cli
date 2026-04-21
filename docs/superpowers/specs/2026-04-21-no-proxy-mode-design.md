# `--no-proxy` モード 設計書

**Date**: 2026-04-21
**Status**: Approved
**Owner**: t-kim
**Related**: #102 (this issue), #98 (direct predecessor), #92/#93/#94 (adjacent)

## 1. 背景と目的

`feat/proxy-deploy` (#98, 2026-04-20 spec) で `conoha app deploy` は conoha-proxy 経由の blue/green に全面置換された。これにより失われたユースケースがある:

- 公開ドメイン / DNS 伝播待ちなしでのテスト。
- blue/green + HTTPS が過剰な使い捨てのホビーアプリ。
- HTTP 以外のプロトコルを公開するサービス (proxy はルーティング不可)。
- 受信 80/443 を持たない内部 / 開発 VPS。

本設計は `--no-proxy` モードを **一級の代替経路** として追加する。proxy ベースの blue/green 経路は触らない。両モードは同一サーバ上で共存する。

**先行スペック §2 の "full replacement (option A)" 判断はここで部分的に巻き戻される**: "全デプロイを proxy 経由に集約" は過剰だった。proxy は既定経路であり続けるが唯一の経路ではない。

### 1.1 ゴール

1. `conoha app {init,deploy,logs,stop,restart,status,destroy,env,rollback}` に no-proxy 動作を追加。
2. 同一サーバ上で proxy モードと no-proxy モードが共存できる。`/opt/conoha/<name>/<slot>/` と `/opt/conoha/<name>/` (flat) がアプリごとに分かれる。
3. モード選択はサーバ側マーカーで自動判定し、必要なら明示フラグで override 可能。
4. モード整合性違反は明示的にエラーにする (サイレント破壊なし)。

### 1.2 非ゴール

- v0.1.x の git-push-deploy 経路の復元。廃止は維持。
- proxy による非 HTTP 転送 (proxy の責務外)。
- `app list` の no-proxy 対応 (#95 で扱う)。
- proxy モードでの `app env` 再設計 (#94 で扱う)。
- `app reset` の両モード対応 (#92 で扱う。本 PR マージ後に作業を再開)。

## 2. モード選択

### 2.1 マーカーファイル

各アプリは初期化時にサーバ側マーカーを取得する。

- **パス**: `/opt/conoha/<name>/.conoha-mode`
- **内容**: 単一行 `proxy\n` または `no-proxy\n`。
- **所有者**: 書き込みは `app init` (または `app init --no-proxy`)。削除は `app destroy` が `rm -rf /opt/conoha/<name>` の副作用として消す。

マーカーはドットファイルにする (通常の `ls /opt/conoha/<name>/` 出力を汚さない)。人間が確認するときは `cat /opt/conoha/<name>/.conoha-mode`。

### 2.2 解決アルゴリズム

`ResolveMode(cmd, cli, app)` の優先順位:

1. `--proxy` / `--no-proxy` フラグ (相互排他) のどちらかが指定されている場合、その値を **希望値** とする。マーカーを読み、不一致なら `ErrModeConflict` を整形して返す。
2. フラグなし:
   - マーカー存在 → その値を返す。
   - マーカー不在 → `ErrNoMarker` を返す。呼び出し側が文脈に応じて解釈する (§3 参照)。

### 2.3 モード衝突時の動作

proxy で初期化されたアプリに `--no-proxy` を指定 (または逆) した場合の動作は **Z: 明示エラー + 手動復旧案内**。自動転換は提供しない (blue/green の drain target が生きている間の切替はデータ欠損の恐れがある)。

エラー文例:

```
app "myapp" is initialized in proxy mode on this server,
but --no-proxy was requested.

To switch modes:
    conoha app destroy <server>               # removes the existing deployment
    conoha app init --no-proxy <server>       # re-initialize in no-proxy mode
```

exit code: 5 (mode-conflict)。

## 3. コマンドごとの動作

### 3.1 `conoha app init <server>`

| モード | 挙動 |
|---|---|
| proxy (既定) | 現行通り。conoha.yml 読込 / 検証、Docker 有無確認、proxy `POST /v1/services` upsert、**加えて `.conoha-mode=proxy` を書き込む**。 |
| no-proxy (`--no-proxy`) | conoha.yml を読まない。`--app-name` 必須 (なければ exit 2 / "app-name is required with --no-proxy")。Docker 有無確認、`mkdir -p /opt/conoha/<name>`、`.conoha-mode=no-proxy` を書き込む。proxy admin API は呼ばない。 |

`--proxy` / `--no-proxy` は相互排他。`--app-name` と conoha.yml の `name` は両者が存在すればフラグ優先。

### 3.2 `conoha app deploy <server>`

| モード | 挙動 |
|---|---|
| proxy (既定) | 現行通り。conoha.yml 必須。マーカー不在なら `run 'conoha app init' first` を案内して exit。マーカー=proxy なら進行。 |
| no-proxy | conoha.yml 無視。`--app-name` 必須。`ResolveComposeFile(".")` で compose 自動検出。tar アップロード → `/opt/conoha/<name>/` (flat) → `docker compose -p <name> up -d --build`。ホストポート割当は compose の `ports:` をそのまま尊重。`.env` 保存ロジックは v0.1.x の `ENV_EXISTS` センチネルを再現 (サーバ側 `/opt/conoha/<name>.env.server` を work-dir に merge)。proxy admin API は呼ばない。 |

**衝突ケース:**

- マーカー=proxy & `--no-proxy` → §2.3 エラー。
- マーカー=no-proxy & (フラグなし or `--proxy`) → §2.3 形式の逆向きエラー。
- マーカー不在 + フラグなし → `run 'conoha app init' first`。
- マーカー不在 + `--no-proxy` → マーカーを先に書くよう案内 (`run 'conoha app init --no-proxy' first`)。init 前の "deploy 一発で全部" は意図的にサポートしない (整合性検査が壊れる)。

### 3.3 `conoha app rollback <server>`

| モード | 挙動 |
|---|---|
| proxy | 現行通り。proxy `/rollback` を呼び出す。 |
| no-proxy | exit 5 + メッセージ: `"rollback is not supported in no-proxy mode. Deploy a previous revision instead: git checkout <rev> && conoha app deploy --no-proxy <server>"` |

明示フラグがなくてもマーカーから判定。`--no-proxy` を明示的に付けた場合も同じエラーを返す (意味的にサポート不能)。

### 3.4 `conoha app destroy <server>`

| モード | 挙動 |
|---|---|
| proxy | 現行通り。全 slot compose down + accessories down + proxy `DELETE /v1/services/<name>` + `rm -rf /opt/conoha/<name>`。 |
| no-proxy | flat compose down (`docker compose -p <name> down`) + `rm -rf /opt/conoha/<name>` (マーカーも一緒に消える)。**proxy DELETE は呼ばない**。 |

既存 `generateDestroyScript` は `docker compose ls -a | grep -E "^${APP_NAME}(-|$)"` で slot プロジェクトと flat プロジェクト両方を網羅するため、shell 側の変更は不要。Go 側で proxy delete 呼び出しをモード分岐する。

**マーカー不在時**: フラグ override なしなら "best-effort" で両経路を実行 (スクリプトで compose ls が一致すれば down、ディレクトリがあれば rm)。レガシー v0.1.x サーバの掃除パスを壊さない。

### 3.5 `conoha app logs|stop|restart|status <server>`

このセクションで **issue #93 を完全に吸収する** (§7 参照)。

| モード | logs | stop | restart | status |
|---|---|---|---|---|
| proxy | 活性 slot: `ReadCurrentSlot` → `docker compose -p <name>-<slot> logs` | 同じ project への compose stop | 同じ project への compose restart | 現行 (slot プロジェクト一覧 + proxy phase) 維持 |
| no-proxy | `cd /opt/conoha/<name> && docker compose logs` (現行コード) | 同 stop | 同 restart | `cd /opt/conoha/<name> && docker compose ps`。proxy phase ブロックは出力しない |

**"never deployed on this server" の判定:**
- no-proxy モード (マーカー=no-proxy): `/opt/conoha/<name>/` 下に `docker compose ls -p <name>` が何も返さない、または work dir が compose ファイルを含まない。
- proxy モード (マーカー=proxy): `CURRENT_SLOT` ファイル不在。
- マーカー不在 + フラグ override なし: ErrNoMarker を伝搬。

いずれも exit 6 の統一エラーにする: `"app \"<name>\" has not been deployed on <server>"`.

`--proxy` / `--no-proxy` フラグは logs/stop/restart/status でも受け付ける (低頻度 override 用途)。

### 3.6 `conoha app env <server>`

| モード | 挙動 |
|---|---|
| no-proxy | 現行動作を **正式仕様化**: `/opt/conoha/<name>.env.server` を読み書きし、次回 deploy の `.env` merge で採用される。 |
| proxy | 現行動作を維持するが、**開始時に 1 行警告**: `"warning: app env has no effect on proxy-mode deployed slots; see #94 for the redesign"`。書き込みは継続 (既存 CI スクリプトを壊さない)。 |

警告は `stderr`。スクリプト利用の互換性のため終了コードは変えない。

### 3.7 `conoha app list`

本スペックの範囲外 (#95)。`list` は現状のまま proxy services を列挙する。

## 4. 実装アーキテクチャ

### 4.1 新規ファイル: `cmd/app/mode.go`

```go
package app

import (
    "errors"
    "fmt"

    "github.com/spf13/cobra"
    "golang.org/x/crypto/ssh"
)

type Mode string

const (
    ModeProxy   Mode = "proxy"
    ModeNoProxy Mode = "no-proxy"
)

var (
    ErrNoMarker     = errors.New("no mode marker on server")
    ErrModeConflict = errors.New("mode conflict")
)

// ReadMarker returns the mode recorded on the server for app, or ErrNoMarker.
func ReadMarker(cli *ssh.Client, app string) (Mode, error)

// WriteMarker persists the mode marker. Creates the app dir if needed.
func WriteMarker(cli *ssh.Client, app string, m Mode) error

// ResolveMode interprets --proxy / --no-proxy flags against the on-server marker.
// Returns ErrNoMarker if neither a flag nor a marker is available.
// Returns a wrapped ErrModeConflict (with formatted user guidance) on mismatch.
func ResolveMode(cmd *cobra.Command, cli *ssh.Client, app string) (Mode, error)

// ReadCurrentSlot returns the active slot ID from /opt/conoha/<app>/CURRENT_SLOT.
// Returns an empty string + nil error if the file is absent (= never deployed).
func ReadCurrentSlot(cli *ssh.Client, app string) (string, error)

// AddModeFlags registers --proxy and --no-proxy as mutually exclusive bool flags.
func AddModeFlags(cmd *cobra.Command)
```

`AddModeFlags` を `init`, `deploy`, `rollback`, `destroy`, `logs`, `stop`, `restart`, `status` の `init()` で呼ぶ。`env` には追加しない (env は警告のみで分岐しないため不要)。

### 4.2 既存ファイル変更点

- `cmd/app/init.go` — `--no-proxy` 分岐追加、全ケースで `WriteMarker`。
- `cmd/app/deploy.go` — モード解決→分岐。no-proxy 経路は `runNoProxyDeploy(cmd, ssh, app)` に切り出し。既存 `runDeploy` は proxy 用にリネーム (`runProxyDeploy`)。
- `cmd/app/rollback.go` — no-proxy モード検出時に早期 exit 5。
- `cmd/app/destroy.go` — モード解決後に proxy `DELETE` 呼び出しを分岐。
- `cmd/app/logs.go` / `stop.go` / `restart.go` — proxy モードでは `ReadCurrentSlot` → `docker compose -p <name>-<slot> ...`。no-proxy は現行コード。
- `cmd/app/status.go` — proxy phase 出力をモードで分岐。no-proxy では flat `docker compose ps` のみ。
- `cmd/app/env.go` — proxy モード時に警告 1 行を stderr に出力。

### 4.3 新規・改修テスト

| ファイル | 目的 |
|---|---|
| `cmd/app/mode_test.go` | ReadMarker / WriteMarker / ResolveMode 全分岐、ErrModeConflict 文字列の snapshot。 |
| `cmd/app/deploy_test.go` (拡張) | `--no-proxy` 分岐: `--app-name` 必須、conoha.yml 不要、proxy admin に到達しない。 |
| `cmd/app/init_test.go` (拡張) | `--no-proxy`: conoha.yml 不要、Docker check + marker write。 |
| `cmd/app/rollback_test.go` | no-proxy モードで exit 5。 |
| `cmd/app/destroy_test.go` (拡張) | no-proxy モードで proxy `DELETE` 呼ばれないこと。マーカー不在 (legacy) で best-effort 成功。 |
| `cmd/app/logs_test.go` (新規) | proxy モードで `docker compose -p <app>-<slot> logs` を実際に発行する。 |
| `cmd/app/status_test.go` (新規) | no-proxy モードで proxy phase 出力なし。 |

SSH は既存パターン通り `internalssh.RunCommand` を interface 化しているところを mock (追加の抽象化は避け、exec 記録型の fake client を `mode_test.go` で用意して使い回す)。

## 5. エラー・終了コード

| code | 意味 | 発生例 |
|---|---|---|
| 0 | 成功 | |
| 1 | 一般失敗 | SSH 切断、docker 失敗 |
| 2 | usage / 引数エラー | `--no-proxy` with no `--app-name` |
| 4 | validation | conoha.yml 解析失敗 (既存) |
| **5 (新)** | mode-conflict | §2.3、rollback in no-proxy |
| **6 (新)** | not-initialized | logs/stop/restart/status でマーカー & CURRENT_SLOT 両方不在 |

実装は `cmd/cmdutil` に `ExitWithCode(err, code)` が既に存在すれば流用、無ければ `return` 値で cobra に任せ Run ラッパーでコード設定。

## 6. 設定・CLI 表面まとめ

### 6.1 新フラグ

- `--proxy` (bool) — モード override。下 8 コマンドに追加。既定 false。
- `--no-proxy` (bool) — 同上。相互排他。既定 false。

対象: `init`, `deploy`, `rollback`, `destroy`, `logs`, `stop`, `restart`, `status`。

### 6.2 `conoha.yml` 変更

**変更なし**。no-proxy モードでは読まない。

### 6.3 サーバ側レイアウト

```
/opt/conoha/<app>/
├── .conoha-mode          # "proxy" or "no-proxy"
├── CURRENT_SLOT          # proxy mode only
├── <slot-id>/            # proxy mode only (per slot work dir)
│   ├── (extracted tar)
│   └── conoha-override.yml
└── (extracted tar)       # no-proxy mode (flat — no slot subdir)

/opt/conoha/<app>.env.server   # unchanged — no-proxy canonical env file (#94 will revisit for proxy)
```

## 7. #93 との統合

#93 は `app logs/stop/restart/status` が proxy モード下でも legacy flat path を参照しているバグの issue。本スペックは §3.5 でこれを完全に解決する。`--no-proxy` モードの実装コードパス = #93 が想定していた "旧コード"。proxy 側のコードパスは新規実装。

本 PR マージ時に #93 を close する。

## 8. 他 issue との関係

- **#92** (`app reset`): 本 PR 後に両モード対応で作り直す。スコープ外。
- **#94** (`app env` 再設計): proxy モードで有効な env の扱い。本 PR では proxy モード時に警告出すのみ。#94 で抜本改修。
- **#95** (`app list` no-proxy 対応): 別 issue で別途。

## 9. 移行と後方互換

- 既存 proxy モードユーザー: `app init` を 1 度再実行するとマーカーが書かれる。再実行しない場合、次の `app deploy` 時にマーカー不在として `run 'conoha app init <server>' first` が案内される (§3.2)。自動 migration は行わない (モード判定の単一ソースを壊さないため)。
- v0.1.x ユーザー: `/opt/conoha/<name>/` が flat 配置で残っているが `.conoha-mode` は無い。`app init --no-proxy` でマーカーを書けば no-proxy モードとして継続可能。deploy は flat を上書きする。
- v0.1.x の `<name>.git` bare repo: 現行 `warnOnLegacyRepo` のまま警告のみ。本 PR で挙動変更しない。

## 10. ドキュメント

- README (en/ja/ko): "Two deploy modes" セクション追加。proxy = 推奨 / no-proxy = TLS-less single-slot。各 3–5 行。
- `docs/recipes/single-server-app.md` を proxy 版として保持、`single-server-app-noproxy.md` を新規追加 (同じシナリオの no-proxy 版)。
- 先行スペック (`2026-04-20-conoha-proxy-deploy-design.md`) の先頭に "Update 2026-04-21: `--no-proxy` mode added — see 2026-04-21-no-proxy-mode-design.md" の一行を追加。

## 11. 受け入れ基準

- [ ] `conoha app init --no-proxy <server>` が conoha.yml なしで成功し、`/opt/conoha/<name>/.conoha-mode=no-proxy` を残す。
- [ ] `conoha app deploy --no-proxy --app-name <x> <server>` が同サーバ上で proxy アプリと併存しながら動作。
- [ ] proxy 初期化済みアプリへの `conoha app deploy --no-proxy` が exit 5 + 復旧手順メッセージで停止。
- [ ] `conoha app logs/stop/restart/status <server>` が proxy モードで活性 slot を正しく対象化する (#93 解決)。
- [ ] `conoha app rollback --no-proxy <server>` が exit 5 + git-based 復旧ヒントで停止。
- [ ] `conoha app destroy <server>` がマーカーを見て proxy DELETE を呼ぶかどうか分岐し、どちらのモードでも `/opt/conoha/<name>/` を完全削除。
- [ ] `conoha app env *` が proxy モードで 1 行警告を出しつつ既存動作を継続。
- [ ] 単体テストで上記分岐をすべて網羅。
- [ ] README / recipes が両モードの例を示す。

## 12. オープンな技術判断

スペック確定済みだが実装段階で見直し可能な点:

- **マーカー書き込み失敗時のロールバック**: init 中 `WriteMarker` が失敗した場合、proxy upsert は既に成功している (proxy モード時)。現行案は "警告のみ、proxy 側は残す"。代案として upsert を取り消す。本 PR では前者を採用 (実装簡潔性)。
- **`destroy` でマーカー不在の legacy サーバ**: "best-effort" スクリプト実行を継続。`--force` フラグを将来追加する余地あり。
- **`--no-proxy` と `--app-name` の関係**: no-proxy モードでは常に `--app-name` 必須。proxy モードでは conoha.yml の `name` が優先されるため `--app-name` は補助的。この非対称は意図的 (no-proxy は設定ファイル不在が正常経路)。
