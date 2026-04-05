# note.com 記事デザイン: conoha-cli 紹介

## 概要

note.com に投稿する conoha-cli 紹介記事の設計。日本語 6000 字以上、技術ブログ風のハンズオン形式。CLI の手動操作と Claude Code Skill による自然言語操作の対比を軸に構成する。

## ターゲット読者

- Claude Code を使っている開発者（AI ツール活用層）
- ConoHa VPS 既存ユーザー
- VPS / インフラ構築に興味がある CLI 初心者

## 記事の形式

- テキスト + コード例のみ（画像なし）
- Markdown で原稿作成、note.com への投稿は手動
- 技術ブログ風: コード例多め、ハンズオン形式

## 構成: 「CLI → Skill 進化ストーリー」型

前半で CLI の基本操作とアプリデプロイを手動で見せ、後半で skill による自然言語操作を紹介する対比構成。

### セクション 1: 導入（〜600字）

- タイトル案: 「CLIひとつでVPSデプロイ完了 — conoha-cliとClaude Code Skillで変わるインフラ構築」
- VPS でアプリを動かすまでの手順の多さを共感ポイントとして提示
- conoha-cli + skill で解決できるという全体像
- 対象読者の明示

### セクション 2: conoha-cli とは（〜800字）

- Go 製 CLI、シングルバイナリ、macOS/Linux/Windows 対応
- 主な特徴: 複数プロファイル、構造化出力、自動トークンリフレッシュ、`--no-input`
- インストール手順（`brew install crowdy/tap/conoha` + `conoha auth login`）
- 「シングルバイナリひとつで VPS のライフサイクル全体を管理」という価値の訴求

### セクション 3: 基本操作 — サーバー作成から SSH まで（〜1000字）

- `conoha flavor list` / `conoha image list` でプラン・OS 確認
- `conoha server create` でサーバー作成（フラグ付きのコード例）
- `conoha server list` で状態確認
- `conoha server ssh` で接続
- 管理画面不要でターミナルから数コマンドで完結する手軽さを強調

### セクション 4: アプリデプロイ — docker-compose.yml があれば OK（〜1200字）

- `conoha app` コマンド群の紹介
- Express.js アプリを例にデプロイ全フロー:
  1. `conoha app init` — Docker 環境構築 + git リポジトリ作成
  2. `conoha app deploy` — tar アップロード → `docker compose up`
  3. `conoha app status` / `conoha app logs --follow`
- 運用系コマンド: `app env set`, `app restart`, `app destroy`
- `.dockerignore` 尊重、`.env.server` 永続化などの実用ディテール

### セクション 5: Claude Code Skill — 自然言語でインフラ構築（〜1500字）

- 記事のクライマックス
- `conoha skill install` で skill をインストール
- skill の概念説明（Claude Code のドメイン知識プラグイン）
- Claude Code との対話形式デモ:
  - 「ConoHa にサーバーを作って Express アプリをデプロイして」
  - skill が自動ロード → レシピに沿ってステップバイステップ実行
- 用意されているレシピ一覧:
  - Docker Compose アプリ / カスタムスクリプト / k3s / OpenStack / Slurm
- 手動操作との対比: 自然言語で同じ結果が得られるインパクト
- CLI 知識がなくても使える敷居の低さ + CLI を理解していれば動作が透明で安心

### セクション 6: まとめ（〜500字）

- CLI + skill の2段構えの振り返り
- 設計思想: シングルバイナリ、AI 親和性、docker-compose.yml ベース
- リンク集:
  - `brew install crowdy/tap/conoha`
  - `github.com/crowdy/conoha-cli`
  - `github.com/crowdy/conoha-cli-skill`

## 想定文字数

合計約 5600 字（骨格）。コード例・解説の充実で 6000 字以上に到達。

## 成果物

- `docs/articles/2026-04-05-note-conoha-cli.md` に Markdown 原稿を作成
