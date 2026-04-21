# conoha-proxy 連携 blue/green デプロイ 設計書

**Date**: 2026-04-20
**Status**: Approved
**Owner**: t-kim

> **Update 2026-04-21:** A `--no-proxy` mode was added as a coexisting alternative path. See `docs/superpowers/specs/2026-04-21-no-proxy-mode-design.md`.

## 1. 背景と目的

現行の `conoha app deploy` は「tar 転送 → `docker compose up -d --build`」の単一スロット構成で、TLS / ドメインルーティング / ゼロダウンタイム切替を持たない。同リポジトリ群に存在する `../conoha-proxy` (ConoHa VPS 向け Go 製リバースプロキシ) が、Let's Encrypt 自動 TLS・Host ヘッダールーティング・blue/green スワップ (drain ウィンドウ) を Admin API として提供しているため、それに統合する。

本設計のゴールは次の 2 つ:

1. `conoha app deploy` を conoha-proxy 経由の blue/green デプロイに **全面置換** する
2. proxy コンテナのライフサイクルを扱う新しいコマンド群 `conoha proxy *` を追加する

Kamal (37signals) の CLI 構成を参照モデルとし、proxy は「インフラコンポーネント」として裏に隠す。ユーザーは通常 `conoha app deploy` のみを使い、proxy の存在を意識しない。

## 2. スコープと破壊的変更

- `conoha app deploy` の実装を全面置換。既存の「単一スロット compose up」経路は削除。
- 既存 `conoha app init` が生成していた git bare repo + `post-receive` フック経路は廃止。全デプロイは `conoha app deploy` 一本に集約。
- 新設定ファイル `conoha.yml` をリポジトリルートに要求。
- 既存ユーザーは `conoha.yml` 作成 + `conoha proxy boot` + `conoha app init` 再実行が必要。CHANGELOG / README / skill レシピを同時更新する。

非スコープ:

- 複数 VPS への同時デプロイ (Kamal の `servers:` 配列相当)。将来検討。
- accessory ごとの独立ライフサイクル管理 (Kamal `kamal accessory *` 相当のきめ細かい操作)。今回は「web 以外は最初の 1 回だけ起動」で十分。
- 統合テスト (実 VPS + 実 proxy 起動) は本設計範囲外。運用 runbook で手順を文書化するのみ。

## 3. 新しいコマンド表面

### 3.1 `conoha proxy` (新規コマンドグループ)

Kamal `kamal proxy *` と同形。proxy コンテナそのものの生涯を扱う。

| コマンド | 動作 |
|---|---|
| `proxy boot <server>` | VPS に `ghcr.io/crowdy/conoha-proxy:latest` を docker run。`/var/lib/conoha-proxy` の作成と所有権変更 (uid 65532) を含む。`--acme-email` 必須。 |
| `proxy reboot <server>` | `docker pull` 後、既存コンテナを `stop`+`rm` して新コンテナを起動。ボリュームは維持。 |
| `proxy start|stop|restart <server>` | 対応する `docker start|stop|restart conoha-proxy` を実行。 |
| `proxy remove <server>` | コンテナを削除。`--purge` 指定時はホストボリューム (`/var/lib/conoha-proxy`) も削除。 |
| `proxy logs <server> [--follow] [-n N]` | `docker logs conoha-proxy` のラッパー。 |
| `proxy details <server>` | admin API `/version`, `/readyz`, 登録サービス数を表示。 |
| `proxy services <server>` | `GET /v1/services` をラップして登録サービス一覧を表形式で表示 (デバッグ用)。 |

### 3.2 `conoha app` (既存グループの修正と追加)

