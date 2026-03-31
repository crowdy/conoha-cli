# conoha-cli-app-samples Design

## Overview

`github.com/crowdy/conoha-cli-app-samples` — `conoha app deploy`用のサンプルアプリ集。各サンプルはそのまま`cd`して`conoha app deploy`できる独立したディレクトリ。

**Goals:**
- `conoha app deploy`の使い方を実際のコードで示す
- 各種フレームワークのDockerfile + compose.ymlのリファレンス
- 最小限のアプリコードで、デプロイフローにフォーカス

**Repository:** `github.com/crowdy/conoha-cli-app-samples` (public)
**Local path:** `/home/tkim/dev/crowdy/conoha-cli-app-samples`
**Language:** 日本語（README、コメント）
**License:** MIT

## Repo Structure

```
crowdy/conoha-cli-app-samples/
├── README.md                    # リポ全体の説明・使い方
├── LICENSE                      # MIT
├── hello-world/                 # 最小サンプル（nginx静的HTML）
│   ├── README.md
│   ├── compose.yml
│   ├── Dockerfile
│   ├── .dockerignore
│   └── index.html
├── nextjs/                      # Next.js デフォルトページ
│   ├── README.md
│   ├── compose.yml
│   ├── Dockerfile
│   ├── .dockerignore
│   └── (Next.js minimal source)
├── fastapi-ai-chatbot/          # FastAPI + Ollama
│   ├── README.md
│   ├── compose.yml
│   ├── Dockerfile
│   ├── .dockerignore
│   └── (Python source)
├── rails-postgresql/            # Rails scaffold + PostgreSQL
│   ├── README.md
│   ├── compose.yml
│   ├── Dockerfile
│   ├── .dockerignore
│   └── (Rails minimal source)
└── wordpress-mysql/             # WordPress + MySQL
    ├── README.md
    ├── compose.yml
    └── .dockerignore
```

## Deploy Flow

All samples follow the same flow:

```bash
# 1. サーバー作成（まだない場合）
conoha server create --name myserver --flavor g2l-t-2 --image ubuntu-24.04 --key mykey

# 2. アプリ初期化（Docker + git bare repoのセットアップ）
conoha app init myserver --app-name <sample-name>

# 3. サンプルディレクトリに移動してデプロイ
cd <sample-name>
conoha app deploy myserver --app-name <sample-name>

# 4. 動作確認
conoha app logs myserver --app-name <sample-name>
```

## Sample Specifications

### hello-world

最もシンプルなサンプル。初めて`conoha app deploy`を試すユーザー向け。

- **Stack:** nginx + 静的HTML
- **Port:** 80
- **Dockerfile:** nginx:alpine、index.htmlをCOPY
- **compose.yml:** 単一コンテナ、port 80:80
- **推奨フレーバー:** g2l-t-1（1GB RAM）

### nextjs

Next.jsのデフォルトページをデプロイ。

- **Stack:** Node.js + Next.js (standalone output)
- **Port:** 3000
- **Dockerfile:** マルチステージビルド（deps → build → runner）、standalone output mode
- **compose.yml:** 単一コンテナ、port 3000:3000
- **ソース:** `create-next-app`のデフォルトページ相当（最小限）
- **推奨フレーバー:** g2l-t-2（2GB RAM）

### fastapi-ai-chatbot

FastAPI + OllamaでシンプルなAIチャットボット。

- **Stack:** Python + FastAPI + Ollama
- **Port:** 8000（FastAPI）、11434（Ollama internal）
- **Dockerfile:** python:3.12-slim、pip install
- **compose.yml:** 2コンテナ（app + ollama）、depends_on
- **ソース:** `/chat`エンドポイント1つ、Ollamaにプロキシ、シンプルなHTML UI
- **推奨フレーバー:** g2l-t-4（4GB RAM、LLM用）
- **モデル:** tinyllama（軽量、デモ用）

### rails-postgresql

Rails scaffoldで最小限のWebアプリ。

- **Stack:** Ruby + Rails + PostgreSQL
- **Port:** 3000
- **Dockerfile:** ruby:3.3-slim、bundle install、assets precompile
- **compose.yml:** 2コンテナ（web + db）、depends_on、volume for db data
- **ソース:** `rails new` + 1つのscaffold（例: Post）
- **entrypoint:** DB migration自動実行（`rails db:prepare`）
- **推奨フレーバー:** g2l-t-2（2GB RAM）

### wordpress-mysql

WordPress + MySQL。Dockerfileなし、公式イメージのみ。

- **Stack:** WordPress (official) + MySQL (official)
- **Port:** 80
- **Dockerfile:** なし（公式イメージを直接使用）
- **compose.yml:** 2コンテナ（wordpress + db）、volumes for wp-content and db data、environment variables
- **推奨フレーバー:** g2l-t-2（2GB RAM）
- **備考:** `.env.server`でDB password等を設定する使い方を示す

## Each Sample README Structure

```markdown
# <Sample Name>

<1行説明>

## 構成

- <stack components>
- ポート: <port>

## 前提条件

- conoha-cli がインストール済み
- ConoHa VPS3 アカウント
- SSH キーペア設定済み

## デプロイ

​```bash
# サーバー作成（まだない場合）
conoha server create --name myserver --flavor <recommended> --image ubuntu-24.04 --key mykey

# アプリ初期化
conoha app init myserver --app-name <name>

# デプロイ
conoha app deploy myserver --app-name <name>
​```

## 動作確認

ブラウザで `http://<サーバーIP>:<port>` にアクセス。

## カスタマイズ

<フレームワーク固有のカスタマイズポイント>
```

## Root README Structure

1. **このリポジトリについて** — conoha-cli app deploy用サンプルアプリ集の説明
2. **前提条件** — conoha-cli, ConoHa VPS3 account, SSH keypair
3. **使い方** — 共通デプロイフロー（server create → app init → app deploy）
4. **サンプル一覧** — テーブル（名前、スタック、説明、推奨フレーバー）
5. **自分のアプリをデプロイするには** — compose.yml + DockerfileがあればOKという説明
6. **関連リンク** — conoha-cli本体、ドキュメントサイト

## .env.server Integration

WordPress等、環境変数でシークレットを渡す必要があるサンプルでは、`.env.server`の使い方を説明:

```bash
# サーバー上に環境変数ファイルを配置
conoha app env set myserver --app-name wordpress MYSQL_ROOT_PASSWORD=secret
conoha app env set myserver --app-name wordpress MYSQL_DATABASE=wordpress
```

deploy時に`/opt/conoha/<app-name>.env.server`が存在すれば、自動的に`.env`としてコピーされる。
