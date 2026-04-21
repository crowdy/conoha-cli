---
name: conoha-cli
description: >
  ConoHa VPS3 CLIによるインフラ構築スキル。サーバー作成、アプリデプロイ、
  Kubernetesクラスター、OpenStackプラットフォーム、Slurmクラスターの構築を支援。
  「ConoHaでサーバーを作って」「k8sクラスターを構築して」「アプリをデプロイして」
  などのリクエストでトリガー。
---

# ConoHa CLI スキル

ConoHa VPS3 CLIを使ったインフラ構築ガイド。

## 前提条件

- `conoha-cli` がインストール済みであること
- `conoha auth login` で認証済みであること
- SSHキーペアが登録済みであること（`conoha keypair create <name>`）

## 基本操作

### サーバー管理

| コマンド | 説明 |
|---------|------|
| `conoha server list` | サーバー一覧を表示する |
| `conoha server create --name <名前> --flavor <ID> --image <ID> --key-name <キー名>` | サーバーを作成する |
| `conoha server create --name <名前> --wait` | サーバー作成完了まで待機する |
| `conoha server show <ID\|名前>` | サーバー詳細を表示する |
| `conoha server delete <ID\|名前>` | サーバーを削除する |
| `conoha server deploy <ID\|名前> --script <ファイル> --env KEY=VALUE` | スクリプトをSSH経由で実行する |

### フレーバー・イメージ

| コマンド | 説明 |
|---------|------|
| `conoha flavor list` | 利用可能なフレーバー一覧を表示する |
| `conoha image list` | 利用可能なイメージ一覧を表示する |

### ネットワーク

| コマンド | 説明 |
|---------|------|
| `conoha network list` | ネットワーク一覧を表示する |
| `conoha network create --name <名前>` | プライベートネットワークを作成する |
| `conoha network subnet create --network-id <ID> --cidr <CIDR>` | サブネットを作成する |
| `conoha network sg list` | セキュリティグループ一覧を表示する |
| `conoha network sg create --name <名前>` | セキュリティグループを作成する |
| `conoha network sgr create --security-group-id <ID> --direction ingress --protocol tcp --port-min <ポート> --port-max <ポート> --remote-ip <CIDR>` | セキュリティグループルールを追加する |

### キーペア

| コマンド | 説明 |
|---------|------|
| `conoha keypair list` | キーペア一覧を表示する |
| `conoha keypair create <名前>` | キーペアを作成する |

### アプリデプロイ（conoha-proxy 経由 blue/green）

v0.2.0 から `conoha app deploy` は [conoha-proxy](https://github.com/crowdy/conoha-proxy) 経由の blue/green デプロイに統一。Let's Encrypt TLS と Host ヘッダールーティングを含む。初回セットアップは「proxy boot → app init → app deploy」の 3 ステップ。

前提：レポルートに `conoha.yml`（name / hosts / web.service / web.port を宣言）。

| コマンド | 説明 |
|---------|------|
| `conoha proxy boot <ID\|名前> --acme-email <mail>` | conoha-proxy コンテナをサーバーに起動する |
| `conoha proxy reboot <ID\|名前> --acme-email <mail>` | プロキシイメージを更新して再起動する |
| `conoha proxy logs <ID\|名前> --follow` | プロキシのログを見る |
| `conoha proxy details <ID\|名前>` | プロキシのバージョン・readiness を表示する |
| `conoha proxy services <ID\|名前>` | 登録されている service 一覧を表示する |
| `conoha app init <ID\|名前>` | `conoha.yml` を proxy に service として登録する |
| `conoha app deploy <ID\|名前>` | 新 slot を立ち上げて proxy に probe + swap を依頼（blue/green） |
| `conoha app rollback <ID\|名前>` | drain 窓内で直前の slot にスワップバックする |
| `conoha app status <ID\|名前>` | compose ps + proxy の phase / active / draining を表示する |
| `conoha app logs <ID\|名前> --follow` | アプリのログをストリーミング表示する |
| `conoha app stop <ID\|名前>` | アプリのコンテナを停止する |
| `conoha app restart <ID\|名前>` | アプリのコンテナを再起動する |
| `conoha app destroy <ID\|名前>` | 全 slot を停止して proxy から service を削除する |

`.env.server` の代わりに blue/green スロットごとに `conoha-override.yml` が注入され、host ポートは kernel 割当（`127.0.0.1:0:<web.port>`）。compose ファイルに host 側ポートをハードコードしないこと。

## レシピ一覧

ユーザーのリクエストに応じて、該当するレシピファイルを読み込んで手順を実行する。

| レシピ | 用途 | ファイル |
|--------|------|---------|
| シングルサーバーアプリ | Docker Composeアプリのデプロイ | [recipes/single-server-app.md](recipes/single-server-app.md) |
| シングルサーバースクリプト | カスタムスクリプトによるセットアップ | [recipes/single-server-script.md](recipes/single-server-script.md) |
| Kubernetesクラスター | k3sによるマルチノードk8sクラスター | [recipes/k8s-cluster.md](recipes/k8s-cluster.md) |
| OpenStackプラットフォーム | DevStackによるOpenStack環境 | [recipes/openstack-platform.md](recipes/openstack-platform.md) |
| Slurmクラスター | HPCジョブスケジューラクラスター | [recipes/slurm-cluster.md](recipes/slurm-cluster.md) |

## 共通パターン

### マルチサーバー作成

複数サーバーを作成する場合、命名規則を統一する：

```bash
# 例: k8sクラスター
conoha server create --name k8s-master-1 --flavor <ID> --image <ID> --key-name <キー名> --security-group <SG名> --wait
conoha server create --name k8s-worker-1 --flavor <ID> --image <ID> --key-name <キー名> --security-group <SG名> --wait
conoha server create --name k8s-worker-2 --flavor <ID> --image <ID> --key-name <キー名> --security-group <SG名> --wait
```

### セキュリティグループの事前準備

クラスター構築前に必要なポートを開放するセキュリティグループを作成する：

```bash
# セキュリティグループ作成
conoha network sg create --name <クラスター名>-sg

# 必要なルールを追加（例: SSH + クラスター固有ポート）
conoha network sgr create --security-group-id <SG-ID> --direction ingress --protocol tcp --port-min 22 --port-max 22 --remote-ip 0.0.0.0/0
```

### デプロイスクリプト実行順序

マルチサーバー構成では以下の順序で実行する：

1. 全ノード共通の初期設定スクリプトを実行する
2. マスター/コントローラーノードのセットアップスクリプトを実行する
3. マスターからジョイン情報（トークン等）を取得する
4. ワーカー/コンピュートノードのセットアップスクリプトを実行する（ジョイン情報を`--env`で渡す）

```bash
# 1. 全ノード共通
conoha server deploy <各ノード> --script common-setup.sh

# 2. マスターセットアップ
conoha server deploy <マスター> --script master-setup.sh

# 3. ジョイン情報取得（SSH経由）
TOKEN=$(ssh root@<マスターIP> "取得コマンド")

# 4. ワーカーセットアップ
conoha server deploy <ワーカー> --script worker-setup.sh --env JOIN_TOKEN=$TOKEN
```

## 問題報告

conoha-cli の使用中に異常な動作や予期しないエラーを発見した場合、`gh` コマンドが利用可能であれば GitHub Issue を自動登録する：

```bash
gh issue create --repo crowdy/conoha-cli --title "<エラーの要約>" --body "<再現手順と詳細>"
```

- 再現手順、実行したコマンド、エラーメッセージを含める
- 可能であれば conoha-cli のバージョン（`conoha version`）も記載する