| コマンド | 状態 | 変更点 |
|---|---|---|
| `app init <server>` | **修正** | Docker 有無確認 + `conoha.yml` 検証 + proxy に service upsert (`POST /v1/services`)。git bare repo / post-receive フック生成は **削除**。 |
| `app deploy <server>` | **置換** | tar 転送 → 新スロットで compose 起動 → ポート特定 → proxy `/deploy` 呼び出し → drain 後に旧スロット停止。 |
| `app rollback <server>` | **新規** | proxy `/rollback` を呼ぶ。drain ウィンドウ外なら `no_drain_target` を明示的に案内。 |
| `app status <server>` | 修正 | `docker compose ps` に加えて proxy 側 `phase` / `active_target` / `drain_deadline` を表示。 |
| `app destroy <server>` | 修正 | compose down に加え、proxy `DELETE /v1/services/<name>` を呼ぶ。 |
| `app logs|stop|restart|env|reset|list` | 維持 | ただし `logs` / `stop` / `restart` は「現在 active のスロット」を対象に動作する。 |

## 4. `conoha.yml` 設定ファイル

リポジトリルートに配置。YAML。他 CLI 設定 (`~/.config/conoha/`) とは独立。

```yaml
# 必須
name: myapp                    # proxy service name 兼 compose project prefix
hosts:                         # proxy の Host ルーティング対象。TLS もこの名前で発行される
  - app.example.com

web:
  service: web                 # compose ファイル内のサービス名。proxy が upstream にするのはこれのみ
  port: 8080                   # web コンテナ内部のリスニングポート。target_url 生成に使用

# 任意 (既定値あり)
compose_file: compose.yml      # 省略時は現行の自動検出順序を踏襲
accessories: [db, redis]       # blue/green から除外。初回のみ起動し、以後維持
health:
  path: /up
  interval_ms: 5000
  timeout_ms: 2000
  healthy_threshold: 1
  unhealthy_threshold: 3
deploy:
  drain_ms: 30000              # blue/green swap 後、旧スロットを drain する時間 (ms)
```

### 検証ルール

- `name`: 空不可。DNS-1123 label (`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`、最大 63 文字)。
- `hosts`: 1 件以上、重複不可、各要素は FQDN として妥当。
- `web.service`: `compose_file` 内のサービスとして存在すること (deploy 時に照合)。
- `web.port`: 1–65535。
- `accessories`: `compose_file` 内のサービスの部分集合。`web.service` を含まない。
- `health.*`, `deploy.drain_ms`: 省略時は proxy 既定値に委譲。

パース失敗は exit code 4 (validation) とし、行番号を含む診断を stderr に出力する。

## 5. blue/green デプロイフロー (`conoha app deploy`)

```
[local]                                                 [VPS]
  │
  │ 1. conoha.yml 読込・検証
  │
  │ 2. proxy に現在の service を問い合わせ (GET /v1/services/<name>)
  │    - 404 なら「app init を先に」案内して exit
  │    - 既存があれば active_target / phase を把握
  │
  │ 3. 新スロット ID を決定
  │    - git 環境なら HEAD の short SHA、そうでなければ Unix タイムスタンプ
  │    - 既に同名が存在する場合は -2, -3, ... と suffix
  │
  │ 4. アーカイブ作成 (既存 tar 処理流用、.dockerignore 尊重)
  │
  │ 5. SSH 経由アップロード
  │    - work dir: /opt/conoha/<name>/<slot>
  │    - .env 保持ロジックは既存仕様を踏襲 (ENV_EXISTS sentinel)
  │
  │ 6. compose override ファイルを生成・注入
  │    - web サービスの ports を [127.0.0.1:0:<web.port>] に差し替え (OS が free port を割当)
  │    - container_name を <name>-<slot>-web に固定
  │
  │ 7. accessory 初回起動チェック
  │    - docker compose -p <name>-accessories ps で存在確認
  │    - 未起動なら up -d <accessories...> (deploy 内で 1 回だけ)
  │
  │ 8. 新スロット起動
  │    - SSH: docker compose -p <name>-<slot> -f compose.yml -f override.yml \
  │            up -d --build <web.service>
  │
  │ 9. 動的ポート解決
  │    - SSH: docker port <name>-<slot>-web <web.port>
  │    - 出力から 127.0.0.1:<host_port> を抽出 (失敗時 10 へ)
  │
  │10. proxy /deploy 呼び出し
  │    - SSH: curl --unix-socket /var/lib/conoha-proxy/admin.sock ... /deploy
  │    - body: { "target_url": "http://127.0.0.1:<host_port>", "drain_ms": <drain_ms> }
  │    - 424 probe_failed の場合 → 11 へ (rollback cleanup)
  │    - 200 OK の場合 → 12 へ
  │
  │11. 失敗時クリーンアップ
  │    - SSH: docker compose -p <name>-<slot> down --volumes=false
  │    - proxy 側 state は 424 時点で未変更のため追加操作不要
  │    - exit 1
  │
  │12. 成功時: 旧スロット停止を予約
  │    - SSH: at now + <drain_ms>ms <<< "docker compose -p <name>-<old-slot> down"
  │    - at 不在環境では nohup bash -c "sleep X; docker compose down" にフォールバック
  │    - CLI は wait しない (await しても旧スロットが消えるのは drain_ms 後)
  │
  ▼ 完了
```

