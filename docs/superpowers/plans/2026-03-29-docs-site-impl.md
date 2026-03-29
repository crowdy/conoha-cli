# conoha-cli.jp Documentation Site Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a VitePress documentation site for conoha-cli at conoha-cli.jp with Japanese/English/Korean support, tutorials, real-world examples, and command reference.

**Architecture:** Separate GitHub repo (`crowdy/conoha-cli-pages`), VitePress SSG, GitHub Actions auto-deploy to GitHub Pages with custom domain. Japanese content at root (`/`), English at `/en/`, Korean at `/ko/`.

**Tech Stack:** VitePress 1.x, Node.js, GitHub Actions, GitHub Pages

**Working directory:** `/home/tkim/dev/crowdy/conoha-cli-pages`

**Note:** This plan covers Phase 1 (MVP) — Japanese content only. Phases 2-3 (translations, more examples) are separate.

---

### Task 1: Create GitHub repo and scaffold VitePress project

**Files:**
- Create: `package.json`
- Create: `docs/.vitepress/config/index.ts`
- Create: `docs/.vitepress/config/shared.ts`
- Create: `docs/.vitepress/config/ja.ts`
- Create: `docs/.vitepress/config/en.ts`
- Create: `docs/.vitepress/config/ko.ts`
- Create: `docs/public/CNAME`
- Create: `docs/index.md`
- Create: `.gitignore`

- [ ] **Step 1: Create GitHub repo**

```bash
gh repo create crowdy/conoha-cli-pages --public --description "Documentation site for conoha-cli (conoha-cli.jp)" --clone
cd /home/tkim/dev/crowdy/conoha-cli-pages
```

- [ ] **Step 2: Initialize npm project and install VitePress**

```bash
cd /home/tkim/dev/crowdy/conoha-cli-pages
npm init -y
npm install -D vitepress
```

- [ ] **Step 3: Update package.json**

Replace `package.json` with:

```json
{
  "name": "conoha-cli-pages",
  "private": true,
  "scripts": {
    "docs:dev": "vitepress dev docs",
    "docs:build": "vitepress build docs",
    "docs:preview": "vitepress preview docs"
  },
  "devDependencies": {
    "vitepress": "^1.6.4"
  }
}
```

Then run `npm install` to update lock file.

- [ ] **Step 4: Create .gitignore**

```
node_modules/
docs/.vitepress/dist/
docs/.vitepress/cache/
```

- [ ] **Step 5: Create shared VitePress config**

Create `docs/.vitepress/config/shared.ts`:

```ts
import { defineConfig } from 'vitepress'

export const shared = defineConfig({
  title: 'ConoHa CLI',
  lastUpdated: true,
  cleanUrls: true,

  head: [
    ['link', { rel: 'icon', href: '/favicon.ico' }],
  ],

  themeConfig: {
    socialLinks: [
      { icon: 'github', link: 'https://github.com/crowdy/conoha-cli' },
    ],
    search: {
      provider: 'local',
    },
  },
})
```

- [ ] **Step 6: Create Japanese locale config**

Create `docs/.vitepress/config/ja.ts`:

```ts
import { DefaultTheme, LocaleSpecificConfig } from 'vitepress'

export const ja: LocaleSpecificConfig<DefaultTheme.Config> = {
  label: '日本語',
  lang: 'ja',
  description: 'ConoHa VPS3をコマンドラインから操作するCLIツール',

  themeConfig: {
    nav: [
      { text: 'ガイド', link: '/guide/getting-started' },
      { text: '実践例', link: '/examples/nextjs' },
      { text: 'リファレンス', link: '/reference/auth' },
    ],

    sidebar: {
      '/guide/': [
        {
          text: 'ガイド',
          items: [
            { text: 'はじめに', link: '/guide/getting-started' },
            { text: 'サーバー管理', link: '/guide/server' },
            { text: 'アプリデプロイ', link: '/guide/app-deploy' },
            { text: 'アプリ管理', link: '/guide/app-management' },
          ],
        },
      ],
      '/examples/': [
        {
          text: '実践デプロイ例',
          items: [
            { text: 'Next.js', link: '/examples/nextjs' },
            { text: 'FastAPI + AIチャットボット', link: '/examples/fastapi-ai-chatbot' },
            { text: 'Rails + PostgreSQL', link: '/examples/rails-postgresql' },
            { text: 'WordPress', link: '/examples/wordpress' },
          ],
        },
      ],
      '/reference/': [
        {
          text: 'コマンドリファレンス',
          items: [
            { text: 'auth', link: '/reference/auth' },
            { text: 'server', link: '/reference/server' },
            { text: 'app', link: '/reference/app' },
          ],
        },
      ],
    },

    editLink: {
      pattern: 'https://github.com/crowdy/conoha-cli-pages/edit/main/docs/:path',
      text: 'このページを編集する',
    },

    lastUpdated: {
      text: '最終更新',
    },

    outline: {
      label: '目次',
    },

    docFooter: {
      prev: '前のページ',
      next: '次のページ',
    },
  },
}
```

- [ ] **Step 7: Create English locale config**

Create `docs/.vitepress/config/en.ts`:

```ts
import { DefaultTheme, LocaleSpecificConfig } from 'vitepress'

export const en: LocaleSpecificConfig<DefaultTheme.Config> = {
  label: 'English',
  lang: 'en',
  description: 'CLI tool for ConoHa VPS3 API',

  themeConfig: {
    nav: [
      { text: 'Guide', link: '/en/guide/getting-started' },
      { text: 'Examples', link: '/en/examples/nextjs' },
      { text: 'Reference', link: '/en/reference/auth' },
    ],

    sidebar: {
      '/en/guide/': [
        {
          text: 'Guide',
          items: [
            { text: 'Getting Started', link: '/en/guide/getting-started' },
            { text: 'Server Management', link: '/en/guide/server' },
            { text: 'App Deploy', link: '/en/guide/app-deploy' },
            { text: 'App Management', link: '/en/guide/app-management' },
          ],
        },
      ],
      '/en/examples/': [
        {
          text: 'Deployment Examples',
          items: [
            { text: 'Next.js', link: '/en/examples/nextjs' },
            { text: 'FastAPI + AI Chatbot', link: '/en/examples/fastapi-ai-chatbot' },
            { text: 'Rails + PostgreSQL', link: '/en/examples/rails-postgresql' },
            { text: 'WordPress', link: '/en/examples/wordpress' },
          ],
        },
      ],
      '/en/reference/': [
        {
          text: 'Command Reference',
          items: [
            { text: 'auth', link: '/en/reference/auth' },
            { text: 'server', link: '/en/reference/server' },
            { text: 'app', link: '/en/reference/app' },
          ],
        },
      ],
    },

    editLink: {
      pattern: 'https://github.com/crowdy/conoha-cli-pages/edit/main/docs/:path',
      text: 'Edit this page',
    },
  },
}
```

- [ ] **Step 8: Create Korean locale config**

Create `docs/.vitepress/config/ko.ts`:

```ts
import { DefaultTheme, LocaleSpecificConfig } from 'vitepress'

export const ko: LocaleSpecificConfig<DefaultTheme.Config> = {
  label: '한국어',
  lang: 'ko',
  description: 'ConoHa VPS3 API용 CLI 도구',

  themeConfig: {
    nav: [
      { text: '가이드', link: '/ko/guide/getting-started' },
      { text: '실전 예제', link: '/ko/examples/nextjs' },
      { text: '레퍼런스', link: '/ko/reference/auth' },
    ],

    sidebar: {
      '/ko/guide/': [
        {
          text: '가이드',
          items: [
            { text: '시작하기', link: '/ko/guide/getting-started' },
            { text: '서버 관리', link: '/ko/guide/server' },
            { text: '앱 배포', link: '/ko/guide/app-deploy' },
            { text: '앱 관리', link: '/ko/guide/app-management' },
          ],
        },
      ],
      '/ko/examples/': [
        {
          text: '실전 배포 예제',
          items: [
            { text: 'Next.js', link: '/ko/examples/nextjs' },
            { text: 'FastAPI + AI 챗봇', link: '/ko/examples/fastapi-ai-chatbot' },
            { text: 'Rails + PostgreSQL', link: '/ko/examples/rails-postgresql' },
            { text: 'WordPress', link: '/ko/examples/wordpress' },
          ],
        },
      ],
      '/ko/reference/': [
        {
          text: '커맨드 레퍼런스',
          items: [
            { text: 'auth', link: '/ko/reference/auth' },
            { text: 'server', link: '/ko/reference/server' },
            { text: 'app', link: '/ko/reference/app' },
          ],
        },
      ],
    },

    editLink: {
      pattern: 'https://github.com/crowdy/conoha-cli-pages/edit/main/docs/:path',
      text: '이 페이지 편집하기',
    },

    lastUpdated: {
      text: '마지막 업데이트',
    },

    outline: {
      label: '목차',
    },

    docFooter: {
      prev: '이전 페이지',
      next: '다음 페이지',
    },
  },
}
```

- [ ] **Step 9: Create main config that merges locales**

Create `docs/.vitepress/config/index.ts`:

```ts
import { defineConfig } from 'vitepress'
import { shared } from './shared'
import { ja } from './ja'
import { en } from './en'
import { ko } from './ko'

export default defineConfig({
  ...shared,
  locales: {
    root: { ...ja },
    en: { ...en },
    ko: { ...ko },
  },
})
```

- [ ] **Step 10: Create CNAME and landing page**

Create `docs/public/CNAME`:

```
conoha-cli.jp
```

Create `docs/index.md`:

```md
---
layout: home
hero:
  name: ConoHa CLI
  text: ConoHa VPS3をコマンドラインから操作
  tagline: サーバー作成からアプリデプロイまで、すべてターミナルから
  actions:
    - theme: brand
      text: はじめに
      link: /guide/getting-started
    - theme: alt
      text: GitHub
      link: https://github.com/crowdy/conoha-cli
features:
  - title: かんたんインストール
    details: Go製のシングルバイナリ。ダウンロードしてすぐ使えます。
  - title: アプリデプロイ
    details: Dockerfileがあれば conoha app deploy の一発でデプロイ完了。
  - title: フル機能
    details: サーバー・ネットワーク・DNS・ストレージ・ロードバランサーまで全API対応。
---
```

- [ ] **Step 11: Verify VitePress builds successfully**

```bash
cd /home/tkim/dev/crowdy/conoha-cli-pages
npx vitepress build docs
```

Expected: Build succeeds with warnings about missing linked pages (guide, examples, reference pages not yet created).

- [ ] **Step 12: Commit**

```bash
git add -A
git commit -m "Scaffold VitePress project with i18n config"
```

---

### Task 2: GitHub Actions deploy workflow

**Files:**
- Create: `.github/workflows/deploy.yml`

- [ ] **Step 1: Create deploy workflow**

Create `.github/workflows/deploy.yml`:

```yaml
name: Deploy to GitHub Pages

on:
  push:
    branches: [main]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: pages
  cancel-in-progress: false

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Setup Node
        uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: npm

      - name: Setup Pages
        uses: actions/configure-pages@v4

      - name: Install dependencies
        run: npm ci

      - name: Build
        run: npm run docs:build

      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: docs/.vitepress/dist

  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    needs: build
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/deploy.yml
git commit -m "Add GitHub Actions deploy workflow"
```

---

### Task 3: Japanese landing page placeholders for linked pages

Create minimal placeholder pages so the landing page links work and VitePress builds without broken links. These will be filled with real content in subsequent tasks.

