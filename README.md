# conoha - ConoHa VPS3 CLI

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

[English](README-en.md) | [한국어](README-ko.md)

ConoHa VPS3 API 用のコマンドラインインターフェースです。Go で書かれたシングルバイナリで、エージェントフレンドリーな設計を採用しています。

**[ドキュメントサイト](https://crowdy.github.io/conoha-cli-pages/)** — ガイド・実践デプロイ例・コマンドリファレンス

> **注意**: 本ツールは VPS3 専用です。旧 VPS2 用の CLI（hironobu-s/conoha-vps、miyabisun/conoha-cli）とは互換性がありません。

## 特徴

- シングルバイナリ、クロスプラットフォーム対応（Linux / macOS / Windows）
- 複数プロファイル対応（`gh auth` スタイル）
- 構造化出力（`--format json/yaml/csv/table`）
- エージェントフレンドリー設計（`--no-input`、決定的な終了コード、stderr/stdout 分離）
- トークン自動更新（有効期限 5 分前に再認証）
- Claude Code スキル連携（`conoha skill install` でインフラ構築レシピを導入）

## インストール

### Homebrew (macOS / Linux)

```bash
brew install crowdy/tap/conoha
```

### Scoop (Windows)

```powershell
scoop bucket add crowdy https://github.com/crowdy/crowdy-bucket
scoop install conoha
```

### ソースからビルド

```bash
go install github.com/crowdy/conoha-cli@latest
```

### リリースバイナリ

[Releases](https://github.com/crowdy/conoha-cli/releases) ページからダウンロード、または以下のコマンドを使用してください：

**Linux (amd64)**

```bash
VERSION=$(curl -s https://api.github.com/repos/crowdy/conoha-cli/releases/latest | grep tag_name | cut -d '"' -f4)
curl -Lo conoha.tar.gz "https://github.com/crowdy/conoha-cli/releases/download/${VERSION}/conoha-cli_${VERSION#v}_linux_amd64.tar.gz"
tar xzf conoha.tar.gz conoha
sudo mv conoha /usr/local/bin/
rm conoha.tar.gz
```

**macOS (Apple Silicon)**

```bash
VERSION=$(curl -s https://api.github.com/repos/crowdy/conoha-cli/releases/latest | grep tag_name | cut -d '"' -f4)
curl -Lo conoha.tar.gz "https://github.com/crowdy/conoha-cli/releases/download/${VERSION}/conoha-cli_${VERSION#v}_darwin_arm64.tar.gz"
tar xzf conoha.tar.gz conoha
sudo mv conoha /usr/local/bin/
rm conoha.tar.gz
```

**Windows (amd64)**

```powershell
$version = (Invoke-RestMethod https://api.github.com/repos/crowdy/conoha-cli/releases/latest).tag_name
$v = $version -replace '^v', ''
Invoke-WebRequest -Uri "https://github.com/crowdy/conoha-cli/releases/download/$version/conoha-cli_${v}_windows_amd64.zip" -OutFile conoha.zip
Expand-Archive conoha.zip -DestinationPath .
Remove-Item conoha.zip
```

> **Tip**: [Scoop](https://scoop.sh/) を導入済みであれば、`%PATH%` に別途登録するより以下のコマンドで `shims` に配置するのが簡単です：
>
> ```cmd
> move conoha.exe %USERPROFILE%\scoop\shims\
> ```

## クイックスタート

```bash
# ログイン（テナント ID、ユーザー名、パスワードを入力）
conoha auth login

# 認証状態を確認
conoha auth status

# サーバー一覧を表示
conoha server list

# JSON 形式で出力
conoha server list --format json

# サーバーの詳細を表示（ID またはサーバー名で指定可能）
conoha server show <server-id-or-name>

# サーバー名の変更
conoha server rename <server-id-or-name> new-name
```

## コマンド一覧

| コマンド | 説明 |
|---------|------|
| `conoha auth` | 認証管理（login / logout / status / list / switch / token / remove） |
| `conoha server` | サーバー管理（list / show / create / delete / start / stop / reboot / resize / rebuild / rename / console / ips / metadata / ssh / deploy / attach-volume / detach-volume） |
| `conoha flavor` | フレーバー一覧・詳細（list / show） |
| `conoha keypair` | SSH キーペア管理（list / show / create / delete） |
| `conoha volume` | ブロックストレージ管理（list / show / create / delete / types / backup） |
| `conoha image` | イメージ管理（list / show / create / upload / import / delete） |
| `conoha network` | ネットワーク管理（network / subnet / port / security-group / qos） |
| `conoha lb` | ロードバランサー管理（lb / listener / pool / member / healthmonitor） |
| `conoha dns` | DNS 管理（domain / record） |
| `conoha storage` | オブジェクトストレージ（container / ls / cp / rm / publish） |
| `conoha identity` | アイデンティティ管理（credential / subuser / role） |
| `conoha app` | アプリデプロイ・管理（init / deploy / rollback / logs / status / stop / restart / env / destroy / list） |
| `conoha proxy` | conoha-proxy リバースプロキシ管理（boot / reboot / start / stop / restart / remove / logs / details / services） |
| `conoha config` | CLI 設定管理（show / set / path） |
| `conoha skill` | Claude Code スキル管理（install / update / remove） |

## サーバー作成

対話的にサーバーを作成できます（フレーバー、イメージ、キーペアを選択）：

```bash
conoha server create --name my-server
```

フラグで直接指定も可能：

```bash
conoha server create --name my-server \
  --flavor g2l-t-c2m1 \
  --image <image-id> \
  --key-name my-key \
  --admin-pass 'P@ssw0rd'
```

### スタートアップスクリプト

サーバー作成時に初期設定スクリプトを指定できます：

```bash
# ファイルから
conoha server create --name my-server --user-data ./init.sh

# インライン
conoha server create --name my-server --user-data-raw '#!/bin/bash
apt update && apt install -y nginx'

# URL 指定
conoha server create --name my-server --user-data-url https://example.com/setup.sh
```

詳細は [docs/startup-script.md](docs/startup-script.md) を参照してください。

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
     drain_ms: 5000                   # 旧スロットを落とすまでの drain 窓 (ミリ秒、未指定時は 30000)
   ```

2. プロキシコンテナを VPS にブート：

   ```bash
   conoha proxy boot my-server --acme-email ops@example.com
   ```

   このステップを飛ばして `app init` に進むと、Admin API ソケットに到達できずエラーで停止します。

3. DNS の A レコードを VPS に向ける（Let's Encrypt HTTP-01 検証に必要）。DNS は `app init` がホストを登録する時点で名前解決できる必要があります — 未設定のまま進めても `app` レイヤのデプロイ自体は成功しますが、ACME が発行に成功するまでは該当ホスト名への HTTPS が TLS ハンドシェイク段階で失敗します（発行できた証明書が無く、SNI に対してサーバが証明書を返せない状態。ブラウザ警告ページではなく接続リセット／ハンドシェイク失敗として見えます）。

4. アプリを proxy に登録してデプロイ：

   ```bash
   conoha app init my-server
   conoha app deploy my-server
   ```

5. ロールバック（drain 窓内のみ、旧スロットへ即時戻し）：

   ```bash
   conoha app rollback my-server
   ```

`deploy --slot <id>` で slot ID を固定できます (規則: `[a-z0-9][a-z0-9-]{0,63}`、既定は git short SHA または timestamp)。同名 slot を明示的に再利用すると作業ディレクトリを削除してから再展開します。`--slot` を省略した場合、既定値が既存の compose プロジェクトと衝突したら CLI が自動で `-2` / `-3` と suffix を付けて衝突を回避するので、drain 中のスロットを破壊的に上書きすることはありません。

### no-proxy モード: フラット単一スロット

`conoha.yml` / proxy / DNS が不要な最短経路。`docker-compose.yml` だけあれば動きます。`docker compose up -d --build` をリモートで叩くのと等価なので、TLS / Host ベースルーティングが必要ないケース (テスト、社内ツール、非 HTTP サービス、ホビー用途) に向きます。

```bash
# 初期化 (Docker / Compose が入っていることを検証してマーカーを書き込む。インストールは行わないので、Docker 未導入の VPS では事前に `conoha server create --user-data ./install-docker.sh` 等で入れておく必要がある)
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

no-proxy モードには blue/green swap が存在しないため、`rollback` は利用できません (実行すると "rollback is not supported in no-proxy mode" エラーが出ます)。履歴から戻したい場合は該当コミットを checkout して `deploy` し直してください。

再デプロイ時は tar 展開が **上書きのみ**行い、リポジトリから消したファイルはサーバー上に残り続けます (`.env.server` や名前付きボリュームの bind mount を守るため意図的にそうしています)。古いファイルを掃除する場合は `ssh <server> rm /opt/conoha/<name>/<path>` で個別に削除してください。将来 `--clean` フラグを追加する可能性があります ([#108](https://github.com/crowdy/conoha-cli/issues/108))。

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
| `--app-name <name>` | `destroy` / `status` / `logs` / `stop` / `restart` / `env` では常に、`init` / `deploy` / `rollback` では `--no-proxy` と併用する場合に必須 | アプリ名。未指定かつ TTY 有りなら対話プロンプト、非 TTY 環境では指定必須 |
| `--proxy` / `--no-proxy` | lifecycle コマンド (`list` 以外) | `init` ではマーカーに書き込むモードを選択、それ以外ではマーカーを上書き (不一致ならエラー) |
| `--slot <id>` | `deploy` | slot ID を固定 (proxy モードのみ意味あり) |
| `--drain-ms <ms>` | `rollback` | 戻し先の drain 窓を上書き (0 = proxy 既定) |
| `--follow` / `-f` | `logs` | リアルタイムストリーミング |
| `--service <name>` | `logs` | 特定サービスのログだけ出す |
| `--tail <n>` | `logs` | 末尾行数 (既定 100) |
| `--data-dir <path>` | proxy を叩くコマンド | サーバー側 proxy データディレクトリ (既定 `/var/lib/conoha-proxy`) |

### 環境変数の管理 (no-proxy モード)

デプロイを跨いで永続する環境変数はサーバー側で管理できます。`conoha app env set` は両モードで動作してサーバー側の `/opt/conoha/<app>.env.server` に書き込みますが、**現状 `app env` による値の反映はデプロイ時 `.env` 合成を行う no-proxy モードでのみ有効です** — proxy モードで `app env set` すると `warning: app env has no effect on proxy-mode deployed slots; see #94 for the redesign` が出ます ([#94](https://github.com/crowdy/conoha-cli/issues/94) で再設計予定)。proxy モードでは当面 compose ファイルの `environment:` / `env_file:` でアプリ設定を渡してください。

```bash
conoha app env set my-server --app-name myapp DATABASE_URL=postgres://...
conoha app env list my-server --app-name myapp
conoha app env get my-server --app-name myapp DATABASE_URL
conoha app env unset my-server --app-name myapp DATABASE_URL
```

no-proxy デプロイ時の `.env` 合成は **リポジトリの `.env` → サーバー側の `/opt/conoha/<app>.env.server` を追記** の順で行われるため、`conoha app env set` で登録した値が後勝ちで上書きします (proxy モードではこの合成は行われません)。

## Claude Code スキル

ConoHa CLI には Claude Code 用のインフラ構築スキルが用意されています。インストールすると、Claude Code から自然言語でインフラ構築を指示できます。

### インストール

```bash
# スキルをインストール
conoha skill install

# スキルを更新
conoha skill update

# スキルを削除
conoha skill remove
```

### 使い方

Claude Code で以下のように指示するだけで、スキルが自動的にトリガーされます：

```
> ConoHa でサーバーを作って
> k8s クラスターを構築して
> アプリをデプロイして
```

### レシピ一覧

| レシピ | 説明 |
|-------|------|
| Docker Compose アプリデプロイ | `conoha app deploy` によるコンテナアプリのデプロイ |
| カスタムスクリプトデプロイ | スタートアップスクリプトによるサーバー構成 |
| Kubernetes クラスター | k3s によるクラスター構築（coming soon） |
| OpenStack プラットフォーム | DevStack によるプラットフォーム構築（coming soon） |
| Slurm HPC クラスター | Slurm による HPC クラスター構築（coming soon） |

詳細は [conoha-cli-skill](https://github.com/crowdy/conoha-cli-skill) を参照してください。

## 設定

設定ファイルは `~/.config/conoha/` に保存されます：

| ファイル | 説明 | パーミッション |
|---------|------|------------|
| `config.yaml` | プロファイル設定 | 0600 |
| `credentials.yaml` | パスワード | 0600 |
| `tokens.yaml` | トークンキャッシュ | 0600 |

### 環境変数

| 変数 | 説明 |
|-----|------|
| `CONOHA_PROFILE` | 使用するプロファイル名 |
| `CONOHA_TENANT_ID` | テナント ID |
| `CONOHA_USERNAME` | API ユーザー名 |
| `CONOHA_PASSWORD` | API パスワード |
| `CONOHA_TOKEN` | 認証トークン（直接指定） |
| `CONOHA_FORMAT` | 出力形式 |
| `CONOHA_CONFIG_DIR` | 設定ディレクトリ |
| `CONOHA_NO_INPUT` | 非対話モード（`1` or `true`） |
| `CONOHA_ENDPOINT` | API エンドポイント上書き |
| `CONOHA_ENDPOINT_MODE` | `int` で内部APIモード（サービス名をパスに追加） |
| `CONOHA_YES` | 確認プロンプトを自動承認（`1` or `true`） |
| `CONOHA_NO_COLOR` | カラー出力を無効化（`1` or `true`） |
| `CONOHA_DEBUG` | デバッグログ（`1` or `api`） |

優先順位: 環境変数 > フラグ > プロファイル設定 > デフォルト値

### グローバルフラグ

```
--profile        使用するプロファイル
--format         出力形式（table / json / yaml / csv）
--no-input       対話プロンプトを無効化
--yes, -y        確認プロンプトを自動承認
--quiet          不要な出力を抑制
--verbose        詳細出力
--no-color       カラー出力を無効化
--no-headers     テーブル / CSV のヘッダーを非表示
--filter         行フィルタ（key=value、複数指定可）
--sort-by        行ソート（フィールド名）
--wait           非同期操作の完了を待機
--wait-timeout   待機タイムアウト（デフォルト 5m）
```

## 終了コード

| コード | 意味 |
|-------|------|
| 0 | 成功 |
| 1 | 一般エラー |
| 2 | 認証失敗 |
| 3 | リソース未検出 |
| 4 | バリデーションエラー |
| 5 | API エラー |
| 6 | ネットワークエラー |
| 10 | ユーザーキャンセル |

## エージェント連携

本 CLI はスクリプトや AI エージェントからの利用を想定して設計されています：

```bash
# 非対話モードで JSON 出力
conoha server list --format json --no-input

# トークンを取得してスクリプトで利用
TOKEN=$(conoha auth token)

# 終了コードでエラーハンドリング
conoha server show abc123 || echo "Exit code: $?"
```

## 開発

```bash
make build     # バイナリをビルド
make test      # テストを実行
make lint      # リンターを実行
make clean     # 成果物を削除
```

## API ドキュメント

- [ConoHa VPS3 API リファレンス](https://doc.conoha.jp/reference/api-vps3/)

## ライセンス

Apache License 2.0 - 詳細は [LICENSE](LICENSE) をご覧ください。