### スロット ID

- git リポジトリ: `git rev-parse --short=7 HEAD`
- 非 git: `YYYYMMDDHHMMSS`
- 衝突時: `<id>-2`, `<id>-3`, ...

`conoha app status` の表示に直接出るため、読めることを優先する。

### accessory の扱い

- `conoha.yml` の `accessories` に列挙されたサービスは「永続化対象」扱い。
- 初回 deploy で `docker compose -p <name>-accessories up -d <accessories...>` を一度だけ実行 (`app init` では起動しない)。
- 以後の deploy では accessory を起動・停止しない。volume も blue/green の影響外。
- accessory 間のネットワークは `<name>-accessories_default` ネットワークに参加、web スロットも外部ネットワークとしてそこに join する (compose override の `networks:` 節で指定)。

## 6. Admin API アクセス方式

conoha-proxy の admin socket は VPS 上の `/var/lib/conoha-proxy/admin.sock`。ローカル CLI からは **既に確立済みの SSH 接続越しに `curl --unix-socket` をリモート実行** する。

```
ssh <user>@<host> curl --unix-socket /var/lib/conoha-proxy/admin.sock \
  -sS -X POST http://admin/v1/services/<name>/deploy \
  -H 'Content-Type: application/json' \
  -d '{"target_url":"http://127.0.0.1:9001","drain_ms":30000}'
```

- TCP 経由 (`--admin-tcp`) は使わない。外部公開リスクを増やさないため。
- 別途の SSH port-forward も行わない。コマンドごとの同期実行で十分。
- レスポンス JSON は CLI が stdout からデコードして Go 構造体へ。
- HTTP ステータスは `-w '%{http_code}'` で末尾に付加して取得。本文と分離。

## 7. 内部レイヤ構成

```
cmd/
  proxy/                        # 新設
    proxy.go                    # コマンドグループ
    boot.go, reboot.go
    start.go, stop.go, restart.go
    remove.go
    logs.go
    details.go
    services.go                 # GET /v1/services のラッパ
  app/
    init.go                     # 修正: git repo 廃止、proxy service upsert 追加
    deploy.go                   # 置換: blue/green フロー
    rollback.go                 # 新設
    destroy.go                  # 修正: proxy DELETE 呼出追加
    status.go                   # 修正: proxy phase 表示追加

internal/
  proxy/                        # 新設
    admin.go                    # Admin API クライアント (SSH+curl)
    admin_test.go
    bootstrap.go                # docker run/stop/rm シェルスクリプト生成
    bootstrap_test.go
    types.go                    # Service / Target / HealthPolicy の Go 型
  config/
    projectfile.go              # conoha.yml パース/検証
    projectfile_test.go
```

### 7.1 Admin API クライアント (`internal/proxy/admin.go`)