**Files:**
- Create: `docs/guide/getting-started.md`
- Create: `docs/guide/server.md`
- Create: `docs/guide/app-deploy.md`
- Create: `docs/guide/app-management.md`
- Create: `docs/examples/nextjs.md`
- Create: `docs/examples/fastapi-ai-chatbot.md`
- Create: `docs/examples/rails-postgresql.md`
- Create: `docs/examples/wordpress.md`
- Create: `docs/reference/auth.md`
- Create: `docs/reference/server.md`
- Create: `docs/reference/app.md`

- [ ] **Step 1: Create guide placeholders**

Create `docs/guide/getting-started.md`:

```md
# はじめに

::: tip 準備中
このページは現在執筆中です。
:::
```

Create `docs/guide/server.md`:

```md
# サーバー管理

::: tip 準備中
このページは現在執筆中です。
:::
```

Create `docs/guide/app-deploy.md`:

```md
# アプリデプロイ

::: tip 準備中
このページは現在執筆中です。
:::
```

Create `docs/guide/app-management.md`:

```md
# アプリ管理

::: tip 準備中
このページは現在執筆中です。
:::
```

- [ ] **Step 2: Create examples placeholders**

Create `docs/examples/nextjs.md`:

```md
# Next.js デプロイ

::: tip 準備中
このページは現在執筆中です。
:::
```

Create `docs/examples/fastapi-ai-chatbot.md`:

```md
# FastAPI + AIチャットボット

::: tip 準備中
このページは現在執筆中です。
:::
```

Create `docs/examples/rails-postgresql.md`:

```md
# Rails + PostgreSQL デプロイ

::: tip 準備中
このページは現在執筆中です。
:::
```

Create `docs/examples/wordpress.md`:

```md
# WordPress デプロイ

::: tip 準備中
このページは現在執筆中です。
:::
```

- [ ] **Step 3: Create reference placeholders**

Create `docs/reference/auth.md`:

```md
# auth

::: tip 準備中
このページは現在執筆中です。
:::
```

Create `docs/reference/server.md`:

```md
# server

::: tip 準備中
このページは現在執筆中です。
:::
```

Create `docs/reference/app.md`:

```md
# app

::: tip 準備中
このページは現在執筆中です。
:::
```

- [ ] **Step 4: Verify build succeeds with no broken links**

```bash
npx vitepress build docs
```

Expected: Build succeeds with no warnings.

- [ ] **Step 5: Commit**

```bash
git add docs/
git commit -m "Add placeholder pages for all sections"
```

---

### Task 4: Guide — Getting Started

**Files:**
- Modify: `docs/guide/getting-started.md`

- [ ] **Step 1: Write getting-started guide**

Replace `docs/guide/getting-started.md` with:

```md
# はじめに

ConoHa CLIは、ConoHa VPS3をターミナルから操作するためのコマンドラインツールです。

## インストール

### macOS (Homebrew)

```bash
brew install crowdy/tap/conoha
```

### Linux / macOS (手動)

[GitHub Releases](https://github.com/crowdy/conoha-cli/releases) からお使いのOS・アーキテクチャに合ったバイナリをダウンロードしてください。

```bash
# 例: Linux amd64
curl -LO https://github.com/crowdy/conoha-cli/releases/latest/download/conoha_linux_amd64.tar.gz
tar xzf conoha_linux_amd64.tar.gz
sudo mv conoha /usr/local/bin/
```

### Windows

[GitHub Releases](https://github.com/crowdy/conoha-cli/releases) から `conoha_windows_amd64.zip` をダウンロードし、パスの通ったディレクトリに配置してください。

## インストール確認

```bash
conoha version
```

バージョン番号が表示されればOKです。

## ログイン

ConoHa APIのユーザー名・パスワード・テナントIDを使ってログインします。これらは [ConoHaコントロールパネル](https://manage.conoha.jp/) の「API」ページで確認できます。

```bash
conoha auth login
```

対話形式で以下を入力します:

- **API User**: APIユーザー名
- **Password**: APIパスワード
- **Tenant ID**: テナントID
- **Region**: tyo3 (東京)

::: tip
`--profile` オプションで複数のアカウントを管理できます。

```bash
conoha auth login --profile work
conoha auth login --profile personal
conoha auth switch work
```
:::

## ログイン確認

```bash
conoha auth status
```

トークンの有効期限とプロファイル情報が表示されます。

## 基本的な使い方

```bash
# サーバー一覧
conoha server list

# JSON形式で出力
conoha server list --format json

# ヘルプを見る
conoha --help
conoha server --help
conoha server create --help
```

## 出力フォーマット

すべてのコマンドで `--format` オプションが使えます。

| フォーマット | 説明 |
|-------------|------|
| `table` | テーブル形式（デフォルト） |
| `json` | JSON形式 |
| `yaml` | YAML形式 |
| `csv` | CSV形式 |

## 次のステップ

- [サーバー管理](/guide/server) — サーバーの作成・起動・停止
- [アプリデプロイ](/guide/app-deploy) — Dockerアプリのデプロイ
```

- [ ] **Step 2: Verify build**

```bash
npx vitepress build docs
```

- [ ] **Step 3: Commit**

```bash
git add docs/guide/getting-started.md
git commit -m "Add getting-started guide"
```

---

### Task 5: Guide — Server Management

**Files:**
- Modify: `docs/guide/server.md`

- [ ] **Step 1: Write server guide**

Replace `docs/guide/server.md` with:

```md
# サーバー管理

ConoHa CLIでサーバーの作成から管理まですべて行えます。

## サーバー一覧

```bash
conoha server list
```

## サーバー作成

### フレーバー（スペック）を選ぶ

```bash
conoha flavor list
```

主なフレーバー:

| フレーバー | CPU | メモリ | ディスク |
|-----------|-----|--------|---------|
| g2l-t-c1m05d30 | 1 vCPU | 512MB | 30GB |
| g2l-t-c2m1d100 | 2 vCPU | 1GB | 100GB |
| g2l-t-c3m2d100 | 3 vCPU | 2GB | 100GB |
| g2l-t-c4m4d100 | 4 vCPU | 4GB | 100GB |

### イメージを選ぶ

```bash
conoha image list
```

### SSHキーペアを作成

```bash
conoha keypair create --name mykey
```

秘密鍵が表示されるので、ファイルに保存してください:

```bash
conoha keypair create --name mykey > ~/.ssh/conoha_mykey
chmod 600 ~/.ssh/conoha_mykey
```

### サーバーを作成

```bash
conoha server create \
  --name myserver \
  --flavor g2l-t-c2m1d100 \
  --image ubuntu-24.04 \
  --key-name mykey
```

作成完了まで1〜2分かかります。

## サーバーの起動・停止

```bash
# 停止
conoha server stop <サーバー名またはID>

# 起動
conoha server start <サーバー名またはID>

# 再起動
conoha server reboot <サーバー名またはID>
```

## SSHログイン

```bash
conoha server ssh <サーバー名> --key ~/.ssh/conoha_mykey
```

## IPアドレスの確認

```bash
conoha server ips <サーバー名>
```

## サーバー削除

```bash
conoha server delete <サーバー名またはID>
```

::: warning
削除したサーバーは復元できません。
:::

## 次のステップ

- [アプリデプロイ](/guide/app-deploy) — Dockerアプリをサーバーにデプロイ
```

- [ ] **Step 2: Verify build**

```bash
npx vitepress build docs
```

- [ ] **Step 3: Commit**

```bash
git add docs/guide/server.md
git commit -m "Add server management guide"
```

---

### Task 6: Guide — App Deploy

**Files:**
- Modify: `docs/guide/app-deploy.md`

- [ ] **Step 1: Write app deploy guide**

Replace `docs/guide/app-deploy.md` with:

```md
# アプリデプロイ

ConoHa CLIを使えば、Dockerfileがあるプロジェクトをコマンド一発でデプロイできます。

## 前提条件

- サーバーが作成済み（[サーバー管理](/guide/server)を参照）
- サーバーにDockerがインストール済み
- プロジェクトに `Dockerfile` と `docker-compose.yml` がある

## デプロイの流れ

```
app init → app deploy → app logs で確認
```

## 1. アプリの初期化

サーバー上にアプリの受け口を作成します。

```bash
conoha app init <サーバー名> --app-name myapp
```

これにより、サーバー上に以下が作成されます:
- `/opt/conoha/myapp/` — 作業ディレクトリ
- `/opt/conoha/myapp.git/` — Gitリポジトリ（push受信用）
- post-receiveフック — push時に自動で `docker compose up -d`

## 2. アプリのデプロイ

プロジェクトのディレクトリで実行します。

```bash
cd /path/to/your/project
conoha app deploy <サーバー名> --app-name myapp
```

実行内容:
1. プロジェクトファイルをtarで圧縮
2. サーバーにSSHで転送
3. `docker compose up -d --build` を実行

`.dockerignore` があれば、記載されたファイルは除外されます。`.git/` ディレクトリは常に除外されます。

## 3. 動作確認

### ログを見る

```bash
conoha app logs <サーバー名> --app-name myapp
```

リアルタイムでフォロー:

```bash
conoha app logs <サーバー名> --app-name myapp --follow
```

### ステータス確認

```bash
conoha app status <サーバー名> --app-name myapp
```

コンテナの状態（running/stopped）が表示されます。

## アプリの再デプロイ

コードを変更したら、同じコマンドで再デプロイできます:

```bash
conoha app deploy <サーバー名> --app-name myapp
```

## アプリの停止・再起動

```bash
# 停止
conoha app stop <サーバー名> --app-name myapp

# 再起動
conoha app restart <サーバー名> --app-name myapp
```

## docker-compose.yml の例

```yaml
services:
  web:
    build: .
    ports:
      - "80:3000"
    restart: unless-stopped
```

## 次のステップ

- [アプリ管理](/guide/app-management) — 環境変数・削除・一覧
- [実践デプロイ例](/examples/nextjs) — フレームワーク別のデプロイ手順
```

- [ ] **Step 2: Verify build**

```bash
npx vitepress build docs
```

- [ ] **Step 3: Commit**

```bash
git add docs/guide/app-deploy.md
git commit -m "Add app deploy guide"
```

---

### Task 7: Guide — App Management

**Files:**
- Modify: `docs/guide/app-management.md`

- [ ] **Step 1: Write app management guide**

Replace `docs/guide/app-management.md` with:

```md
# アプリ管理

デプロイしたアプリの環境変数管理、削除、一覧表示を行います。

## 環境変数

### 環境変数を設定

```bash
conoha app env set <サーバー名> --app-name myapp DATABASE_URL=postgres://... SECRET_KEY=mysecret
```

複数の変数を一度に設定できます。

### 環境変数を確認

```bash
# 一覧
conoha app env list <サーバー名> --app-name myapp

# 特定の変数を取得
conoha app env get <サーバー名> --app-name myapp DATABASE_URL
```

### 環境変数を削除

```bash
conoha app env unset <サーバー名> --app-name myapp SECRET_KEY
```

### 環境変数の反映

環境変数を変更した後、アプリに反映するには再デプロイが必要です:

```bash
conoha app deploy <サーバー名> --app-name myapp
```

::: tip 仕組み
環境変数はサーバー上の `/opt/conoha/{app-name}.env.server` に保存されます。`app deploy` 実行時にこのファイルが `.env` としてコピーされ、docker composeから参照されます。
:::

## デプロイ済みアプリ一覧

```bash
conoha app list <サーバー名>
```

アプリ名とコンテナの状態（running / stopped / no containers）が表示されます。

## アプリの削除

```bash
conoha app destroy <サーバー名> --app-name myapp
```

確認プロンプトが表示されます。 `--yes` で確認をスキップできます。

::: warning
削除すると以下がすべて消えます:
- コンテナ（停止・削除）
- 作業ディレクトリ（`/opt/conoha/myapp/`）
- Gitリポジトリ（`/opt/conoha/myapp.git/`）
- 環境変数ファイル（`/opt/conoha/myapp.env.server`）
:::
```

