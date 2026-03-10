# conoha - ConoHa VPS3 CLI

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[English](README-en.md) | [한국어](README-ko.md)

ConoHa VPS3 API 用のコマンドラインインターフェースです。Go で書かれたシングルバイナリで、エージェントフレンドリーな設計を採用しています。

> **注意**: 本ツールは VPS3 専用です。旧 VPS2 用の CLI（hironobu-s/conoha-vps、miyabisun/conoha-cli）とは互換性がありません。

## 特徴

- シングルバイナリ、クロスプラットフォーム対応（Linux / macOS / Windows）
- 複数プロファイル対応（`gh auth` スタイル）
- 構造化出力（`--format json/yaml/csv/table`）
- エージェントフレンドリー設計（`--no-input`、決定的な終了コード、stderr/stdout 分離）
- トークン自動更新（有効期限 5 分前に再認証）

## インストール

### ソースからビルド

```bash
go install github.com/crowdy/conoha-cli@latest
```

### リリースバイナリ

[Releases](https://github.com/crowdy/conoha-cli/releases) ページからダウンロードしてください。

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
| `conoha server` | サーバー管理（list / show / create / delete / start / stop / reboot / resize / rebuild / rename / console） |
| `conoha flavor` | フレーバー一覧・詳細（list / show） |
| `conoha keypair` | SSH キーペア管理（list / create / delete） |
| `conoha volume` | ブロックストレージ管理（list / show / create / delete / types / backup） |
| `conoha image` | イメージ管理（list / show / delete） |
| `conoha network` | ネットワーク管理（network / subnet / port / security-group / qos） |
| `conoha lb` | ロードバランサー管理（lb / listener / pool / member / healthmonitor） |
| `conoha dns` | DNS 管理（domain / record） |
| `conoha storage` | オブジェクトストレージ（container / ls / cp / rm / publish） |
| `conoha identity` | アイデンティティ管理（credential / subuser / role） |
| `conoha config` | CLI 設定管理（show / set / path） |

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
| `CONOHA_DEBUG` | デバッグログ（`1` or `api`） |

優先順位: 環境変数 > フラグ > プロファイル設定 > デフォルト値

### グローバルフラグ

```
--profile    使用するプロファイル
--format     出力形式（table / json / yaml / csv）
--no-input   対話プロンプトを無効化
--quiet      不要な出力を抑制
--verbose    詳細出力
--no-color   カラー出力を無効化
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

MIT License - 詳細は [LICENSE](LICENSE) をご覧ください。
