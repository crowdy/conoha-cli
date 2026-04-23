# End-to-end 統合テスト設計

**Date**: 2026-04-23
**Status**: Proposed
**Owner**: t-kim
**Related**: #96 (this issue), #98 (proxy-deploy refactor), #99 (unit tests)

## 1. 背景

`2026-04-20-conoha-proxy-deploy-design.md` §9 で "E2E は v0.2.0 tagging 前に必要" と明記しつつ、初期 refactor では実装を見送った。既存テストは単位 (unit) のみで以下をカバーしていない:

- 実 conoha-proxy コンテナとの Admin API ハンドシェイク
- blue/green swap 実動作 (drain deadline、`active_target` / `draining_target` の遷移)
- rollback window (drain 内・drain 切れ後の挙動分岐)
- TLS 発行 (Let's Encrypt HTTP-01 フロー) ← 外部依存のため今回は対象外

v0.2.x 系を安心してタグ付けするには、CLI ↔ proxy 間の契約が壊れたとき気付ける E2E テストが要る。

## 2. ハーネス選択

### 2.1 選択肢

| 選択肢 | 実行環境 | Pros | Cons |
|---|---|---|---|
| (A) Docker-in-Docker (DinD) on GitHub Actions | ephemeral, self-hosted runner 不要 | 無料、PR gating 可能、再現可能 | ACME/DNS 不可 (TLS 経路はスタブ)、Compose v5 コンテナレイヤの深さ制限 |
| (B) 専有テスト VPS (ConoHa) | 実サーバ | 実環境と同一、ACME 検証可 | 固定コスト、SSH key/DNS レコード管理、CI ↔ VPS の secrets 管理 |
| (C) ハイブリッド (A で golden path、B で TLS/smoke) | 両方 | (A) 速度 + (B) 本物 | 運用コスト倍 |

### 2.2 決定: **(A) DinD on GitHub Actions を primary、(B) manual smoke を README 化**

理由:
- PR ごとに走る CI が blue/green 契約の回帰を検知できるのが最大の目的。これは (A) で十分。
- TLS 発行・DNS 解決は proxy 側の issue (crowdy/conoha-proxy) で担保すべき。CLI repo のスコープは "CLI が Admin API を正しく呼べるか" に絞る。
- 専有 VPS は固定コスト・secrets 管理負担が大きい。release candidate の手動検証フローに回す (§7 参照)。

## 3. テスト対象のシナリオ

既存 #96 スコープ 8 項目を以下に再整理:

| # | シナリオ | 検証ポイント | DinD で可能 |
|---|---|---|---|
| 1 | `proxy boot` | admin socket 起動、`/version` `/readyz` 応答 | ✓ |
| 2 | `app init` (with sample conoha.yml) | `GET /v1/services/<name>` で upsert 結果を取得 | ✓ |
| 3 | `app deploy` 1 回目 | `active_target` set、`GET /` が 200 | ✓ (TLS 無し HTTP) |
| 4 | `app deploy` 2 回目 | blue/green swap、`draining_target` set、drain 後に旧 slot ダウン | ✓ |
| 5 | `app rollback` (drain 窓内) | active が即戻る | ✓ |
| 6 | `app rollback` (drain 窓外) | `no_drain_target` エラー | ✓ |
| 7 | `app destroy` | proxy から service 消滅、slot work dir 消滅 | ✓ |
| 8 | `proxy remove` | container 消滅、data dir はオプションで保持 | ✓ |

§3 以外でカバーすべき edge:
- 9. `app init` を 2 回実行 → idempotent upsert
- 10. `app deploy` 直後に `app list` → 新 service が列挙される (#95 回帰 guard)
- 11. `app env set` → `app deploy` → 新 slot の web コンテナが env を受け取る (#94 回帰 guard)
- 12. `app destroy` 直後に `app destroy` → "not initialized" に落ち着く (既存挙動 lock-in)
- 13. **legacy env path → `app env migrate`**: 事前に legacy パス (`/opt/conoha/<app>.env.server`) にだけファイルを置いた状態で `app env migrate` を走らせ、new パス (`/opt/conoha/<app>/.env.server`) に移動する + 0600 になっていることを `stat -c '%a'` で確認 (#94 spec §6 acceptance)。
- 14. **legacy env path → `app env list`/`set` の deprecation warning**: legacy-only 状態で `app env list` を叩き stderr に warning が出ること、`app env set` が exit 2 で拒否されること (PR #139 data-loss guard の回帰 guard)。v0.3 で legacy fallback を外す予定なので、その切替 PR のときにこのシナリオを反転させる契約になっている。

## 4. ハーネスアーキテクチャ

### 4.1 DinD ランナー構成

```
GitHub Actions job "e2e":
  runs-on: ubuntu-latest
  services:
    docker: (built-in via actions/setup-docker or runner default)
  steps:
    1. checkout
    2. go build -o ./bin/conoha ./
    3. docker network create conoha-e2e
    4. docker run -d --name target --network conoha-e2e \
         --privileged \
         <target-image>         # systemd + docker + sshd inside
    5. ssh-keygen を生成、authorized_keys に登録
    6. export CONOHA_SSH_INSECURE=1 (TOFU プロンプト回避、#101)
    7. ./bin/conoha app init → deploy → swap → rollback → destroy を
       スクリプトで順次実行
    8. 各 step の stdout/stderr を assert (grep + exit code)
```

**target-image 選定**: ConoHa VMI の模倣を狙う `vmi-docker-29.2-ubuntu-24.04-amd64` が一番近いが、これは ConoHa プラットフォーム専用で pull できない。代替として:

- `ubuntu:24.04` に docker + openssh-server を apt で足した ephemeral イメージを CI で build。
- `tests/e2e/Dockerfile.target` を追加し、CI で `docker build` してから `docker run` する。

**conoha-proxy の導入**: CLI は SSH 越しに `docker compose` で proxy を起こす (`conoha proxy boot`) ので、target コンテナに docker daemon が動いていればそのまま走る (DinD の nesting が 1 層)。ただし nesting 制限 (cgroup v2 fsmount 等) を踏む可能性があるので、最初の PoC で `--privileged` + `docker:dind` image の派生を試す必要あり。

### 4.2 テストドライバの形

Go の test 内からシェル経由で `./bin/conoha ...` を叩くより、スクリプト (`tests/e2e/run.sh`) + Go の薄いラッパで assertion を実行する方が stack trace が読みやすい。

候補レイアウト:

```
tests/e2e/
├── Dockerfile.target        # ubuntu + docker + sshd
├── compose.yml              # target container + CLI side network
├── run.sh                   # ./bin/conoha invoke と assert を並べた bash
├── fixtures/
│   ├── conoha.yml           # sample app
│   └── docker-compose.yml   # sample app compose
└── e2e_test.go              # go test が run.sh を spawn; build tag 'e2e'
```

`go test ./tests/e2e/ -tags e2e` で明示的に起動。普通の `go test ./...` には含まれない (PR ごとに走らせる別 job)。

### 4.3 実行時間バジェット

目標: 1 回 5 分以内 (`go test -race ./...` が ~2 分なので合計 7 分以内)。内訳予想:
- target 起動 + docker install: 60-90s
- proxy boot + 各 CLI step: 90-120s (drain 窓 30s x 2 が支配的)
- teardown: 10s

drain 窓を設計値 30s → テスト設定 2s に縮めるため `--drain-ms 2000` を `app deploy` で指定する (既存フラグ、spec §3.2)。`--drain-ms` は test-only override ではなく正式サポート flag として維持される前提 (削除するとこのバジェットが静かに壊れる)。

### 4.4 フォールバック計画 (DinD PoC が詰まった場合)

§4.1 で触れた nesting 制限 (cgroup v2 / fsmount / docker-in-docker の kernel 依存) を Phase 1 の PoC で踏んだ場合、以下の順で escalate する:

1. **まず**: `docker:dind` base image を派生して `--privileged` + `--cgroupns=host` で再試行。GitHub Actions の `ubuntu-latest` は cgroup v2 + unified mode なので、systemd not-required 経路 (サービス起動ではなく `dockerd --host=tcp://` を直接叩く) で回避できることが多い。
2. **ダメなら**: `sysbox-runc` ランタイム導入を検討 (自己ホスト runner 必要 → コスト増)。
3. **それもダメなら**: 選択肢 (C) ハイブリッドに escalate — §2.2 を修正し、Phase 1-2 の基本シナリオは専有テスト VPS (ConoHa 最安 flavor × 1 台、月額固定) で回し、§3 #9-12 の edge は引き続き DinD で走らせる構成。PR ごと gating は維持したいので専有 VPS を semaphore で直列化する。

**判断基準**: Phase 1 PR を 2 回 reroll しても DinD で boot/init/deploy が通らない場合、その PR は advisory のまま merge して次 PR で (C) に切り替える。PoC に 1 週間以上かけない。

## 5. Secrets / 環境変数

DinD 内で完結するため ConoHa 本番 secrets (tenant ID 等) は不要。必要な env:
- `CONOHA_SSH_INSECURE=1` — TOFU プロンプト回避。
- `CONOHA_NO_INPUT=1` — `app destroy` / `app reset` の prompt スキップ (既存)。

Actions job には追加 secret 不要 → public fork PR からも走れる。

## 6. CI への組込

`.github/workflows/ci.yml` に新 job `e2e` を追加:

```yaml
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.26' }
      - run: go build -o bin/conoha ./
      - run: docker build -t conoha-e2e-target -f tests/e2e/Dockerfile.target tests/e2e/
      - run: bash tests/e2e/run.sh
    timeout-minutes: 10
```

既存 `test` / `lint` / `build` と並列。`e2e` を **required status check** にするかは別判断 (初期は advisory、安定したら required)。

## 7. 手動 / VPS smoke (release gate)

release candidate (`git tag v0.2.0-rcN`) タグ時:
1. ConoHa 上に新規 VPS を作成 (`conoha server create`)。
2. DNS A レコードをテスト用ドメインに振る。
3. `conoha proxy boot` → `app init` → `app deploy` を fresh conoha.yml で実行。
4. HTTPS (TLS) が発行されて 200 を返すことをブラウザ確認。
5. OK ならタグを push。

この手順は `docs/release-checklist.md` (新規) に箇条書きで明文化する。本 PR では言及に留め、実文書は別 PR。

### 7.1 VPS smoke でしか catch できない CLI バグ種別

DinD で擬似できない、CLI レイヤで real VPS 固有に壊れうるクラスを明示する。release-checklist PR を書くときのチェック対象:

| 種別 | 具体例 | DinD で不可な理由 |
|---|---|---|
| **cloud-init timing** | `conoha server create` 直後に `conoha proxy boot` が SSH connect できずタイムアウトする regression (cloud-init 完了検知の健全性) | DinD の target は起動時に SSH が立ち上がっていて cloud-init が無い |
| **systemd unit の競合** | `conoha proxy boot` が Docker daemon を assume する一方、ConoHa VMI では `docker.socket` の activation timing と proxy の systemd unit が race する可能性 | DinD target は systemd 無し、または簡素化された init で動かしている |
| **SSH known_hosts TOFU (#101)** | 初回接続時の known_hosts 書き込み、2 回目以降の host key verify エラー経路 (プロバイダ側 rebuild で fingerprint 変わったケース) | DinD 内では `CONOHA_SSH_INSECURE=1` を強制しており TOFU 経路を通らない |
| **ACME レートリミット** | Let's Encrypt 本番 rate limit に引っかかって発行できない状態で `app deploy` が正しく "TLS pending → degraded" を report するか | proxy repo 側で ACME stub を使うのが一般的、DinD でも本物 ACME は呼ばない |
| **DNS 伝播** | `conoha dns` で追加した A レコードがまだ NS に乗っていない間の `app init` 挙動 (hosts 未到達を warn するか、fail 早期にするか) | DinD には DNS なし、compose の `extra_hosts` で static 解決 |
| **ConoHa API の actual response shape** | flavor / image / key-pair ID の形が API 側で変わった場合の CLI 側 parse 耐性 | compute API はスタブされる |

release-checklist PR ではこの表を mirror して、各項目について "何を観察すれば OK と言えるか" の手順を書き下す。

## 8. 実装フェーズ分割

本 spec 承認後、実装は以下に分割:

- **Phase 1 (PR1)**: `tests/e2e/Dockerfile.target` + 最小 `run.sh` で #1-3 (boot/init/deploy 1 回目) を pass。CI job 追加だが advisory。
- **Phase 2 (PR2)**: #4-6 (swap, rollback 両ケース) 追加。
- **Phase 3 (PR3)**: #7-12 (destroy, idempotency, list 回帰 guard, env 流れ) 追加。
- **Phase 4 (PR4)**: `e2e` を required status check に昇格。release checklist 文書化。

各フェーズを独立 PR にする理由: DinD + systemd 系は GitHub runner の kernel バージョン依存で起動失敗することがあり、段階的に調整できるほうが安全。

## 9. 受け入れ基準

本 spec 自体の受け入れ:
- [ ] ハーネス選択 (§2.2) に合意。
- [ ] テスト対象 §3 の 14 項目に漏れがない (特に #13/#14 の legacy env 移行経路)。
- [ ] 実装フェーズ §8 の分割がレビュー通過。
- [ ] §4.4 のフォールバック基準 (Phase 1 PR 2 回で詰まったら (C) ハイブリッド) に合意。
- [ ] §7.1 の VPS-only バグ種別表を release-checklist PR の元ネタとして承認。

Phase 4 完了時の最終受け入れ:
- [ ] `.github/workflows/ci.yml` の `e2e` が required status check。
- [ ] 全 12 シナリオが安定して 5 分以内 pass。
- [ ] `docs/release-checklist.md` で VPS smoke 手順が文書化。

## 10. 非ゴール

- TLS / ACME / Let's Encrypt 発行の実テスト (proxy repo スコープ)。
- 実 ConoHa API 呼び出し (auth は compute API 依存、今回の対象は CLI ↔ proxy)。
- パフォーマンス / スケール (drain 後の slot 削除遅延など)。

## 11. 参考

- 先行 spec: [`2026-04-20-conoha-proxy-deploy-design.md`](2026-04-20-conoha-proxy-deploy-design.md) §9
- 関連 issue: #96 (本), #99 (unit tests)
- 既存 unit 層: `internal/proxy/admin_test.go` (fakeExecutor), `cmd/proxy/observability_test.go` (#140)