- [ ] **Step 2: Verify build**

```bash
npx vitepress build docs
```

- [ ] **Step 3: Commit**

```bash
git add docs/guide/app-management.md
git commit -m "Add app management guide"
```

---

### Task 8: Example — Next.js

**Files:**
- Modify: `docs/examples/nextjs.md`

- [ ] **Step 1: Write Next.js example**

Replace `docs/examples/nextjs.md` with:

```md
# Next.js デプロイ

Next.jsアプリをConoHa VPSにデプロイする手順です。Vercelの代替として、自分のサーバーでNext.jsを動かしたい方向け。

## 完成イメージ

- Next.js アプリが `http://<サーバーIP>` でアクセス可能
- `conoha app deploy` でコード更新を即座に反映

## 前提条件

- ConoHa CLIがインストール・ログイン済み（[はじめに](/guide/getting-started)）
- サーバーが作成済み（[サーバー管理](/guide/server)）

## 1. Next.js プロジェクトを作成

```bash
npx create-next-app@latest myapp
cd myapp
```

## 2. Dockerfile を作成

```dockerfile
FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public
EXPOSE 3000
CMD ["node", "server.js"]
```

::: tip
Next.jsの `standalone` 出力を使うには、`next.config.ts` に以下を追加:

```ts
const nextConfig = {
  output: 'standalone',
}
```
:::

## 3. docker-compose.yml を作成

```yaml
services:
  web:
    build: .
    ports:
      - "80:3000"
    restart: unless-stopped
```

## 4. .dockerignore を作成

```
node_modules
.next
.git
```

## 5. デプロイ

```bash
# 初期化（初回のみ）
conoha app init <サーバー名> --app-name myapp

# デプロイ
conoha app deploy <サーバー名> --app-name myapp
```

## 6. 動作確認

```bash
# ステータス確認
conoha app status <サーバー名> --app-name myapp

# ログ確認
conoha app logs <サーバー名> --app-name myapp
```

ブラウザで `http://<サーバーIP>` にアクセスして、Next.jsのページが表示されれば完了です。

## 環境変数を使う場合

```bash
conoha app env set <サーバー名> --app-name myapp \
  DATABASE_URL=postgres://user:pass@db:5432/mydb \
  NEXT_PUBLIC_API_URL=https://api.example.com

# 再デプロイで反映
conoha app deploy <サーバー名> --app-name myapp
```

## コード更新

コードを変更したら、同じコマンドで再デプロイ:

```bash
conoha app deploy <サーバー名> --app-name myapp
```
```

- [ ] **Step 2: Verify build**

```bash
npx vitepress build docs
```

- [ ] **Step 3: Commit**

```bash
git add docs/examples/nextjs.md
git commit -m "Add Next.js deploy example"
```

---

### Task 9: Example — FastAPI + AI Chatbot

**Files:**
- Modify: `docs/examples/fastapi-ai-chatbot.md`

- [ ] **Step 1: Write FastAPI AI chatbot example**

Replace `docs/examples/fastapi-ai-chatbot.md` with:

```md
# FastAPI + AIチャットボット

FastAPIとOllamaを使って、セルフホスティングのAIチャットボットをConoHa VPSにデプロイする手順です。

## 完成イメージ

- FastAPI製のチャットAPI が `http://<サーバーIP>` でアクセス可能
- Ollama でLLMモデルをローカル実行（APIキー不要）

## 前提条件

- ConoHa CLIがインストール・ログイン済み
- **メモリ4GB以上のサーバー**を推奨（LLM実行のため）

## 1. プロジェクト構成

```
fastapi-chatbot/
├── app/
│   └── main.py
├── Dockerfile
├── docker-compose.yml
├── requirements.txt
└── .dockerignore
```

## 2. FastAPI アプリ

`app/main.py`:

```python
from fastapi import FastAPI
from pydantic import BaseModel
import httpx

app = FastAPI()

class ChatRequest(BaseModel):
    message: str

class ChatResponse(BaseModel):
    reply: str

@app.post("/chat", response_model=ChatResponse)
async def chat(req: ChatRequest):
    async with httpx.AsyncClient() as client:
        resp = await client.post(
            "http://ollama:11434/api/generate",
            json={"model": "gemma3:4b", "prompt": req.message, "stream": False},
        )
        data = resp.json()
    return ChatResponse(reply=data["response"])

@app.get("/health")
async def health():
    return {"status": "ok"}
```

`requirements.txt`:

```
fastapi>=0.115
uvicorn>=0.34
httpx>=0.28
```

## 3. Dockerfile

```dockerfile
FROM python:3.13-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY app/ ./app/
EXPOSE 8000
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
```

## 4. docker-compose.yml

```yaml
services:
  web:
    build: .
    ports:
      - "80:8000"
    depends_on:
      - ollama
    restart: unless-stopped

  ollama:
    image: ollama/ollama
    volumes:
      - ollama_data:/root/.ollama
    restart: unless-stopped

volumes:
  ollama_data:
```

## 5. .dockerignore

```
__pycache__
*.pyc
.git
.venv
```

## 6. デプロイ

```bash
conoha app init <サーバー名> --app-name chatbot
conoha app deploy <サーバー名> --app-name chatbot
```

## 7. モデルをダウンロード

初回デプロイ後、Ollamaにモデルをダウンロードさせます:

```bash
conoha server ssh <サーバー名> --key ~/.ssh/conoha_mykey
# サーバー内で:
docker exec -it chatbot-ollama-1 ollama pull gemma3:4b
```

## 8. 動作確認

```bash
curl http://<サーバーIP>/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "こんにちは、自己紹介してください"}'
```