```go
type Client struct {
    ssh  SSHExecutor             // interface, 既存 internal/ssh のラッパ
    sock string                  // 既定 "/var/lib/conoha-proxy/admin.sock"
}

func (c *Client) Get(name string) (*Service, error)
func (c *Client) List() ([]Service, error)
func (c *Client) Upsert(req UpsertRequest) (*Service, error)
func (c *Client) Deploy(name string, req DeployRequest) (*Service, error)
func (c *Client) Rollback(name string, drainMs int) (*Service, error)
func (c *Client) Delete(name string) error
```

エラーマッピング:

| proxy 応答 | Go エラー |
|---|---|
| `200` / `201` / `204` | nil |
| `404 not_found` | `ErrNotFound` (sentinel) |
| `400 validation_failed` / `invalid_body` | `ValidationError` |
| `424 probe_failed` | `ProbeFailedError` (本文 message 含む) |
| `409 no_drain_target` | `ErrNoDrainTarget` |
| `503 *` | `ServerError` |
| その他 / SSH 失敗 | `fmt.Errorf("...: %w", err)` |

### 7.2 Bootstrap (`internal/proxy/bootstrap.go`)

proxy コンテナ関連の生成系:

```go
func BootScript(email string, image string) []byte    // docker run ... の bash
func RebootScript(image string) []byte
func StartScript() []byte                             // docker start conoha-proxy
func StopScript() []byte
func RestartScript() []byte
func RemoveScript(purge bool) []byte
func LogsScript(follow bool, lines int) []byte
```

すべて `bash -euo pipefail` ヘッダ付きで、SSH の `RunScript` に渡して実行。構造は現行 `cmd/app/init.go` の `generateInitScript` と揃える。

### 7.3 Project file (`internal/config/projectfile.go`)

```go
type ProjectFile struct {
    Name        string           `yaml:"name"`
    Hosts       []string         `yaml:"hosts"`
    Web         WebSpec          `yaml:"web"`
    ComposeFile string           `yaml:"compose_file,omitempty"`
    Accessories []string         `yaml:"accessories,omitempty"`
    Health      *HealthSpec      `yaml:"health,omitempty"`
    Deploy      *DeploySpec      `yaml:"deploy,omitempty"`
}

func Load(path string) (*ProjectFile, error)
func (p *ProjectFile) Validate() error
func (p *ProjectFile) ComposeFilePath() (string, error)   // compose_file 未指定時は現行の自動検出
```

## 8. エラー処理と観測性

stderr は進行ログを人間可読で出す (既存 `fmt.Fprintf(os.Stderr, ...)` 踏襲)。

```
==> Loading conoha.yml
==> Querying proxy service 'myapp'
==> Building slot 'a1b2c3d' (compose project: myapp-a1b2c3d)
==> Uploading archive (1.2 MB)
==> docker compose up -d --build web
==> Allocated host port: 49231
==> Probing http://127.0.0.1:49231/up via proxy
==> Swapped. Draining previous slot 'ffe1234' for 30s.
Deploy complete.
```

`--format json` 指定時は最終 service オブジェクト (proxy 応答) のみ stdout に出す。

| シナリオ | 終了コード | メッセージ方針 |
|---|---|---|
| `conoha.yml` 不在 / パース失敗 | 4 | 行番号付き診断 |
| admin socket 不在 (proxy 未 boot) | 1 | `run 'conoha proxy boot <server>' first` |
| proxy `/deploy` 424 probe_failed | 1 | proxy 応答 `message` をそのまま表示 + 「upstream ログを確認」 |
| proxy `/rollback` 409 no_drain_target | 3 | 「drain 窓が切れています。代わりに直前 SHA を deploy してください」 |
| docker port 出力パース失敗 | 1 | 新スロット teardown 後に exit |
| SSH 接続失敗 | 6 (network) | 既存 `internal/ssh` のエラー伝播 |

## 9. テスト戦略

### 単体テスト

