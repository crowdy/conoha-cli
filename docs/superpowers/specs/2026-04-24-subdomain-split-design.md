# サブドメイン分割 (multi-host / multi-service blue/green) 設計書

**Date**: 2026-04-24
**Status**: Proposed (RFC — open questions のマージ前レビュー要)
**Owner**: t-kim
**Related**: #96 (E2E, closed 2026-04-24), #97 (app-samples migration, closed via `conoha-cli-app-samples#53`), #98 (proxy blue/green), #103 (no-proxy), #94 (app env redesign)

## 1. 背景

`conoha-cli` v0.2.0 に向けた app-samples の移行 (`conoha-cli-app-samples` PR #47–#53) で **43 / 43 サンプル** が proxy blue/green レイアウトに移ったが、うち **8 サンプル** が既知の制限付きでマージされた。いずれも「一つの `conoha.yml` で一つの `(host, service, port)` しか扱えない」という現行スキーマの構造的な限界に帰着する。

現行の `conoha.yml` (`internal/config/projectfile.go:15-23`):

```yaml
name: myapp
hosts: [app.example.com]        # proxy が SNI / Host でルーティング
web:
  service: web                  # blue/green 対象はこの 1 サービスだけ
  port: 8080
accessories: [db, redis, ...]   # 単発起動 / blue/green 対象外
```

proxy は service 単位で `(name, hosts[], active_target, draining_target)` を持ち (`internal/proxy/types.go:31-45`)、CLI は `composeOverride` (`cmd/app/override.go:17`) で web 1 サービスのみを slot-local 名に固定する。accessory は共通プロジェクト `<app>-accessories` に単発起動 (`cmd/app/deploy.go:278-287`)。この構造の結果として、8 サンプルに次のような実用上の穴が空いている。

### 1.1 影響を受けている 8 サンプル

memory (`project_issue_96_status.md` 2026-04-24) の整理と各サンプルの `conoha.yml` コメント (`conoha-cli-app-samples/<sample>/conoha.yml`) の実記述を突き合わせた表。

| サンプル | 制限カテゴリ | 必要な公開サブドメイン | 内容 |
|---|---|---|---|
| `gitea` | A (外部到達アクセサリ) | `dex.example.com` | Dex OIDC の browser redirect が `dex:5556` を参照するが compose ネットワーク内部のみ。結果、OIDC サインインが失敗。 |
| `outline` | A | `dex.example.com` | 同上 (outline も Dex 使用)。現状は local account / magic-link 経由のみ。 |
| `rails-mercari` | A + C | `auth.example.com`, `app.example.com` | Dex OIDC 同じ理由で外部到達必要 + Rails `web` が accessory なので `nginx` しか blue/green しない。 |
| `hydra-python-api` | A | `auth.example.com` | Hydra 公開 OAuth2 (`:4444`) が compose ネットワーク限定。browser redirect を伴う authz code flow が完結しない。 |
| `nextjs-fastapi-clerk-stripe` | B (署名保持 webhook) | `api.example.com` | Clerk / Stripe の webhook は body の byte-exact HMAC 検証を要するため Next.js rewrite が使えず、`backend:8000` に直接到達したい。 |
| `supabase-selfhost` | A | `admin.example.com` | Studio admin UI (`:3000`) は `docker exec` 経由でしか届かない。ブラウザから触れない。 |
| `quickwit-otel` | A (但しプロトコル制約あり) | `otel.example.com` | 外部からの OTLP (HTTP `:4318`) を受けたい。gRPC (`:4317`) は h2c のため proxy 範囲外 (§7 オープン質問 参照)。 |
| `dify-https` | C (内側 blue/green) | — (既存の `dify-https.example.com` に `api`/`web` 追加) | `nginx` は blue/green するが `api` / `web` / `worker` は accessory 扱いで新 slot を受けない。コード更新時に slot と内側が乖離。 |

### 1.2 制限の整理

実は 3 つに見えるが、根本原因は 2 つに集約される。

- **Gap-M**: 1 アプリが複数の公開ホストを持てない (multi-host)。Category A と B はすべてここに帰着。
- **Gap-B**: blue/green 対象が `web.service` の 1 個に固定されている (multi-service blue/green)。Category C (`dify-https`, `rails-mercari` の Rails 側)。

これを `conoha.yml` のスキーマで "複数の `(host, service, port)` ブロック" を並べられるようにすると Gap-M と Gap-B は同一の設計で解ける (各ブロックがそれぞれ proxy service となり slot 回転する)。したがって本 RFC は Gap-M と Gap-B を **同時に** 扱う。

### 1.3 非目標

- **quickwit-otel gRPC (h2c)**: proxy は現状 HTTP/1.1 フロントなので h2c 透過はスコープ外。`:4318` (HTTP) のみ扱う。OTLP gRPC の外部受けは別 RFC とする。
- **mTLS / Client Cert 認証**: subdomain 単位の認証要件は取り扱わない。必要であれば後続。
- **accessory 単位のライフサイクル制御**: 本 RFC 後も accessory の単発起動モデルを維持する。
- **公開ホストを持たない内部専用 blue/green (worker / 背景 job)**: 本 RFC の `expose` ブロックは `host` / `port` が必須で、proxy service 登録を伴う。`dify-https` の `worker` や同種のバックグラウンドサービス (sidekiq、cron runner 等) は引き続き accessory 扱いのまま残し、コード更新時に slot と worker が乖離する既存の限界は温存する。別 RFC (`internal-only slot rotation`) で扱う。したがって本 RFC 実装後も `dify-https` の `worker` / `rails-mercari` の sidekiq 相当は README の「既知の制限」に残る (api / web / nginx 側は解消)。

---

## 2. 選択肢比較

### (i) `conoha.yml` に multi-host ルーティングを拡張する

CLI/proxy 側にスキーマ拡張と service 複数登録を入れて、1 アプリ = 1 compose プロジェクト = 複数公開ホストの構造を公式に扱う。

```yaml
name: gitea
# 後方互換: 既存の hosts/web はそのまま動く
hosts: [gitea.example.com]
web:
  service: gitea
  port: 3000

# 新規: 追加の公開ホストブロック
expose:
  - label: dex           # proxy service name サフィックス用 (<name>-dex)
    host: dex.example.com
    service: dex         # compose service 名
    port: 5556
    blue_green: true     # default true。false にすると accessory 相当 (slot 外)
accessories: [db]        # ここからは expose で拾ったサービスを除く
```

- **trade-off**:
  - 利点: 単一 git リポジトリ、単一 `app deploy`、単一 `.env.server` (#94) のまま。accessory と slot リソースは共有。TLS も既存パス (proxy の ACME HTTP-01) で別ホスト名ごとに発行される。8 サンプルすべてが一発で救われる。
  - 難点: `internal/config/projectfile.go` のスキーマ破り (後方互換は取れる)、deploy state machine が N-service になる、rollback / status / destroy のセマンティクスを再定義。
- **影響範囲**: CLI のみ。conoha-proxy は触らない (後述 §3.2 参照。proxy service は N 個に増えるだけで 1 service あたりの形は据え置き)。
- **後方互換**: 新フィールド `expose` が未指定なら現行挙動と完全に同一。43 サンプル中 35 件は無改変でよい。

### (ii) サンプルを multi-project に分割する

CLI は触らない。8 サンプルのリポジトリレイアウトを `dify-https-web/`, `dify-https-api/` のように複数プロジェクトへ分割する。

- **trade-off**:
  - 利点: CLI 変更なし。v0.2.0 リリースを遅らせない。
  - 難点: ユーザ体験が崩壊する (accessory DB/Redis の共有ネットワーク / volume の接続を README で手動説明するしかない、deploy が N 回になる、`.env.server` も N 個になる、状態が複数プロジェクトに分散して destroy / rollback がぎこちない)。`app-samples` リポジトリのディレクトリ数が 43 から 51+ に増える。
- **影響範囲**: `conoha-cli-app-samples` のみ。`conoha-cli` は完全に無関係。
- **後方互換**: 無風 (CLI は変わらない)。ただしサンプル利用者が "一つのアプリ" を "N プロジェクト" として理解し直す必要がある。

### (iii) 両方 — CLI 機能 + 推奨パターンとしての分離

(i) で CLI に multi-host サポートを入れつつ、app-samples はその **新しい単一プロジェクトパターン** にマイグレーションする。分離運用は "anti-pattern" として README で明示する。

- **trade-off**: (i) と (ii) の上位集合。短期に手が増えるが、CLI 機能投資は一回で済むのに対し、(ii) 単独を選ぶと 8 件 × N プロジェクト分の README 負荷を恒常的に背負う。
- **影響範囲**: `conoha-cli` + `conoha-cli-app-samples` 両方。
- **後方互換**: (i) と同じで、既存 43 サンプル (8 移行予定含む) は CLI 側では無改変で動く。

---

## 3. 推奨案と根拠

**推奨: (iii)**。実質は「CLI スコープは (i) を採る、そのうえで 8 サンプルを新パターンへ移行する」。

理由:

1. **実利用の方向**: 8 サンプルはいずれも "一つのアプリだが複数ホストを持ちたい" という自然な要求。プロジェクトを割る (ii) のは技術的に可能なだけで、accessory 共有・env 共有・destroy の scope が壊れる。ユーザに見せる粒度として不自然。
2. **CLI 投資の寿命**: multi-host は OIDC / admin UI / webhook-safe backend の 3 パターンで登場する汎用要求で、今後もサンプル外のユーザが踏む。一度入れれば 8 件以上の負債を先に解消する。
3. **proxy は触らない**: §3.2 で詳述するとおり、`expose` ブロックごとに proxy service を別個に登録する設計を採ると conoha-proxy 側のプロトコル (Admin API / state.db) を変更せずに済む。#98 で確定した proxy の公開契約に手を付けないまま多重化できることは大きい。
4. **サンプルの逃げ道が残る**: (iii) を採っても (ii) と排他ではない。どうしても別プロジェクトが適切なサンプル (例えば異なるチーム運用を模す場合) は手動でリポジトリを割ってよい。ただし推奨パターンは単一プロジェクト multi-host。

### 3.1 スキーマ案 (RFC レベル)

```yaml
name: gitea

# 後方互換: 単一ホスト構成は引き続き有効。expose 未使用時は現行と 1 バイト違わない。
hosts: [gitea.example.com]
web:
  service: gitea
  port: 3000

# 追加の公開ホスト。ゼロ個以上。
expose:
  - label: dex                 # proxy service 名のサフィックス。DNS-1123 label, <name> と合わせて全体 63 文字以内
    host: dex.example.com      # FQDN。hosts[] と重複してはならない
    service: dex               # compose service 名。web.service および他 expose.service と排他
    port: 5556
    blue_green: true           # default: true。false のときは slot を取らず accessories 相当
    health: {...}              # 省略時は proxy 既定 (既存 HealthSpec 形式)
    # deploy.drain_ms は app 全体 (トップレベル) に従う。per-block は将来課題 (§7)。

accessories: [db]              # expose で扱うサービスは accessories に列挙できない (validation で拒否)
```

### 3.2 proxy 側多重化戦略

- **各 `expose` ブロック → proxy service `<name>-<label>` を 1 件ずつ登録**する (`gitea-dex`, `supabase-selfhost-admin`, `hydra-python-api-auth` …)。
  - proxy から見ると従来と同じ "1 service = 1 host = 1 active target"。proxy API / state.db の互換性は完全に据え置き。
  - TLS 証明書は host ごとに ACME HTTP-01 で独立発行される (proxy の既存動作で足りる)。
- **ルート web (`web:` ブロック)** は引き続き proxy service `<name>` で登録される。既存動作と同一。
- `conoha app status` はルート service + 全 expose service の状態をまとめて表示する (§5.2)。

トレードオフ:
- 1 アプリで N+1 個の proxy service 登録が発生する (`conoha proxy services` の出力が増える)。受容。
- proxy 側で N+1 個の swap がアトミックでない (§3.3 参照)。

### 3.3 multi-service blue/green の整合性

`conoha app deploy` の 1 回の実行で、ルート web + 全 `blue_green: true` ブロックの slot を新造し、それぞれに proxy `/deploy` を順次呼ぶ。

- **順序**: (1) `expose` ブロック群を並列で build + up → (2) 各ブロックの host port を `docker port` で取得 → (3) proxy `/deploy` を **ブロックごと順次** 呼び出す (expose ブロック → ルート web の順)。**ルート web を最後にする理由**: ユーザがルート URL にアクセスした直後に補助サブドメイン (dex, auth, admin) へ redirect されるケースが多く、先に expose 側を新 slot へ切り替えておかないと「新ルート → 旧 dex」というバージョン不整合を踏む。最後にルートを swap することでクロスバージョン redirect 窓を最小化する。
- **部分失敗 (424 probe_failed)**: 途中で `/deploy` が 424 を返した場合、それまで 200 で swap した proxy service に対し **CLI が `/rollback` を逆順に発射** する。`/rollback` のエラー処理は以下:
  - **409 `no_drain_target`**: drain_deadline が閉じただけ (データは安全) なので stderr 警告を出して次のブロックの rollback に進む。既存の `ErrNoDrainTarget` (`internal/proxy/errors.go:12`) 判定は再利用するが、`cmd/app/rollback.go:71-74` の "abort-with-error" 動作は手動 rollback 用 UX であり deploy 経路には流用できない。deploy 経路用に warn-only ヘルパ (例: `rollbackWithNoDrainWarning`) を新設する。
  - **非 409 エラー (5xx / SSH timeout / ネットワーク断)**: 以降のブロックの rollback は中止し、全ブロックの proxy service 状態 (`name`, `active_target`, `phase`) を stderr に列挙して手動復旧を促す。自動復旧を試みると不整合が拡大するリスクの方が高い。
- **新 slot の回収**: 自動 rollback が完了した (あるいは警告で止めた) あとは必ず `tearDownSlotOps(ops, pf.Name, slot)` を呼んで新 slot の compose project を落とす。ルート web のみの単一ブロック時代の後処理 (`cmd/app/deploy.go:318-319`) を multi-block にもそのまま適用する。
- **スロット ID は全ブロックで共有** (同じ git short SHA)。work dir は `/opt/conoha/<name>/<slot>/` (既存)、compose project も単一 (`<name>-<slot>`) のままで、compose override が per-block の `container_name` と `ports` を吐く。

---

## 4. 影響を受ける `conoha-cli` ファイル (予想)

既存構造 (`cmd/app/`, `internal/proxy/`, `internal/config/`) を踏襲する。新規ファイルは最小限。

### 4.1 スキーマ・検証

- `internal/config/projectfile.go`
  - `ProjectFile` に `Expose []ExposeBlock` を追加。
  - 新型 `ExposeBlock{Label, Host, Service string; Port int; BlueGreen *bool; Health *HealthSpec}`。
  - `Validate()` 拡張:
    - `label` は DNS-1123 label、`<name>-<label>` が 63 文字以内。**全 expose ブロック間で `label` は一意** (衝突すると同名 proxy service への重複 Upsert になり、あとから登録した方が先行分を上書きする)。
    - `host` は `hosts[]` と重複しない FQDN。全 expose の `host` 同士も重複不可。
    - `service` は `web.service` / 他 expose の service / `accessories` と排他。
    - `port` は 1–65535。
  - `ValidateAgainstCompose()` 拡張: 全 expose ブロックの `service` が compose の services に存在することを確認。
- `internal/config/projectfile_test.go`: 上記ルールの正常系 + 境界ケース (重複 host / service、label 長、互換モード = `expose` 未指定時に既存テストがそのまま通ること)。

### 4.2 Deploy state machine

- `cmd/app/deploy.go`:
  - `runProxyDeployState` をブロック配列を跨いだ state machine に拡張。
  - `proxyDeployParams` に `Blocks []DeployBlock` を加え、ルート web + expose ブロックを正規化して渡す。
  - proxy `/deploy` 呼び出しを N 回に拡張、部分失敗時の rollback 発射を追加。
- `cmd/app/override.go`:
  - `composeOverride` シグネチャ変更または `composeOverrideFor(blocks []DeployBlock)` を新設。各ブロックごとに:
    - `services.<svc>.container_name: <name>-<slot>-<svc>`
    - `services.<svc>.ports: ["127.0.0.1:0:<port>"]`
    - `services.<svc>.env_file: [/opt/conoha/<name>/.env.server]` — **全ブロック (ルート web + expose) 共通で同一の env_file を注入する**。「1 アプリ = 1 `.env.server`」(#94) の原則をそのまま踏襲。per-block 分離の要否は §7 Q-env で議論。
  - accessory ネットワーク接続はこれまでどおり。
- `cmd/app/deploy_ops.go`: `DeployOps` に `Proxy()` を複数回呼ぶパスが増えるだけ。内部変更は薄い。

### 4.3 init / destroy / status / rollback

- `cmd/app/init.go` `runInitProxy`:
  - ルート service `<name>` に加え、`expose` ブロックごとに `proxypkg.Client.Upsert` を呼ぶ。
  - service 登録順は決め打ち (ルート → expose[0] → expose[1] …) でログを stderr に流す。
  - ロールバック: 途中失敗時は登録済み service を順に `Delete` する。
- `cmd/app/destroy.go`:
  - `expose` ブロック分の proxy service を `Delete` してからルート service を削除、さらに `/opt/conoha/<name>` を `rm -rf`。
- `cmd/app/status.go`:
  - proxy `/v1/services` を app 名プレフィックス (`<name>`, `<name>-*`) でフィルタして表示。各 service の phase / active_target / drain_deadline を 1 表に。
- `cmd/app/rollback.go`:
  - `--target=<label>` で単一ブロックの rollback をサポート。無指定時は全ブロック (ルート web も含む) 逆順 rollback。§7 Q-rollback 参照。

### 4.4 docs / recipes / skill

- `README.md` / `docs/superpowers/specs/2026-04-20-conoha-proxy-deploy-design.md` の「非スコープ: 複数 VPS / accessory きめ細かい」節に本 RFC への参照を追加 (RFC 単独の文書差分ではなく、実装 PR で反映)。
- `docs/skills/` のレシピに expose ブロックのサンプルを追加。
- `docs/release-checklist.md` の smoke シナリオに "multi-host sample (gitea) で Dex サブドメインが 200 を返すこと" を 1 行追加 (実装 PR で)。

### 4.5 app-samples (別リポジトリ)

`conoha-cli-app-samples` 側の変更はここでは箇条書きのみ:

1. `gitea/conoha.yml`: `expose: [{label: dex, host: dex.example.com, service: dex, port: 5556, blue_green: false}]`。`accessories` から `dex` を外す。
2. `outline/conoha.yml`: 同上。
3. `rails-mercari/conoha.yml`: Dex + `web` (Rails) を expose に。`web` は `blue_green: true`。nginx はルート web のまま。
4. `hydra-python-api/conoha.yml`: `expose: [{label: auth, host: auth.example.com, service: hydra, port: 4444}]` (hydra 本体は blue/green false)。**注**: compose service 名は実リポジトリの `conoha-docker-compose.yml` で確認する — Hydra は `hydra-public` / `hydra-admin` のように Public / Admin エンドポイントを分けていることが多く、公開したいのは `:4444` を listen する Public 側。サービス名の確定は Phase 5 の移行 PR で行う。
5. `nextjs-fastapi-clerk-stripe/conoha.yml`: `expose: [{label: api, host: api.example.com, service: backend, port: 8000}]`、webhook 受信ホストは `api.example.com` に移す旨 README で明示。
6. `supabase-selfhost/conoha.yml`: `expose: [{label: admin, host: admin.example.com, service: studio, port: 3000, blue_green: false}]`。
7. `quickwit-otel/conoha.yml`: `expose: [{label: otel, host: otel.example.com, service: otel-collector, port: 4318, blue_green: false}]`。gRPC `:4317` は引き続き内部のみ (README で限界明示、次段 RFC へ譲渡)。
8. `dify-https/conoha.yml`: `expose` に `api`, `web` (Dify フロント) を blue/green true で追加。nginx はルート web のまま。

各 README の「既知の制限」節を更新、CLI 最低バージョンを明示 (§6 の phase 3 以降)。

---

## 5. 受け入れ基準

### 5.1 CLI 単体

- [ ] `internal/config/projectfile_test.go`: `expose` 未指定の既存テストがそのままパス。新規に重複 host / 重複 service / label 長超過 / compose 不在の各エラーケースのテストが追加される。
- [ ] `expose` 1 件付きの fixture で `conoha app init <server>` を実行すると、proxy service が `<name>` と `<name>-<label>` の **2 件** 登録されることが `internal/proxy/admin_test.go` レベルで確認できる (fake executor でリクエストを収集)。
- [ ] `expose` 1 件付きの fixture で `conoha app deploy <server>` を実行すると、
  - compose override が web と expose 両方の container_name / ports / env_file を吐く (gold file テスト)
  - proxy `/deploy` が 2 回呼ばれる (順序: expose → root web)
  - 片方の `/deploy` が 424 を返すと、既に swap 済みの側に自動で `/rollback` が発射される
- [ ] `conoha app status <server>` の json 出力 (`--format json`) に root と expose の service 配列が揃う。
- [ ] `conoha app destroy <server>` 後、`<name>` と `<name>-<label>` の proxy service が **両方** 消え、`/opt/conoha/<name>` が存在しない。
- [ ] `tests/e2e/` シナリオに multi-host fixture (`tests/e2e/fixtures/multi-host/conoha.yml`) を 1 件追加、`e2e` ジョブがそのシナリオを含んで通る (スクリプトは phase 3 で追加)。

### 5.2 app-samples 側

- [ ] `gitea`, `outline`, `rails-mercari` の Dex ブラウザ OIDC フローが、`dex.example.com` でログインし元アプリにリダイレクト戻って 200 を返すことを README 手順で再現できる (smoke; 実 DNS が必要なため CI には載せず `docs/release-checklist.md` に追記)。
- [ ] `hydra-python-api` の authz code flow が browser 経由で完走する (同上)。
- [ ] `nextjs-fastapi-clerk-stripe` で Stripe CLI から `api.example.com/webhooks/stripe` に test event を流し、`backend` ログに検証成功が出る。
- [ ] `supabase-selfhost` で `admin.example.com` から Studio に到達する。
- [ ] `quickwit-otel` で `curl https://otel.example.com/v1/traces` (HTTP OTLP) に 200 が返る。
- [ ] `dify-https`, `rails-mercari` で `conoha app deploy` 2 回目以降に内部 (`api`, `web`) のイメージが新 slot で再ビルドされていることを `docker images` タイムスタンプで確認。

### 5.3 後方互換

- [ ] `expose` を持たない既存 43 サンプル (今の main) は CLI 変更後も無改変でデプロイできる。`tests/e2e/` の既存シナリオが全部パス。

---

## 6. フェーズ分割

CLI 機能 → app-samples の順で段階リリースする。CLI は v0.3.0 にバンプ (後方互換ではあるがスキーマ追加のため minor bump)。

### Phase 0: RFC マージ (本 PR)

- 本ファイルのマージのみ。`Status: Proposed` のまま残し、§7 オープン質問に対するユーザ回答を待つ。
- CLI コード・サンプル変更を **含まない**。

### Phase 1: スキーマ + 検証

- `internal/config/projectfile.go` / `projectfile_test.go` のみ。
- `ExposeBlock` 型と `Validate()` / `ValidateAgainstCompose()` の拡張。
- 他コードパスは `len(pf.Expose) == 0` を仮定しているため動作変更なし。
- CI: lint + unit test green。

### Phase 2: init / destroy への拡張

- `cmd/app/init.go` / `destroy.go` で N+1 service の upsert / delete を実装。
- `app status` はまだ拡張せず、既存表示に expose 分を単に追記する最小形。
- 既存テスト: fake proxy client にメソッド呼び出し回数の assertion を追加。

### Phase 3: multi-service deploy

- `cmd/app/deploy.go` / `override.go` / `deploy_ops.go` を拡張。
- proxy `/deploy` を N+1 回呼ぶ state machine、部分失敗時の逆順 rollback。
- `tests/e2e/` に multi-host シナリオ fixture + 実行ステップを追加。既存 fixture はそのまま。
- このフェーズで CLI v0.3.0-rc を tag。リリースノートに `expose` の追加とサンプル移行の予告を記載。

### Phase 4: status / rollback / docs

- `cmd/app/status.go` で全 service を 1 表にまとめる。**併せて `--format json` フラグを新設**する — 現状 (`cmd/app/status.go:72-84`) は stderr への人間可読出力のみで JSON 経路が無いため、acceptance §5.1 の JSON 出力要件はこの新フラグで満たす。
- `cmd/app/rollback.go` に `--target=<label>` を追加。
- README / specs / recipes / skill を更新。
- `docs/release-checklist.md` の smoke シナリオに multi-host の 1 行追加。
- CLI v0.3.0 を tag。

### Phase 5: app-samples 移行 (別リポジトリ、別 issue)

- 1 サンプル = 1 PR、8 PR 合計。Phase 4 の CLI リリース後に開始。
- 各 PR は conoha.yml 変更 + README 既知制限節の書き換え + 実 VPS smoke ログ貼り付け。
- 全 PR マージ後、`conoha-cli-app-samples` に v0.3.0 タグを打って README で CLI 最低バージョンを v0.3.0 と明記。

---

## 7. オープン質問 (マージ前レビュー要)

このセクションは設計の分岐点で、マージ前に owner の裁定が必要なもの。RFC 本体を承認する前にそれぞれ Yes/No または選択を伝えてほしい。

- **Q-naming**: proxy service 名を `<name>-<label>` (例: `gitea-dex`) とするか、`<label>.<name>` (例: `dex.gitea`) とするか。前者は既存 DNS-1123 バリデーション (`dnsLabelRe`) と相性がよく、`conoha proxy services` 出力でソートが見やすい (同一 app が隣接)。後者は "subdomain感" が出るが長さ制約と衝突しやすい (合計 63 文字制限)。**推奨: 前者 (`<name>-<label>`)**。補足: `<name>-<label>` が compose service 名 (例: gitea アプリ内の `dex` service) と文字列として一致するケースがある — 両者は別レイヤ (proxy service vs compose service) だが、ログを読む際の混乱を避けるため README / status 出力では `proxy service: gitea-dex (compose service: dex)` のように明示ラベル付けする。
- **Q-rollback**: multi-service deploy の途中失敗 (例: `/deploy` 2 本目が 424) のとき、CLI が 1 本目の成功 service に対し **自動で `/rollback` を打つか、打たずに警告のみにするか**。自動 rollback は drain_deadline 内に収まれば安全だがエッジ (drain_deadline ぎりぎりで 2 本目が落ちたケース) で 409 no_drain_target を踏む。**推奨: 自動 rollback を試行し、409 の場合のみ stderr 警告に落として次ブロックの rollback に進む。非 409 のエラー (5xx / SSH 断) が出た時点で後続 rollback は中止し、手動復旧を促すメッセージを出す (§3.3 参照)**。実装面では既存 `ErrNoDrainTarget` 判定は流用できるが、`cmd/app/rollback.go:71-74` の abort 動作は手動 rollback 用 UX なので deploy 経路には使えない — warn-only ヘルパを新設する。
- **Q-atomic**: N+1 proxy service の swap は原理的に atomic にできない (proxy はサービスごと独立)。DNS/ブラウザのキャッシュも相まって、ユーザから見て「ルートは新版だけど dex サブドメインは一瞬旧版」は起こる。**ユーザに許容してもらうか、または "最初に upstream だけ差し替え、最後に DNS 切替" のような 2 段階を要求するか**。**推奨: 許容。理由: 対象ドメインが別 FQDN なので同一 URL 内の一貫性は問われない。ドキュメントで "N サービスが順次 swap される" 旨を明示するだけに留める**。
- **Q-env**: `.env.server` は現状アプリ 1 個 = 1 ファイル (`/opt/conoha/<name>/.env.server`、#94)。`expose` ブロックごとに別 env を注入したい需要はあるか (例: backend だけ別の STRIPE_WEBHOOK_SECRET)。**推奨: Phase 4 時点では共有のまま。必要なら `expose.env_file` キーで個別オーバーライドを後続 RFC で扱う**。
- **Q-grpc**: quickwit-otel の OTLP gRPC (`:4317`) を受けるために conoha-proxy 側で h2c 透過または L4 pass-through を導入すべきか。**推奨: 本 RFC 外。別 RFC (`proxy grpc support`) で扱う。quickwit-otel のサンプルは HTTP OTLP (`:4318`) だけ subdomain 化して gRPC は "internal only" 制限を明記**。
- **Q-version-gate**: app-samples 側で `expose` 付きの conoha.yml は v0.3.0 未満の CLI で parse するとどう失敗するか。yaml.v3 は未知フィールドを無視するデフォルトなので、`expose` は **silently 無視される** (既存 known-limitation に戻る)。README で CLI 最低バージョンを明記するだけで十分か、それとも `conoha.yml` に `conoha_version: ">=0.3.0"` のようなゲートを追加するか。**推奨: README 明記のみ。将来的にゲートが必要になれば別 issue**。

---

## 8. 後続 issue 案 (PR 本文にも同内容を記載)

Phase 0 マージ後、以下の GitHub issue を立てて実装フェーズに入る (`conoha-cli` リポジトリ)。

1. **issue: `feat(config): expose-block schema + validation (phase 1 of subdomain-split)`**
   - labels: `enhancement`, `rfc:subdomain-split`, `area:config`
   - body: §4.1 の実装、§5.1 のうち projectfile_test.go 範囲をカバー。
2. **issue: `feat(proxy): init/destroy multi-service registration (phase 2)`**
   - labels: `enhancement`, `rfc:subdomain-split`, `area:proxy`
   - depends-on: (1)
3. **issue: `feat(deploy): multi-block blue/green deploy (phase 3)`**
   - labels: `enhancement`, `rfc:subdomain-split`, `area:deploy`
   - depends-on: (2)
4. **issue: `feat(ux): status/rollback/docs for expose blocks (phase 4)`**
   - labels: `enhancement`, `rfc:subdomain-split`, `area:ux`, `docs`
   - depends-on: (3)
5. **issue: `test(e2e): multi-host fixture + scenario`**
   - labels: `testing`, `rfc:subdomain-split`
   - depends-on: (3)、(4) と並列可
6. **issue (app-samples repo): `refactor: migrate 8 samples to expose blocks`**
   - labels: `samples`, `rfc:subdomain-split`
   - depends-on: conoha-cli side (4) + v0.3.0 release tag
   - 1 サンプル = 1 PR の原則。8 PR ぶらさげ。

すべての issue タイトルに `(refs #<this-PR>)` を付け、closed 後に本 RFC に backlink を残す。

> **Pre-flight**: 起票前に `gh label list` で `rfc:subdomain-split`, `area:config`, `area:proxy`, `area:deploy`, `area:ux`, `samples` の各ラベルが repo に存在することを確認する。未作成のものは `gh label create <name> --color <hex> --description <desc>` でまとめて用意してから issue を立てる。

---

## 9. 参考

- 先行 spec: `docs/superpowers/specs/2026-04-20-conoha-proxy-deploy-design.md` (#98)
- 先行 spec: `docs/superpowers/specs/2026-04-21-no-proxy-mode-design.md` (#103)
- 先行 spec: `docs/superpowers/specs/2026-04-23-app-env-redesign.md` (#94)
- 先行 spec: `docs/superpowers/specs/2026-04-23-e2e-tests-design.md` (#96)
- memory: `project_issue_96_status.md` (2026-04-24) — 8 サンプルの限界一覧の出所
- サンプル `conoha.yml` (既知制限コメント): `conoha-cli-app-samples/{gitea,outline,rails-mercari,hydra-python-api,nextjs-fastapi-clerk-stripe,supabase-selfhost,quickwit-otel,dify-https}/conoha.yml`