```json
{"reply": "こんにちは！私はAIアシスタントです。..."}
```
```

- [ ] **Step 2: Verify build**

```bash
npx vitepress build docs
```

- [ ] **Step 3: Commit**

```bash
git add docs/examples/fastapi-ai-chatbot.md
git commit -m "Add FastAPI AI chatbot example"
```

---

### Task 10: Example — Rails + PostgreSQL

**Files:**
- Modify: `docs/examples/rails-postgresql.md`

- [ ] **Step 1: Write Rails + PostgreSQL example**

Replace `docs/examples/rails-postgresql.md` with:

```md
# Rails + PostgreSQL デプロイ

Ruby on RailsアプリをPostgreSQLと一緒にConoHa VPSにデプロイする手順です。

## 完成イメージ

- Railsアプリが `http://<サーバーIP>` でアクセス可能
- PostgreSQLがサイドカーコンテナで動作

## 前提条件

- ConoHa CLIがインストール・ログイン済み
- メモリ1GB以上のサーバー

## 1. Rails プロジェクトを作成

```bash
rails new myapp --database=postgresql
cd myapp
```

## 2. Dockerfile

```dockerfile
FROM ruby:3.4-slim AS builder
RUN apt-get update && apt-get install -y build-essential libpq-dev
WORKDIR /app
COPY Gemfile Gemfile.lock ./
RUN bundle install --jobs 4 --without development test

FROM ruby:3.4-slim
RUN apt-get update && apt-get install -y libpq5 && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /usr/local/bundle /usr/local/bundle
COPY . .
RUN bundle exec rails assets:precompile SECRET_KEY_BASE=dummy
EXPOSE 3000
CMD ["bundle", "exec", "rails", "server", "-b", "0.0.0.0"]
```

## 3. docker-compose.yml

```yaml
services:
  web:
    build: .
    ports:
      - "80:3000"
    depends_on:
      - db
    environment:
      - DATABASE_URL=postgres://postgres:${POSTGRES_PASSWORD}@db:5432/myapp_production
      - RAILS_ENV=production
      - SECRET_KEY_BASE=${SECRET_KEY_BASE}
    restart: unless-stopped

  db:
    image: postgres:17
    volumes:
      - pg_data:/var/lib/postgresql/data
    environment:
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - POSTGRES_DB=myapp_production
    restart: unless-stopped

volumes:
  pg_data:
```

## 4. .dockerignore

```
.git
log/*
tmp/*
node_modules
```

## 5. config/database.yml を修正

```yaml
production:
  url: <%= ENV["DATABASE_URL"] %>
```

## 6. デプロイ

```bash
# 初期化
conoha app init <サーバー名> --app-name myapp

# 環境変数を設定
conoha app env set <サーバー名> --app-name myapp \
  POSTGRES_PASSWORD=your-secure-password \
  SECRET_KEY_BASE=$(rails secret)

# デプロイ
conoha app deploy <サーバー名> --app-name myapp
```

## 7. データベースのセットアップ

初回デプロイ後:

```bash
conoha server ssh <サーバー名> --key ~/.ssh/conoha_mykey
# サーバー内で:
cd /opt/conoha/myapp
docker compose exec web bundle exec rails db:create db:migrate
```

## 8. 動作確認

ブラウザで `http://<サーバーIP>` にアクセスしてRailsのページが表示されれば完了です。

```bash
conoha app status <サーバー名> --app-name myapp
conoha app logs <サーバー名> --app-name myapp
```
```

- [ ] **Step 2: Verify build**

```bash
npx vitepress build docs
```

- [ ] **Step 3: Commit**

```bash
git add docs/examples/rails-postgresql.md
git commit -m "Add Rails + PostgreSQL deploy example"
```

---

### Task 11: Example — WordPress

**Files:**
- Modify: `docs/examples/wordpress.md`

- [ ] **Step 1: Write WordPress example**

Replace `docs/examples/wordpress.md` with:

```md
# WordPress デプロイ

WordPressをConoHa VPSにデプロイする手順です。レンタルサーバーからVPSに移行したい方向け。

## 完成イメージ

- WordPressが `http://<サーバーIP>` でアクセス可能
- MySQLがサイドカーコンテナで動作
- データは永続化

## 前提条件

- ConoHa CLIがインストール・ログイン済み
- メモリ1GB以上のサーバー

## 1. プロジェクト構成

```
wordpress-site/
├── docker-compose.yml
└── .dockerignore
```

WordPressは公式Dockerイメージを使うため、Dockerfileは不要です。

## 2. docker-compose.yml

```yaml
services:
  wordpress:
    image: wordpress:6
    ports:
      - "80:80"
    depends_on:
      - db
    environment:
      - WORDPRESS_DB_HOST=db
      - WORDPRESS_DB_USER=wordpress
      - WORDPRESS_DB_PASSWORD=${MYSQL_PASSWORD}
      - WORDPRESS_DB_NAME=wordpress
    volumes:
      - wp_data:/var/www/html
    restart: unless-stopped

  db:
    image: mysql:8
    environment:
      - MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD}
      - MYSQL_DATABASE=wordpress
      - MYSQL_USER=wordpress
      - MYSQL_PASSWORD=${MYSQL_PASSWORD}
    volumes:
      - db_data:/var/lib/mysql
    restart: unless-stopped

volumes:
  wp_data:
  db_data:
```

## 3. .dockerignore

```
.git
```

## 4. デプロイ

```bash
# 初期化
conoha app init <サーバー名> --app-name wordpress

# 環境変数を設定
conoha app env set <サーバー名> --app-name wordpress \
  MYSQL_PASSWORD=your-secure-password \
  MYSQL_ROOT_PASSWORD=your-root-password

# デプロイ
conoha app deploy <サーバー名> --app-name wordpress
```

## 5. 動作確認