- `internal/proxy/admin_test.go` — `SSHExecutor` を fake に差し替え、各 HTTP ステータス (200/400/404/409/424/503) のレスポンス本文に対して適切な Go エラー型にマップされることを確認。リクエスト合成 (JSON body, URL, メソッド) も検証。
- `internal/proxy/bootstrap_test.go` — `BootScript` / `RebootScript` 等の生成結果を gold ファイルと突合。
- `internal/config/projectfile_test.go` — 必須欠落、不正 DNS label、重複 hosts、範囲外 port、web.service と compose 不一致などのエラーケース網羅。
- `cmd/app/deploy_test.go` — 既存テストを刷新。スロット ID 決定、compose override 生成、probe 失敗時 teardown、成功時 drain 予約の呼び出しを fake SSH で検証。
- `cmd/app/rollback_test.go` — 200 / 409 / 404 の伝播。
- `cmd/proxy/*_test.go` — 各サブコマンドが期待するシェルコマンド / admin API 呼び出しを合成することを検証。

### 統合テスト

今回のスコープ外。`docs/superpowers/specs/` 配下に手動検証手順を別添するか、`recipes/` の既存サンプルを proxy 対応に更新する形で代替。

## 10. 不変条件と安全性

1. proxy `/deploy` が 424 を返した場合、VPS 上の state.db は一切変わらない (proxy 側の保証)。CLI 側はそれに依拠して新スロットを安全に teardown してよい。
2. 新スロットの host port は `127.0.0.1:0` でカーネル割当を使用するため、blue/green 間でのポート衝突は原理的に発生しない。
3. accessory は blue/green の影響を受けない単一プロジェクト (`<name>-accessories`) で管理するため、web スロットがどれだけ切り替わっても volume / network は維持される。
4. CLI は probe を自前で行わない。健全性判断は全面的に proxy に委任する (proxy の `internal/health.ProbeOnce` が真実)。
5. `conoha.yml` は VPS に転送されない。proxy 側 state.db が「サーバー上の真実」、`conoha.yml` は「レポ上の真実」で、両者の突き合わせは `conoha proxy services` / `conoha app status` で観測可能。

## 11. マイグレーション

既存ユーザー向け手順 (README / skill レシピに掲載):

1. レポに `conoha.yml` を作成 (最小構成でよい)。
2. `conoha proxy boot <server> --acme-email you@example.com`
3. DNS の A レコードを VPS に向ける (Let's Encrypt HTTP-01 に必要)。
4. `conoha app init <server>` (新しい init は proxy に service を登録する)。
5. `conoha app deploy <server>`

旧 `/opt/conoha/<app>.git` repo と post-receive フックは新 `init` が検出した場合、**警告のみ** 出して触らない。完全削除はユーザーが手動で `ssh <user>@<server> rm -rf /opt/conoha/<app>.git /opt/conoha/<app>.env.server` する。`conoha app reset` は本リファクタで一旦廃止 (issue #92 で再実装予定)。

## 12. 実装順序 (plan で詳細化)

1. `internal/config/projectfile.go` — conoha.yml パース・検証。
2. `internal/proxy/types.go` — Service / Target / HealthPolicy / エラー型。
3. `internal/proxy/admin.go` — Admin API クライアント。
4. `internal/proxy/bootstrap.go` — proxy コンテナ生涯スクリプト。
5. `cmd/proxy/*` — 全サブコマンド (boot/reboot/start/stop/restart/remove/logs/details/services)。
6. `cmd/app/init.go` — service upsert への置換、git repo 経路削除。
7. `cmd/app/deploy.go` — blue/green フローへの全面書き換え + compose override 生成。
8. `cmd/app/rollback.go` — 新設。
9. `cmd/app/destroy.go` / `status.go` — proxy 連携追加。
10. 既存テスト再作成 + 新規テスト。
11. README / docs / recipes / skill 更新。

---

承認後、writing-plans スキルで詳細タスクプランに落とす。
