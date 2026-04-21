# README / Skill Refresh for Proxy + No-Proxy Deploy Modes

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refresh user-facing documentation so that both `conoha app deploy` modes — the conoha-proxy blue/green path (default) and the newer `--no-proxy` flat path (shipped in #102/#103 on 2026-04-09) — are documented symmetrically, with accurate flags, conoha.yml schema references, and mode-marker semantics.

**Architecture:** Docs-only refresh across two repositories:
1. `crowdy/conoha-cli` — `README.md` (JA), `README-en.md`, `README-ko.md`
2. `crowdy/conoha-cli-skill` — `SKILL.md`, `recipes/single-server-app.md` (split into proxy + no-proxy recipes; update SKILL index)

No code changes. Verification is visual + ensuring JA/EN/KO stay structurally aligned and that every flag/subcommand claim can be traced back to `conoha app <sub> --help` output.

**Tech Stack:** Markdown. No build-time validation, so self-review against live `--help` output is the only quality gate.

---

## Pre-Work: Reference Data

Before editing any file, capture the current truth from the binary. **Run these once at the start and keep the output in the scratch area** — every task below cites values from it.

```bash
cd /root/dev/crowdy/conoha-cli
go build -o /tmp/conoha .
/tmp/conoha version
for sub in init deploy destroy logs status stop restart rollback env list; do
  echo "### conoha app $sub ###"
  /tmp/conoha app $sub --help 2>&1
  echo
done > /tmp/conoha-app-help.txt
```

**Known truths (already verified for this plan):**

- App subcommands present: `init / deploy / rollback / logs / status / stop / restart / env / destroy / list`
- Every lifecycle subcommand **except `list`** carries the mutually-exclusive pair `--proxy` / `--no-proxy` (string: `"force {mode} mode, overriding server marker"`).
- `env` has four sub-subcommands: `get / list / set / unset`.
- `deploy --slot <id>` is user-facing (regex `[a-z0-9][a-z0-9-]{0,63}`; reuse semantics documented inline).
- `rollback --drain-ms <int>` is user-facing.
- Mode marker: `/opt/conoha/<app>/.conoha-mode` — written by `init`, read by every other subcommand (`cmd/app/mode.go:46-57`). Absence → `ErrNoMarker`. Mismatch between marker and `--proxy`/`--no-proxy` flag → formatted error telling the user to `conoha app destroy` + re-init (`cmd/app/mode.go:67-79`).
- `conoha.yml` schema (`internal/config/projectfile.go:15-23`):
  ```
  name (required, DNS-1123 label, ≤63 chars)
  hosts (required, non-empty list, no duplicates)
  web.service (required, must exist in compose file)
  web.port (required, 1-65535)
  compose_file (optional; auto-detected from: conoha-docker-compose.yml, conoha-docker-compose.yaml, docker-compose.yml, docker-compose.yaml, compose.yml, compose.yaml)
  accessories (optional, list of compose service names)
  health.{path, interval_ms, timeout_ms, healthy_threshold, unhealthy_threshold} (optional)
  deploy.drain_ms (optional)
  ```
- `conoha.yml` is **required** for proxy mode, **not used** by no-proxy mode.
- Apps with different modes can coexist on the same VPS (different `<app-name>`s, different marker files).

---

## File Structure

**conoha-cli repo** (branch: `docs/readme-noproxy-refresh`):
- Modify: `README.md` (lines ~162-212 — the "2つのデプロイモード" + "アプリデプロイ" block)
- Modify: `README-en.md` (mirror section)
- Modify: `README-ko.md` (mirror section)

**conoha-cli-skill repo** (clone fresh at `/tmp/conoha-cli-skill`, branch: `docs/noproxy-refresh`):
- Modify: `SKILL.md` — app commands table, add `--no-proxy`/`--proxy` note, add missing subcommands (`destroy`, `rollback`, `list`, `env`), fix the stale "Docker環境を初期化" description for `init`.
- Rewrite: `recipes/single-server-app.md` — retire pre-proxy "tar+SSH flat layout" narrative. Present **two parallel flows**: proxy blue/green (the primary recommendation) and no-proxy flat. Cross-reference from SKILL.md index with both paths.

**Decision on recipes/:** Keep as one file with a mode-switch at the top. A single recipe is easier to maintain than two near-duplicates and matches the README pattern of showing both modes side by side.

---

## Task 1: Clone skill repo & capture current state

**Files:**
- Create: `/tmp/conoha-cli-skill` (git clone)

- [ ] **Step 1: Clone the canonical skill repo**

```bash
cd /tmp
rm -rf conoha-cli-skill
git clone https://github.com/crowdy/conoha-cli-skill.git
cd conoha-cli-skill
git checkout -b docs/noproxy-refresh
```

Expected: clone succeeds, branch created. Confirm `ls` shows `SKILL.md` and `recipes/`.

- [ ] **Step 2: Record current `SKILL.md` and recipes line counts for later diff sanity**

```bash
wc -l SKILL.md recipes/*.md
```

Expected: counts roughly matching `/root/dev/crowdy-cc/skills/conoha-cli-skill/` (the local backup copy — both should be in sync at a94a717 / 2026-04-02).

---

## Task 2: Rewrite README.md (JA) — deploy modes section

**Files:**
- Modify: `/root/dev/crowdy/conoha-cli/README.md` (replace lines 162-212 wholesale — the "### 2 つのデプロイモード" header through the end of the conoha-proxy blue/green section)

- [ ] **Step 1: Verify base branch**

```bash
cd /root/dev/crowdy/conoha-cli
git checkout main
git pull --ff-only origin main
git checkout -b docs/readme-noproxy-refresh
```

Expected: clean working tree on new branch.

- [ ] **Step 2: Replace the section**

Replace lines 162-212 (from `### 2 つのデプロイモード` through the closing of the conoha-proxy rollback code block, inclusive) with the block below. Use Edit with enough surrounding context — specifically, the `old_string` should start at `### 2 つのデプロイモード` and end at the backtick-fenced `conoha app rollback my-server` code block that precedes `## Claude Code スキル`.

```markdown
## アプリデプロイ

`conoha app` は同一 VPS 上で共存可能な 2 つのデプロイモードを提供します。どちらのモードも `conoha app init` で初期化した時点でサーバー側にマーカー (`/opt/conoha/<name>/.conoha-mode`) が書かれ、以降の `deploy` / `status` / `logs` / `stop` / `restart` / `destroy` / `rollback` は自動的にそのモードで動作します。`--proxy` / `--no-proxy` フラグを明示した場合はフラグが優先され、マーカーと不一致ならエラーになります（`conoha app destroy` → 再 `init` で切り替え可能）。

| モード | 既定 | 用途 | レイアウト | `conoha.yml` | `conoha proxy boot` | DNS / TLS |
|---|:-:|---|---|:-:|:-:|:-:|
| **proxy** (blue/green) | ✓ | ドメイン + Let's Encrypt TLS の公開アプリ | `/opt/conoha/<name>/<slot>/` (blue/green スロット) | 必要 | 必要 | 必要 |
| **no-proxy** (flat) |  | テスト、内部・開発 VPS、非 HTTP サービス、ホビーアプリ | `/opt/conoha/<name>/` (フラット単一ディレクトリ) | 不要 | 不要 | 不要 |

### proxy モード (既定): conoha-proxy 経由 blue/green

[conoha-proxy](https://github.com/crowdy/conoha-proxy) が Let's Encrypt HTTPS、Host ヘッダールーティング、drain 窓内の即時ロールバックを提供します。

1. レポジトリルートに `conoha.yml` を作成：

   ```yaml
   name: myapp                   # DNS-1123 ラベル (小文字英数字とハイフン、1-63 文字)
   hosts:
     - app.example.com           # 複数指定可、重複不可
   web:
     service: web                # compose ファイル内のサービス名と一致必須
     port: 8080                  # コンテナ側のリッスンポート (1-65535)
   # --- 以下は任意 ---
   compose_file: docker-compose.yml   # 未指定時は conoha-docker-compose.yml → docker-compose.yml → compose.yml の順で自動検出
   accessories: [db, redis]           # web と同じネットワークに接続する副次サービス
   health:
     path: /healthz
     interval_ms: 1000
     timeout_ms: 500
     healthy_threshold: 2
     unhealthy_threshold: 3
   deploy:
     drain_ms: 5000                   # 旧スロットを落とすまでの drain 窓 (ミリ秒)
   ```

2. プロキシコンテナを VPS にブート：

   ```bash
   conoha proxy boot my-server --acme-email ops@example.com
   ```

3. DNS の A レコードを VPS に向ける（Let's Encrypt HTTP-01 検証に必要）。

4. アプリを proxy に登録してデプロイ：

   ```bash
   conoha app init my-server --app-name myapp
   conoha app deploy my-server --app-name myapp
   ```

5. ロールバック（drain 窓内のみ、旧スロットへ即時戻し）：

   ```bash
   conoha app rollback my-server --app-name myapp
   ```

`deploy --slot <id>` で slot ID を固定できます (規則: `[a-z0-9][a-z0-9-]{0,63}`、既定は git short SHA または timestamp)。同名 slot を再利用すると作業ディレクトリを削除してから再展開します。

### no-proxy モード: フラット単一スロット

`conoha.yml` / proxy / DNS が不要な最短経路。`docker-compose.yml` だけあれば動きます。`docker compose up -d --build` をリモートで叩くのと等価なので、TLS / Host ベースルーティングが必要ないケース (テスト、社内ツール、非 HTTP サービス、ホビー用途) に向きます。

```bash
# 初期化 (Docker / Compose のインストールのみ、proxy は不要)
conoha app init my-server --app-name myapp --no-proxy

# デプロイ (カレントディレクトリを tar して転送 → /opt/conoha/myapp/ に展開 → docker compose up -d --build)
conoha app deploy my-server --app-name myapp --no-proxy
```

以降の `status` / `logs` / `stop` / `restart` / `destroy` はサーバー上のマーカーから自動的に no-proxy モードで動作するため、フラグの再指定は不要です (明示してもエラーにはなりません):

```bash
conoha app status my-server --app-name myapp
conoha app logs my-server --app-name myapp --follow
conoha app destroy my-server --app-name myapp
```

no-proxy モードでは blue/green swap が存在しないため、`rollback` は利用できません (利用するとモード不一致エラー)。履歴から戻したい場合は該当コミットを checkout して `deploy` し直してください。

### モードの切り替え

既存のアプリのモードを変更するには、一度破棄してから反対のモードで再 init します:

```bash
conoha app destroy my-server --app-name myapp          # マーカーとディレクトリを削除
conoha app init my-server --app-name myapp --no-proxy  # 反対モードで再初期化
```

同一 VPS 上で `<app-name>` が異なれば proxy / no-proxy を並列に共存させられます。

### 主要フラグ

| フラグ | コマンド | 説明 |
|---|---|---|
| `--app-name <name>` | すべて | アプリ名 (非対話環境では必須) |
| `--proxy` / `--no-proxy` | `init` 以外の lifecycle 全て | マーカーを無視してモードを強制 (マーカーと不一致ならエラー) |
| `--slot <id>` | `deploy` | slot ID を固定 (proxy モードのみ意味あり) |
| `--drain-ms <ms>` | `rollback` | 戻し先の drain 窓を上書き (0 = proxy 既定) |
| `--follow` / `-f` | `logs` | リアルタイムストリーミング |
| `--service <name>` | `logs` | 特定サービスのログだけ出す |
| `--tail <n>` | `logs` | 末尾行数 (既定 100) |
| `--data-dir <path>` | proxy を叩くコマンド | サーバー側 proxy データディレクトリ (既定 `/var/lib/conoha-proxy`) |

### 環境変数の管理

デプロイを跨いで永続する環境変数はサーバー側で管理できます (両モード共通):

```bash
conoha app env set my-server --app-name myapp DATABASE_URL=postgres://...
conoha app env list my-server --app-name myapp
conoha app env get my-server --app-name myapp DATABASE_URL
conoha app env unset my-server --app-name myapp DATABASE_URL
```

デプロイ時の `.env` 合成は **リポジトリの `.env` → サーバー側の `/opt/conoha/<app>.env.server` を追記** の順で行われるため、`conoha app env set` で登録した値が後勝ちで上書きします。リポジトリ側にコミットした `.env` があればそれも `docker compose` に渡されます。
```

- [ ] **Step 3: Validate the replacement**

```bash
grep -n "## アプリデプロイ\|### proxy モード\|### no-proxy モード" /root/dev/crowdy/conoha-cli/README.md
grep -n "conoha app init\|conoha app deploy\|--no-proxy\|--proxy" /root/dev/crowdy/conoha-cli/README.md | head -20
```

Expected: the three new section headers appear exactly once; flag references are consistent with the live help output in `/tmp/conoha-app-help.txt`.

- [ ] **Step 4: Commit**

```bash
cd /root/dev/crowdy/conoha-cli
git add README.md
git commit -m "$(cat <<'EOF'
docs(readme): refresh deploy-mode coverage — document proxy + no-proxy in parallel

Expands the two-mode summary table with conoha.yml / proxy boot / DNS
columns so users can pick a mode at a glance. Adds a parallel "no-proxy
mode" subsection that previously only existed as a one-line footnote,
plus explicit mode-marker semantics (set by init, auto-detected, --proxy
/ --no-proxy as override-with-error-on-mismatch).

Also documents the full conoha.yml schema (compose_file, accessories,
health, deploy.drain_ms) — previously only name/hosts/web were shown.

Adds a flags reference table covering --slot, --drain-ms, --follow,
--service, --tail, --data-dir that were reachable only via --help.

No functional change; just documents features already shipped in
#98 (proxy blue/green) and #102/#103 (--no-proxy mode).
EOF
)"
```

Expected: commit succeeds, `git log --oneline -1` shows the new commit.

---

## Task 3: Mirror the rewrite into README-en.md

**Files:**
- Modify: `/root/dev/crowdy/conoha-cli/README-en.md` (replace the corresponding lines ~123-170)

- [ ] **Step 1: Apply the English translation of the new section**

Replace from `### Two deploy modes` through the closing fence of the rollback code block with:

```markdown
## App Deploy

`conoha app` supports two deploy modes that can coexist on the same VPS. `conoha app init` writes a marker on the server (`/opt/conoha/<name>/.conoha-mode`), and every subsequent `deploy` / `status` / `logs` / `stop` / `restart` / `destroy` / `rollback` auto-detects the mode from it. Pass `--proxy` or `--no-proxy` to override; a mismatch with the marker is an error (destroy + re-init to switch modes).

| Mode | Default | When to use | Layout | `conoha.yml` | `conoha proxy boot` | DNS / TLS |
|---|:-:|---|---|:-:|:-:|:-:|
| **proxy** (blue/green) | ✓ | Public app with a domain + Let's Encrypt TLS | `/opt/conoha/<name>/<slot>/` blue/green slots | required | required | required |
| **no-proxy** (flat) |  | Testing, internal / dev VPS, non-HTTP services, hobby apps | `/opt/conoha/<name>/` flat single dir | n/a | n/a | n/a |

### proxy mode (default): conoha-proxy blue/green

[conoha-proxy](https://github.com/crowdy/conoha-proxy) provides Let's Encrypt HTTPS, Host-header routing, and instant rollback inside the drain window.

1. Create `conoha.yml` at your repo root:

   ```yaml
   name: myapp                   # DNS-1123 label (lowercase alnum + hyphen, 1-63 chars)
   hosts:
     - app.example.com           # one or more FQDNs, no duplicates
   web:
     service: web                # must match a service in the compose file
     port: 8080                  # container-side listen port (1-65535)
   # --- optional ---
   compose_file: docker-compose.yml   # auto-detected (conoha-docker-compose.yml → docker-compose.yml → compose.yml)
   accessories: [db, redis]           # sibling services that join the same network
   health:
     path: /healthz
     interval_ms: 1000
     timeout_ms: 500
     healthy_threshold: 2
     unhealthy_threshold: 3
   deploy:
     drain_ms: 5000                   # drain window before tearing down the old slot (milliseconds)
   ```

2. Boot the proxy container on the VPS:

   ```bash
   conoha proxy boot my-server --acme-email ops@example.com
   ```

3. Point the DNS A record at the VPS (Let's Encrypt HTTP-01 validation needs it).

4. Register with the proxy and deploy:

   ```bash
   conoha app init my-server --app-name myapp
   conoha app deploy my-server --app-name myapp
   ```

5. Rollback (drain window only — instant swap back to the previous slot):

   ```bash
   conoha app rollback my-server --app-name myapp
   ```

`deploy --slot <id>` pins the slot ID (rule: `[a-z0-9][a-z0-9-]{0,63}`; default is git short SHA or timestamp). Reusing an existing slot ID purges its work dir before re-extracting.

### no-proxy mode: flat single-slot

Shortest path: no `conoha.yml`, no proxy, no DNS required. A `docker-compose.yml` is enough. This is equivalent to `docker compose up -d --build` over SSH and is the right choice when you do not need TLS or Host-based routing (testing, internal tools, non-HTTP services, hobby deployments).

```bash
# Initialize (installs Docker / Compose only; proxy is not required)
conoha app init my-server --app-name myapp --no-proxy

# Deploy (tar current dir → upload → extract to /opt/conoha/myapp/ → docker compose up -d --build)
conoha app deploy my-server --app-name myapp --no-proxy
```

Subsequent `status` / `logs` / `stop` / `restart` / `destroy` auto-detect no-proxy mode from the server marker, so you do not need to repeat `--no-proxy` (passing it again is allowed and is a no-op):

```bash
conoha app status my-server --app-name myapp
conoha app logs my-server --app-name myapp --follow
conoha app destroy my-server --app-name myapp
```

`rollback` is not available in no-proxy mode (there is no blue/green swap; invoking it raises a mode-mismatch error). Redeploy a previous commit instead: `git checkout <sha> && conoha app deploy ...`.

### Switching modes

Destroy, then re-init in the opposite mode:

```bash
conoha app destroy my-server --app-name myapp            # removes marker and work dir
conoha app init my-server --app-name myapp --no-proxy    # re-initialize in the other mode
```

Different `<app-name>`s on the same VPS can run in different modes side by side.

### Key flags

| Flag | Command | Description |
|---|---|---|
| `--app-name <name>` | all | App name (required in non-TTY environments) |
| `--proxy` / `--no-proxy` | every lifecycle cmd except `init` | Override the server marker (mismatch is an error) |
| `--slot <id>` | `deploy` | Pin the slot ID (proxy mode only) |
| `--drain-ms <ms>` | `rollback` | Override the rollback drain window (0 = proxy default) |
| `--follow` / `-f` | `logs` | Stream in real time |
| `--service <name>` | `logs` | Restrict to one service |
| `--tail <n>` | `logs` | Line count (default 100) |
| `--data-dir <path>` | proxy-facing cmds | Server-side proxy data dir (default `/var/lib/conoha-proxy`) |

### Environment variables

Server-side env vars persist across deploys (both modes):

```bash
conoha app env set my-server --app-name myapp DATABASE_URL=postgres://...
conoha app env list my-server --app-name myapp
conoha app env get my-server --app-name myapp DATABASE_URL
conoha app env unset my-server --app-name myapp DATABASE_URL
```

At deploy time, `.env` is assembled as **repo-committed `.env` first, then `/opt/conoha/<app>.env.server` (written by `conoha app env set`) appended**, so server-side values win via last-occurrence semantics. Any `.env` you committed in the repo is also picked up by `docker compose`.
```

- [ ] **Step 2: Verify structure alignment**

```bash
grep -n "^## \|^### " /root/dev/crowdy/conoha-cli/README.md | head -20
grep -n "^## \|^### " /root/dev/crowdy/conoha-cli/README-en.md | head -20
```

Expected: same header sequence (with translated titles) in the rewritten span.

- [ ] **Step 3: Commit**

```bash
git add README-en.md
git commit -m "docs(readme-en): mirror proxy + no-proxy refresh from README.md"
```

---

## Task 4: Mirror the rewrite into README-ko.md

**Files:**
- Modify: `/root/dev/crowdy/conoha-cli/README-ko.md` (replace corresponding lines)

- [ ] **Step 1: Apply the Korean translation**

Replace from `### 두 가지 배포 모드` (or its current equivalent header) through the end of the rollback section with:

```markdown
## 앱 배포

`conoha app` 은 같은 VPS 에서 공존할 수 있는 두 가지 배포 모드를 제공합니다. `conoha app init` 시점에 서버 측 마커 (`/opt/conoha/<name>/.conoha-mode`) 가 기록되고, 이후의 `deploy` / `status` / `logs` / `stop` / `restart` / `destroy` / `rollback` 은 자동으로 그 모드로 동작합니다. `--proxy` / `--no-proxy` 플래그는 마커를 덮어쓰되, 불일치 시 에러가 납니다 (모드 전환은 `destroy` → 재 `init`).

| 모드 | 기본 | 용도 | 레이아웃 | `conoha.yml` | `conoha proxy boot` | DNS / TLS |
|---|:-:|---|---|:-:|:-:|:-:|
| **proxy** (blue/green) | ✓ | 도메인 + Let's Encrypt TLS 공개 앱 | `/opt/conoha/<name>/<slot>/` blue/green 슬롯 | 필수 | 필수 | 필수 |
| **no-proxy** (flat) |  | 테스트, 사내/개발 VPS, 비 HTTP 서비스, 취미 앱 | `/opt/conoha/<name>/` 평면 단일 디렉터리 | 불필요 | 불필요 | 불필요 |

### proxy 모드 (기본): conoha-proxy 기반 blue/green

[conoha-proxy](https://github.com/crowdy/conoha-proxy) 가 Let's Encrypt HTTPS, Host 헤더 라우팅, drain 윈도우 내 즉시 롤백을 제공합니다.

1. 리포지토리 루트에 `conoha.yml` 작성:

   ```yaml
   name: myapp                   # DNS-1123 라벨 (소문자 영숫자 + 하이픈, 1-63 자)
   hosts:
     - app.example.com           # 하나 이상, 중복 불가
   web:
     service: web                # compose 파일의 서비스명과 일치해야 함
     port: 8080                  # 컨테이너 listen 포트 (1-65535)
   # --- 선택 ---
   compose_file: docker-compose.yml   # 생략 시 conoha-docker-compose.yml → docker-compose.yml → compose.yml 순으로 자동 검출
   accessories: [db, redis]           # web 과 같은 네트워크에 붙는 부속 서비스
   health:
     path: /healthz
     interval_ms: 1000
     timeout_ms: 500
     healthy_threshold: 2
     unhealthy_threshold: 3
   deploy:
     drain_ms: 5000                   # 구 슬롯을 내릴 때까지의 drain 윈도우 (ms)
   ```

2. VPS 에 프록시 컨테이너 부팅:

   ```bash
   conoha proxy boot my-server --acme-email ops@example.com
   ```

3. DNS A 레코드를 VPS 로 향하게 하기 (Let's Encrypt HTTP-01 검증에 필요).

4. 프록시에 앱을 등록하고 배포:

   ```bash
   conoha app init my-server --app-name myapp
   conoha app deploy my-server --app-name myapp
   ```

5. 롤백 (drain 윈도우 내에서만, 이전 슬롯으로 즉시 전환):

   ```bash
   conoha app rollback my-server --app-name myapp
   ```

`deploy --slot <id>` 로 슬롯 ID 를 고정할 수 있습니다 (규칙: `[a-z0-9][a-z0-9-]{0,63}`, 기본값은 git short SHA 또는 timestamp). 기존 슬롯명을 재사용하면 작업 디렉터리를 정리한 뒤 재전개합니다.

### no-proxy 모드: 평면 단일 슬롯

`conoha.yml` / proxy / DNS 없이도 가능한 최단 경로. `docker-compose.yml` 만 있으면 됩니다. SSH 로 `docker compose up -d --build` 를 실행하는 것과 동등하며, TLS / Host 기반 라우팅이 필요 없는 용도 (테스트, 사내 도구, 비 HTTP 서비스, 취미 배포) 에 적합합니다.

```bash
# 초기화 (Docker / Compose 설치만 수행, proxy 없음)
conoha app init my-server --app-name myapp --no-proxy

# 배포 (현재 디렉터리 tar → 업로드 → /opt/conoha/myapp/ 에 전개 → docker compose up -d --build)
conoha app deploy my-server --app-name myapp --no-proxy
```

이후의 `status` / `logs` / `stop` / `restart` / `destroy` 는 서버 마커에서 자동 판별되므로 `--no-proxy` 를 반복할 필요가 없습니다 (다시 넘겨도 에러는 아니며 no-op):

```bash
conoha app status my-server --app-name myapp
conoha app logs my-server --app-name myapp --follow
conoha app destroy my-server --app-name myapp
```

no-proxy 모드에는 blue/green 스왑이 없으므로 `rollback` 은 사용할 수 없습니다 (호출 시 모드 불일치 에러). 이전 커밋으로 되돌리려면 `git checkout <sha> && conoha app deploy ...` 로 재배포하세요.

### 모드 전환

기존 앱의 모드를 바꾸려면 한 번 제거한 뒤 반대 모드로 재 init 합니다:

```bash
conoha app destroy my-server --app-name myapp            # 마커와 작업 디렉터리 제거
conoha app init my-server --app-name myapp --no-proxy    # 반대 모드로 재초기화
```

같은 VPS 위에서도 `<app-name>` 이 다르면 proxy / no-proxy 를 나란히 공존시킬 수 있습니다.

### 주요 플래그

| 플래그 | 명령 | 설명 |
|---|---|---|
| `--app-name <name>` | 전체 | 앱 이름 (비 TTY 환경에서는 필수) |
| `--proxy` / `--no-proxy` | `init` 이외 lifecycle 전체 | 마커를 덮어쓰고 모드 강제 (불일치 시 에러) |
| `--slot <id>` | `deploy` | 슬롯 ID 고정 (proxy 모드에서만 의미) |
| `--drain-ms <ms>` | `rollback` | 롤백 drain 윈도우 오버라이드 (0 = proxy 기본값) |
| `--follow` / `-f` | `logs` | 실시간 스트리밍 |
| `--service <name>` | `logs` | 특정 서비스만 |
| `--tail <n>` | `logs` | 출력 줄 수 (기본 100) |
| `--data-dir <path>` | proxy 를 호출하는 명령 | 서버 측 proxy 데이터 디렉터리 (기본 `/var/lib/conoha-proxy`) |

### 환경 변수 관리

배포를 가로질러 유지되는 환경 변수는 서버 측에서 관리합니다 (두 모드 공통):

```bash
conoha app env set my-server --app-name myapp DATABASE_URL=postgres://...
conoha app env list my-server --app-name myapp
conoha app env get my-server --app-name myapp DATABASE_URL
conoha app env unset my-server --app-name myapp DATABASE_URL
```

배포 시 `.env` 는 **리포지토리에 커밋된 `.env` → 서버 측 `/opt/conoha/<app>.env.server` (즉 `conoha app env set` 값) 순으로 이어붙여** 조립됩니다. 따라서 서버 측 값이 뒤에 오는 원칙에 따라 우선합니다. 리포지토리에 커밋한 `.env` 도 `docker compose` 에 그대로 전달됩니다.
```

- [ ] **Step 2: Commit**

```bash
git add README-ko.md
git commit -m "docs(readme-ko): mirror proxy + no-proxy refresh from README.md"
```

---

## Task 5: Push conoha-cli docs branch + open PR

**Files:** none (git + gh only)

- [ ] **Step 1: Push and open PR**

```bash
cd /root/dev/crowdy/conoha-cli
git push -u origin docs/readme-noproxy-refresh
gh pr create --title "docs(readme): document proxy + no-proxy deploy modes in parallel" --body "$(cat <<'EOF'
## Summary

Refreshes the three READMEs (JA/EN/KO) to document `conoha app`'s two deploy modes symmetrically:

- The pre-existing **proxy** (blue/green via conoha-proxy) flow, now with explicit mode-marker semantics and the full `conoha.yml` schema (compose_file / accessories / health / deploy.drain_ms were previously undocumented).
- The **no-proxy** flat flow shipped in #102/#103, which was previously only a one-line footnote.
- A flags reference table covering `--slot`, `--drain-ms`, `--follow`, `--service`, `--tail`, `--data-dir`.
- `conoha app env` (get/list/set/unset) — previously missing from the README.

No code changes. All claims traced back to `conoha app <sub> --help` output and `internal/config/projectfile.go`.

## Test plan

- [x] Eyeball `conoha app init/deploy/logs/status/stop/restart/destroy/rollback --help` against the flag tables.
- [x] Cross-check `conoha.yml` field list against `internal/config/projectfile.go:15-23`.
- [x] JA/EN/KO header structure matches.
EOF
)"
```

Expected: PR URL returned.

---

## Task 6: Update SKILL.md in conoha-cli-skill repo

**Files:**
- Modify: `/tmp/conoha-cli-skill/SKILL.md`

- [ ] **Step 1: Replace the "アプリデプロイ" subsection (currently lines 80-89)**

Replace the `### アプリデプロイ` table and its heading with:

```markdown
### アプリデプロイ

`conoha app` は 2 つのデプロイモードを提供する：

- **proxy モード (既定)** — conoha-proxy 経由の blue/green デプロイ。ドメイン + Let's Encrypt TLS、`conoha.yml` 必須、事前に `conoha proxy boot` が必要。
- **no-proxy モード (`--no-proxy`)** — フラット単一スロット。`conoha.yml` / proxy / DNS 不要。テスト・内部 VPS・非 HTTP サービス・ホビー用途に適する。

`conoha app init` がサーバーに `.conoha-mode` マーカーを書き込み、以降の lifecycle コマンドは自動的に同じモードで動作する。`--proxy` / `--no-proxy` フラグはマーカーを上書きするが、不一致ならエラーになる。

| コマンド | 説明 |
|---------|------|
| `conoha app init <server> --app-name <app>` | proxy モードで初期化 (conoha.yml と `conoha proxy boot` 済み前提) |
| `conoha app init <server> --app-name <app> --no-proxy` | no-proxy モードで初期化 (Docker / Compose のみインストール) |
| `conoha app deploy <server> --app-name <app>` | カレントディレクトリをデプロイ (モードはマーカーから自動判別) |
| `conoha app deploy <server> --app-name <app> --slot <id>` | slot ID を固定 (proxy モード) |
| `conoha app rollback <server> --app-name <app>` | 前 slot へ即時ロールバック (proxy モードのみ、drain 窓内) |
| `conoha app status <server> --app-name <app>` | コンテナ状態を表示 |
| `conoha app logs <server> --app-name <app> --follow` | ログをストリーミング |
| `conoha app logs <server> --app-name <app> --service <svc>` | 特定サービスのログ |
| `conoha app stop <server> --app-name <app>` | コンテナを停止 |
| `conoha app restart <server> --app-name <app>` | コンテナを再起動 |
| `conoha app destroy <server> --app-name <app> --yes` | アプリをサーバーから完全削除 (非対話) |
| `conoha app list <server>` | サーバー上のデプロイ済みアプリ一覧 |
| `conoha app env set <server> --app-name <app> KEY=VALUE` | サーバー側永続環境変数を設定 |
| `conoha app env list/get/unset` | 環境変数の一覧・取得・削除 |

モード切り替えは `destroy` → 反対モードで `init`。同一 VPS で `<app-name>` が異なれば 2 モードを並列で共存可。

選択指針:
- ドメイン + HTTPS が必要 / 本番公開 → **proxy**
- `docker compose up -d --build` 相当で十分 / DNS 未取得 / 社内・検証・ホビー → **no-proxy**
```

- [ ] **Step 2: Fix the non-TTY guidance to cover `rollback`/`destroy` explicitly (currently line 30)**

Replace the line:
```
5. **破壊的コマンド（server delete、app destroy、app stop等）は `--yes` フラグで確認をスキップする**（環境変数: `CONOHA_YES=1`）
```

with:

```
5. **破壊的コマンド（`server delete`、`app destroy`、`app stop`、`app restart`、`app rollback` 等）は `--yes` フラグで確認をスキップする**（環境変数: `CONOHA_YES=1`）
6. **アプリの既存モードと異なる `--proxy` / `--no-proxy` を指定するとモード不一致エラーで停止する** — 切り替えたい場合は `conoha app destroy --yes` → 反対モードで `init` し直す
```

- [ ] **Step 3: Commit**

```bash
cd /tmp/conoha-cli-skill
git add SKILL.md
git commit -m "$(cat <<'EOF'
docs(skill): refresh app commands for proxy + no-proxy modes

- Adds `--no-proxy` flag coverage throughout the app commands table.
- Documents mode-marker auto-detection and --proxy/--no-proxy override
  semantics (see crowdy/conoha-cli#102, #103).
- Adds previously missing subcommands: `destroy`, `rollback`, `list`,
  and the full `env` family (get/list/set/unset).
- Fixes stale `init` description: `init` no longer installs Docker in
  proxy mode (that's `conoha proxy boot` now); it installs Docker only
  in no-proxy mode.
- Expands the non-TTY guidance to call out `app restart` / `app rollback`
  as also needing `--yes`, and notes the mode-mismatch error class.
EOF
)"
```

---

## Task 7: Rewrite recipes/single-server-app.md for both modes

**Files:**
- Rewrite: `/tmp/conoha-cli-skill/recipes/single-server-app.md`

- [ ] **Step 1: Overwrite the recipe with two parallel flows**

Replace the entire file with:

```markdown
# シングルサーバーアプリデプロイ

## 概要

Docker Compose アプリをサーバー 1 台にデプロイするレシピ。`conoha app` コマンドでカレントディレクトリの Compose プロジェクトを転送・起動する。

`conoha app` は 2 つのモードを提供するので、用途に応じて選ぶ:

| モード | いつ選ぶか | 事前準備 |
|---|---|---|
| **proxy (blue/green, 既定)** | ドメイン + HTTPS が必要、本番公開、無停止デプロイ | `conoha.yml`、`conoha proxy boot`、DNS A レコード |
| **no-proxy (flat)** | テスト、社内ツール、非 HTTP サービス、ホビー用途 | `docker-compose.yml` だけ |

## 共通の前提

- `conoha auth login` で認証済み
- キーペアが登録済み (`conoha keypair create <name>`)
- カレントディレクトリに `docker-compose.yml` (または `compose.yml`) がある
- 非 TTY 環境では `--no-input` / `--yes` / 必須フラグを明示 (SKILL.md 冒頭の注意を参照)

### サーバー作成 (両モード共通)

```bash
conoha flavor list
conoha image list
conoha keypair create my-key    # 既存ならスキップ

conoha server create \
  --name my-app-server \
  --flavor <フレーバーID> \
  --image <UbuntuイメージID> \
  --key-name my-key \
  --wait
```

`--wait` でサーバーが ACTIVE になるまで待つ。

---

## モード A: proxy blue/green (推奨)

### 1. `conoha.yml` をレポジトリルートに作成

```yaml
name: myapp
hosts:
  - app.example.com
web:
  service: web    # docker-compose.yml のサービス名と一致
  port: 8080      # コンテナ側 listen ポート
# 任意:
# compose_file: docker-compose.yml
# accessories: [db, redis]
# health: { path: /healthz, interval_ms: 1000, timeout_ms: 500, healthy_threshold: 2, unhealthy_threshold: 3 }
# deploy: { drain_ms: 5000 }
```

スキーマ詳細は conoha-cli README を参照。

### 2. DNS A レコードを VPS に向ける

Let's Encrypt HTTP-01 検証に必要。ACME 発行前にポート 80 が到達できる状態にしておく。

### 3. conoha-proxy をブート

```bash
conoha proxy boot my-app-server --acme-email ops@example.com
```

### 4. アプリを proxy に登録

```bash
conoha app init my-app-server --app-name myapp
```

これで `.conoha-mode=proxy` のマーカーが書き込まれ、proxy に service が登録される。

### 5. デプロイ

カレントディレクトリで実行:

```bash
conoha app deploy my-app-server --app-name myapp
```

実行内容:
- カレントディレクトリを tar.gz アーカイブ化 (`.git/` 除外)
- SSH で転送
- `/opt/conoha/myapp/<slot>/` に展開 (slot は git short SHA または timestamp、`--slot <id>` で上書き可)
- 新 slot を動的ポートで起動
- proxy が health probe → 新 slot に swap
- drain 窓経過後に旧 slot をテアダウン

### 6. 動作確認 / 管理

```bash
conoha app status my-app-server --app-name myapp
conoha app logs my-app-server --app-name myapp --follow
```

ロールバック (drain 窓内のみ):

```bash
conoha app rollback my-app-server --app-name myapp
```

廃棄:

```bash
conoha app destroy my-app-server --app-name myapp --yes
```

---

## モード B: no-proxy (flat)

proxy / DNS / TLS が不要なとき。`docker-compose.yml` があれば動く最短経路。

### 1. 初期化

```bash
conoha app init my-app-server --app-name myapp --no-proxy
```

これで `.conoha-mode=no-proxy` のマーカーが書き込まれる。Docker / Compose がインストールされるが、proxy コンテナは起動しない。

### 2. デプロイ

```bash
conoha app deploy my-app-server --app-name myapp --no-proxy
```

実行内容:
- カレントディレクトリを tar.gz アーカイブ化 (`.git/` 除外)
- `/opt/conoha/myapp/` に展開 (フラット、slot なし)
- サーバー側 `/opt/conoha/<app>.env.server` (`conoha app env set` で登録) があれば、リポジトリの `.env` に追記する形で合成
- `docker compose up -d --build --remove-orphans`

### 3. 動作確認 / 管理

以降のコマンドはマーカーから自動判別されるので `--no-proxy` の再指定は不要 (付けても動く):

```bash
conoha app status my-app-server --app-name myapp
conoha app logs my-app-server --app-name myapp --follow
conoha app logs my-app-server --app-name myapp --service web
conoha app stop my-app-server --app-name myapp --yes
conoha app restart my-app-server --app-name myapp --yes
conoha app destroy my-app-server --app-name myapp --yes
```

ポート公開は `docker-compose.yml` の `ports` セクションで直接行う。セキュリティグループで該当ポートを開放すること。

no-proxy モードには blue/green swap が無いため `rollback` は使えない。前のリビジョンに戻すには `git checkout <sha> && conoha app deploy ...` で再デプロイする。

---

## 環境変数

サーバー側に永続する環境変数はデプロイを跨いで維持される (両モード共通):

```bash
conoha app env set my-app-server --app-name myapp DATABASE_URL=postgres://...
conoha app env list my-app-server --app-name myapp
conoha app env get my-app-server --app-name myapp DATABASE_URL
conoha app env unset my-app-server --app-name myapp DATABASE_URL
```

デプロイ時の `.env` 合成は **リポジトリの `.env` → サーバー側の `/opt/conoha/<app>.env.server` を追記** の順で行われるため、`conoha app env set` で登録した値が後勝ちで上書きする。リポジトリ側にコミットした `.env` があればそれも `docker compose` に渡される。

## モード切り替え

既存アプリのモードを変えるときは、一度破棄してから反対モードで再 init する:

```bash
conoha app destroy my-app-server --app-name myapp --yes
conoha app init my-app-server --app-name myapp --no-proxy   # または --proxy (フラグ省略)
```

同一 VPS 上で `<app-name>` が異なれば、proxy / no-proxy を並列に共存可能。

## トラブルシューティング

| 問題 | 対処 |
|------|------|
| `mode conflict` エラー | `--proxy` / `--no-proxy` フラグが既存マーカーと不一致。上記「モード切り替え」を参照 |
| `docker compose up` が失敗 | `conoha app logs` でエラー確認 |
| ポートにアクセス不可 (no-proxy) | セキュリティグループで `docker-compose.yml` の `ports` が開放されているか確認 |
| Let's Encrypt 発行失敗 (proxy) | DNS A レコードが VPS を指しているか、ポート 80 が到達可能かを確認 |
| デプロイが遅い | `.dockerignore` で `node_modules` 等を除外 |
```

- [ ] **Step 2: Commit**

```bash
cd /tmp/conoha-cli-skill
git add recipes/single-server-app.md
git commit -m "$(cat <<'EOF'
docs(recipe/single-server-app): refresh for proxy + no-proxy modes

The previous recipe described the pre-proxy "tar + SSH + docker compose up"
flow exclusively and pre-dated both the conoha-proxy blue/green refactor
(#98) and the --no-proxy mode (#102/#103).

Restructures as:
- Shared prerequisites + server creation (same for both modes)
- Mode A: proxy blue/green — conoha.yml, proxy boot, DNS, blue/green slots, rollback
- Mode B: no-proxy flat — Docker only, docker compose up, no rollback
- Shared footer: env vars, mode switching, troubleshooting

Adds explicit guidance on:
- Choosing between modes (domain/TLS vs testing/internal)
- Mode marker auto-detection and mismatch errors
- Why rollback doesn't exist in no-proxy mode (no blue/green swap)
EOF
)"
```

---

## Task 8: Push skill branch + open PR

**Files:** none (git + gh only)

- [ ] **Step 1: Push and open PR**

```bash
cd /tmp/conoha-cli-skill
git push -u origin docs/noproxy-refresh
gh pr create --repo crowdy/conoha-cli-skill --title "docs: refresh SKILL.md and single-server-app recipe for proxy + no-proxy modes" --body "$(cat <<'EOF'
## Summary

Brings the skill docs up to date with two features shipped in crowdy/conoha-cli since the last skill update (2026-04-02):

1. **conoha-proxy blue/green deploy** (crowdy/conoha-cli#98) — default mode; `conoha.yml`, blue/green slots, `rollback` within the drain window.
2. **`--no-proxy` flat mode** (crowdy/conoha-cli#102, #103) — `docker compose up` equivalent over SSH; no proxy / DNS / TLS needed.

### SKILL.md

- App commands table now covers both modes, every `--proxy` / `--no-proxy` flag, and previously missing subcommands (`destroy`, `rollback`, `list`, `env {get,list,set,unset}`).
- Explains mode marker auto-detection and mismatch errors.
- Fixes the stale "Docker 環境を初期化" description of `init` — Docker installation is now `conoha proxy boot`'s job in proxy mode; `init` only installs Docker in no-proxy mode.
- Extends non-TTY guidance to cover `app restart` / `app rollback` needing `--yes`, and the mode-mismatch error class.

### recipes/single-server-app.md

Full rewrite. Previously documented only the pre-proxy flat flow. Now presents two parallel flows (proxy blue/green vs no-proxy flat) with a shared prerequisites section, plus env vars, mode switching, and a troubleshooting table.

## Test plan

- [x] Claims traced back to `conoha app <sub> --help`.
- [x] Mode marker behavior cross-checked against `cmd/app/mode.go` in the CLI repo.
- [x] `conoha.yml` schema matches `internal/config/projectfile.go`.
EOF
)"
```

Expected: PR URL returned.

- [ ] **Step 2: Also update the crowdy-cc backup copy for local dev parity**

If the operator uses the crowdy-cc backup copy (`/root/dev/crowdy-cc/skills/conoha-cli-skill/`), sync after the upstream PR merges:

```bash
cd /root/dev/crowdy-cc/skills/conoha-cli-skill
git pull --ff-only origin main   # assumes submodule or mirror; otherwise copy files manually
```

(Non-blocking; only applies if crowdy-cc is configured to mirror. Skip if unsure.)

---

## Self-Review

### Spec coverage

Every item in the user's request covers:

- **Proxy vs no-proxy visibility in README** → Task 2 (JA), 3 (EN), 4 (KO): new parallel sections with comparison table.
- **Flag freshness** → Task 2 flag-reference table; Task 6 SKILL app commands table with `--no-proxy`, `--slot`, `--drain-ms`, `--follow`, `--service`, `--tail`, `--data-dir`, `--yes`.
- **Skill staleness** → Task 6 (SKILL.md) + Task 7 (recipe rewrite).
- **Missing subcommands** → Task 6 adds `destroy`, `rollback`, `list`, `env *`.

### Placeholder scan

- No "TBD / implement later" text.
- Every code/yaml block is complete.
- No "similar to Task N" — each task's Markdown blocks are self-contained.

### Type / terminology consistency

- "proxy" (blue/green) / "no-proxy" (flat) naming used uniformly in JA/EN/KO.
- `.conoha-mode` marker path is consistent across tasks (matches `cmd/app/mode.go:48`).
- `conoha.yml` field names match `internal/config/projectfile.go` exactly (`compose_file`, `accessories`, `health`, `deploy.drain_ms`).
- Slot regex `[a-z0-9][a-z0-9-]{0,63}` used consistently (matches `cmd/app/deploy.go` help text).

### Known deviations from skill guidance

- Plan was not authored inside a dedicated worktree (docs-only, low blast radius).
- No tests in the TDD sense — docs changes are verified by eyeballing `--help` output and schema source. This is noted in PR test plans.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-21-readme-skill-noproxy-refresh.md`. Two execution options:

**1. Subagent-Driven (recommended)** — fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — execute tasks in this session with checkpoints after Tasks 4 (CLI docs PR ready) and 7 (skill docs PR ready).

Which approach?