ブラウザで `http://<サーバーIP>` にアクセスすると、WordPressのセットアップ画面が表示されます。

```bash
conoha app status <サーバー名> --app-name wordpress
conoha app logs <サーバー名> --app-name wordpress
```

## バックアップ

データベースのバックアップ:

```bash
conoha server ssh <サーバー名> --key ~/.ssh/conoha_mykey
# サーバー内で:
cd /opt/conoha/wordpress
docker compose exec db mysqldump -u root -p wordpress > backup.sql
```
```

- [ ] **Step 2: Verify build**

```bash
npx vitepress build docs
```

- [ ] **Step 3: Commit**

```bash
git add docs/examples/wordpress.md
git commit -m "Add WordPress deploy example"
```

---

### Task 12: Reference — auth

**Files:**
- Modify: `docs/reference/auth.md`

- [ ] **Step 1: Write auth reference**

Replace `docs/reference/auth.md` with:

```md
# auth

認証の管理を行うコマンドグループです。

## auth login

APIの認証情報を入力してログインします。

### 使い方

```bash
conoha auth login [flags]
```

### オプション

| オプション | 説明 |
|-----------|------|
| `--profile` | 保存するプロファイル名（デフォルト: default） |

### 例

```bash
# 対話形式でログイン
conoha auth login

# プロファイルを指定
conoha auth login --profile work
```

---

## auth status

現在の認証状態を表示します。

### 使い方

```bash
conoha auth status
```

### 例

```bash
conoha auth status
```

---

## auth list

設定済みのプロファイル一覧を表示します。

### 使い方

```bash
conoha auth list
```

---

## auth switch

アクティブなプロファイルを切り替えます。

### 使い方

```bash
conoha auth switch <プロファイル名>
```

### 例

```bash
conoha auth switch work
```

---

## auth token

現在のトークンを標準出力に出力します。スクリプトから利用する場合に便利です。

### 使い方

```bash
conoha auth token
```

### 例

```bash
# 他のコマンドでトークンを使う
curl -H "X-Auth-Token: $(conoha auth token)" https://...
```

---

## auth logout

アクティブなプロファイルのトークンと認証情報を削除します。

### 使い方

```bash
conoha auth logout
```

---

## auth remove

プロファイルを完全に削除します。

### 使い方

```bash
conoha auth remove <プロファイル名>
```
```

- [ ] **Step 2: Verify build**

```bash
npx vitepress build docs
```

- [ ] **Step 3: Commit**

```bash
git add docs/reference/auth.md
git commit -m "Add auth command reference"
```

---

### Task 13: Reference — server

**Files:**
- Modify: `docs/reference/server.md`

- [ ] **Step 1: Write server reference**

Replace `docs/reference/server.md` with:

```md
# server

サーバー（VM）の管理を行うコマンドグループです。

## server list

サーバー一覧を表示します。

### 使い方

```bash
conoha server list
```

### 例

```bash
# テーブル形式
conoha server list

# JSON形式
conoha server list --format json

# フィルタリング
conoha server list --filter status=ACTIVE
```

---

## server show

サーバーの詳細情報を表示します。

### 使い方

```bash
conoha server show <サーバー名またはID>
```

---

## server create

新しいサーバーを作成します。

### 使い方

```bash
conoha server create [flags]
```

### オプション

| オプション | 説明 | 必須 |
|-----------|------|------|
| `--name` | サーバー名 | ○ |
| `--flavor` | フレーバーID | ○ |
| `--image` | イメージ名またはID | ○ |
| `--key-name` | SSHキーペア名 | |
| `--security-group` | セキュリティグループ名 | |
| `--startup-script` | 起動スクリプトファイルパス | |

### 例

```bash
conoha server create \
  --name myserver \
  --flavor g2l-t-c2m1d100 \
  --image ubuntu-24.04 \
  --key-name mykey
```

---

## server delete

サーバーを削除します。

### 使い方

```bash
conoha server delete <サーバー名またはID>
```

---

## server start

停止中のサーバーを起動します。

### 使い方

```bash
conoha server start <サーバー名またはID>
```

---

## server stop

サーバーを停止します。

### 使い方

```bash
conoha server stop <サーバー名またはID>
```

---

## server reboot

サーバーを再起動します。

### 使い方

```bash
conoha server reboot <サーバー名またはID>
```

### オプション

| オプション | 説明 |
|-----------|------|
| `--hard` | ハードリブート |

---

## server resize

サーバーのスペックを変更します。

### 使い方

```bash
conoha server resize <サーバー名またはID> --flavor <フレーバーID>
```

---

## server rebuild

サーバーを新しいイメージで再構築します。

### 使い方

```bash
conoha server rebuild <サーバー名またはID> --image <イメージ名またはID>
```

---

## server rename

サーバー名を変更します。

### 使い方

```bash
conoha server rename <サーバー名またはID> --name <新しい名前>
```

---

## server ssh

サーバーにSSH接続します。

### 使い方

```bash
conoha server ssh <サーバー名> [flags]
```

### オプション

| オプション | 説明 |
|-----------|------|
| `--key` | 秘密鍵のパス |
| `--user` | ユーザー名（デフォルト: root） |

### 例

```bash
conoha server ssh myserver --key ~/.ssh/conoha_mykey
```

---

## server deploy

サーバー上でスクリプトを実行します。

### 使い方

```bash
conoha server deploy <サーバー名> --script <スクリプトファイル>
```

---

## server console

VNCコンソールのURLを取得します。

### 使い方

```bash
conoha server console <サーバー名またはID>
```

---

## server ips

サーバーのIPアドレス一覧を表示します。

### 使い方

```bash
conoha server ips <サーバー名またはID>
```

---

## server metadata

サーバーのメタデータを表示します。

### 使い方

```bash
conoha server metadata <サーバー名またはID>
```

---

## server attach-volume

ボリュームをサーバーにアタッチします。

### 使い方

```bash
conoha server attach-volume <サーバー名またはID> --volume <ボリュームID>
```

---

## server detach-volume

ボリュームをサーバーからデタッチします。

### 使い方

```bash
conoha server detach-volume <サーバー名またはID> --volume <ボリュームID>
```
```

- [ ] **Step 2: Verify build**

```bash
npx vitepress build docs
```

- [ ] **Step 3: Commit**

```bash
git add docs/reference/server.md
git commit -m "Add server command reference"
```

---

### Task 14: Reference — app

**Files:**
- Modify: `docs/reference/app.md`

- [ ] **Step 1: Write app reference**

Replace `docs/reference/app.md` with:

```md
# app

アプリケーションのデプロイと管理を行うコマンドグループです。

## app init

サーバー上にアプリの受け口を作成します。

### 使い方

```bash
conoha app init <サーバー名> [flags]
```

### オプション

| オプション | 説明 | 必須 |
|-----------|------|------|
| `--app-name` | アプリ名 | ○ |
| `--key` | SSH秘密鍵のパス | |

---

## app deploy

カレントディレクトリのプロジェクトをサーバーにデプロイします。

### 使い方

```bash
conoha app deploy <サーバー名> [flags]
```

### オプション

| オプション | 説明 | 必須 |
|-----------|------|------|
| `--app-name` | アプリ名 | ○ |
| `--key` | SSH秘密鍵のパス | |

### 動作

1. プロジェクトファイルをtarで圧縮（`.dockerignore` と `.git/` を除外）
2. サーバーにSSHで転送
3. `.env.server` があれば `.env` にコピー
4. `docker compose up -d --build` を実行

---

## app logs

アプリのコンテナログを表示します。

### 使い方

```bash
conoha app logs <サーバー名> [flags]
```

### オプション

| オプション | 説明 |
|-----------|------|
| `--app-name` | アプリ名 |
| `--follow` | リアルタイムでフォロー |
| `--tail` | 末尾の行数（デフォルト: 100） |
| `--key` | SSH秘密鍵のパス |

---

## app status

アプリのコンテナ状態を表示します。

### 使い方

```bash
conoha app status <サーバー名> [flags]
```

### オプション

| オプション | 説明 | 必須 |
|-----------|------|------|
| `--app-name` | アプリ名 | ○ |
| `--key` | SSH秘密鍵のパス | |

---

## app stop

アプリのコンテナを停止します。

### 使い方

```bash
conoha app stop <サーバー名> [flags]
```

### オプション

| オプション | 説明 | 必須 |
|-----------|------|------|
| `--app-name` | アプリ名 | ○ |
| `--key` | SSH秘密鍵のパス | |

---

## app restart

アプリのコンテナを再起動します。

### 使い方

```bash
conoha app restart <サーバー名> [flags]
```

### オプション

| オプション | 説明 | 必須 |
|-----------|------|------|
| `--app-name` | アプリ名 | ○ |
| `--key` | SSH秘密鍵のパス | |

---

## app env set

環境変数を設定します。

### 使い方

```bash
conoha app env set <サーバー名> --app-name <アプリ名> KEY=VALUE [KEY=VALUE...]
```

### 例

```bash
conoha app env set myserver --app-name myapp DATABASE_URL=postgres://... SECRET_KEY=abc123
```

::: tip
設定後、`app deploy` で再デプロイすると反映されます。
:::

---

## app env get

特定の環境変数の値を取得します。

### 使い方

```bash
conoha app env get <サーバー名> --app-name <アプリ名> KEY
```

---

## app env list

設定済みの環境変数一覧を表示します。

### 使い方

```bash
conoha app env list <サーバー名> --app-name <アプリ名>
```

---

## app env unset

環境変数を削除します。

### 使い方

```bash
conoha app env unset <サーバー名> --app-name <アプリ名> KEY [KEY...]
```

---

## app list

サーバー上のデプロイ済みアプリ一覧を表示します。

### 使い方

```bash
conoha app list <サーバー名> [flags]
```

### オプション

| オプション | 説明 |
|-----------|------|
| `--key` | SSH秘密鍵のパス |

---

## app destroy

アプリとそのデータをすべて削除します。

### 使い方

```bash
conoha app destroy <サーバー名> [flags]
```

### オプション

| オプション | 説明 | 必須 |
|-----------|------|------|
| `--app-name` | アプリ名 | ○ |
| `--key` | SSH秘密鍵のパス | |
| `--yes` | 確認をスキップ | |

### 削除されるもの

- コンテナ（停止・削除）
- 作業ディレクトリ（`/opt/conoha/{app-name}/`）
- Gitリポジトリ（`/opt/conoha/{app-name}.git/`）
- 環境変数ファイル（`/opt/conoha/{app-name}.env.server`）
```

- [ ] **Step 2: Verify build**

```bash
npx vitepress build docs
```

- [ ] **Step 3: Commit**

```bash
git add docs/reference/app.md
git commit -m "Add app command reference"
```

---

### Task 15: Push and verify deploy

- [ ] **Step 1: Push to main**

```bash
git push -u origin main
```

- [ ] **Step 2: Configure GitHub Pages**

In the GitHub repo settings (`crowdy/conoha-cli-pages`):
1. Settings → Pages → Source: "GitHub Actions"
2. Settings → Pages → Custom domain: `conoha-cli.jp`

- [ ] **Step 3: Configure DNS**

Add a CNAME record for `conoha-cli.jp` pointing to `crowdy.github.io`.

- [ ] **Step 4: Verify deployment**

Check GitHub Actions for the deploy workflow completion, then access `https://conoha-cli.jp` (or `https://crowdy.github.io/conoha-cli-pages` before DNS propagation).

- [ ] **Step 5: Commit link update to conoha-cli**

In the conoha-cli repo, update README.md to add a link to the docs site:

```bash
cd /home/tkim/dev/crowdy/conoha-cli
# Add documentation link to README files
```
