# conoha-cli-skill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create the `crowdy/conoha-cli-skill` repository with SKILL.md and 5 recipe files for ConoHa VPS3 infrastructure orchestration.

**Architecture:** Single skill repository with SKILL.md as entry point + progressive disclosure via recipe files in `recipes/` directory. All content in Japanese, imperative form.

**Tech Stack:** Markdown, Git, GitHub CLI (`gh`)

**Spec:** `docs/superpowers/specs/2026-04-01-conoha-cli-skill-design.md`

---

### Task 1: Create GitHub Repository

**Files:**
- Create: `crowdy/conoha-cli-skill` (GitHub repo)

- [ ] **Step 1: Create the repository**

```bash
gh repo create crowdy/conoha-cli-skill --public --description "Claude Code skill for ConoHa VPS3 CLI infrastructure orchestration" --clone
```

- [ ] **Step 2: Verify**

```bash
cd conoha-cli-skill && git status
```

Expected: empty repo, on main branch.

---

### Task 2: Write SKILL.md

**Files:**
- Create: `SKILL.md`

- [ ] **Step 1: Create SKILL.md**

Write the following content to `SKILL.md`:

```markdown
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

### アプリデプロイ

| コマンド | 説明 |
|---------|------|
| `conoha app init <ID\|名前>` | サーバーにDocker環境を初期化する |
| `conoha app deploy <ID\|名前>` | カレントディレクトリをサーバーにデプロイする |
| `conoha app status <ID\|名前>` | アプリのコンテナ状態を表示する |
| `conoha app logs <ID\|名前> --follow` | アプリのログをストリーミング表示する |
| `conoha app stop <ID\|名前>` | アプリのコンテナを停止する |
| `conoha app restart <ID\|名前>` | アプリのコンテナを再起動する |

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
```

- [ ] **Step 2: Commit**

```bash
git add SKILL.md && git commit -m "Add SKILL.md with command reference and recipe index"
```

---

### Task 3: Write single-server-app.md

**Files:**
- Create: `recipes/single-server-app.md`

- [ ] **Step 1: Create recipes directory and file**

Write the following to `recipes/single-server-app.md`:

```markdown
# シングルサーバーアプリデプロイ

## 概要

Docker Composeアプリをサーバー1台にデプロイするレシピ。`conoha app` コマンドを使用し、カレントディレクトリのDocker Composeプロジェクトをサーバーに転送・起動する。

## 基本構成

- **ノード数**: 1
- **OS**: Ubuntu
- **必須**: Docker Compose対応の `docker-compose.yml` がカレントディレクトリにあること

```
[ローカルPC] -- tar+SSH --> [ConoHa VPS]
                             └── /opt/conoha/<app-name>/
                                 ├── docker-compose.yml
                                 ├── Dockerfile
                                 ├── .env (← .env.server からコピー)
                                 └── (ソースコード)
```

## 手順

### 1. 事前準備

フレーバーとイメージを確認する：

```bash
conoha flavor list
conoha image list
```

キーペアが未作成の場合は作成する：

```bash
conoha keypair create my-key
```

### 2. サーバー作成

```bash
conoha server create --name my-app-server --flavor <フレーバーID> --image <UbuntuイメージID> --key-name my-key --wait
```

`--wait` を付けてサーバーがACTIVEになるまで待機する。

### 3. アプリ初期化

サーバーにDocker環境をセットアップする：

```bash
conoha app init my-app-server
```

これにより以下が実行される：
- Docker/Docker Composeのインストール
- git bare リポジトリの作成（post-receive hook付き）

### 4. アプリデプロイ

カレントディレクトリ（`docker-compose.yml` があるディレクトリ）で実行する：

```bash
conoha app deploy my-app-server
```

これにより以下が実行される：
- カレントディレクトリのtar.gzアーカイブ作成（`.git/` 除外）
- SSH経由でアップロード
- `/opt/conoha/<app-name>/` に展開
- `.env.server` → `.env` コピー（存在する場合）
- `docker compose up -d --build --remove-orphans` 実行

### 5. 動作確認

```bash
# コンテナ状態を確認する
conoha app status my-app-server

# ログを確認する
conoha app logs my-app-server --follow

# 直接SSHでアクセスする場合
ssh root@<サーバーIP> "curl -s localhost:<ポート>"
```

## カスタマイズ

### 環境変数

サーバー側に永続的な環境変数を設定する（デプロイを跨いで維持される）：

```bash
conoha app env set my-app-server DATABASE_URL=postgres://...
conoha app env list my-app-server
```

または、ローカルに `.env.server` ファイルを作成してデプロイ時に自動コピーさせる。

### アプリの管理

```bash
# 停止
conoha app stop my-app-server

# 再起動
conoha app restart my-app-server

# ログ（特定サービス）
conoha app logs my-app-server --service web
```

## トラブルシューティング

| 問題 | 対処 |
|------|------|
| `docker compose up` が失敗する | `conoha app logs my-app-server` でエラーを確認する |
| ポートにアクセスできない | セキュリティグループで該当ポートが開放されているか確認する |
| デプロイが遅い | `.dockerignore` でnode_modules等を除外する |
```

- [ ] **Step 2: Commit**

```bash
git add recipes/single-server-app.md && git commit -m "Add single-server-app recipe"
```

---

### Task 4: Write single-server-script.md

**Files:**
- Create: `recipes/single-server-script.md`

- [ ] **Step 1: Create file**

Write the following to `recipes/single-server-script.md`:

```markdown
# シングルサーバースクリプトデプロイ

## 概要

カスタムスクリプトをサーバー1台にSSH経由で実行するレシピ。Docker Composeを使わず、任意のセットアップを行う場合に使用する。

## 基本構成

- **ノード数**: 1
- **OS**: Ubuntu
- **方式**: `conoha server deploy` でローカルスクリプトをアップロード・実行

```
[ローカルPC] -- SSH --> [ConoHa VPS]
                         └── スクリプトを実行
                             ├── パッケージインストール
                             ├── 設定ファイル配置
                             └── サービス起動
```

## 手順

### 1. 事前準備

フレーバーとイメージを確認する：

```bash
conoha flavor list
conoha image list
```

### 2. サーバー作成

```bash
conoha server create --name my-server --flavor <フレーバーID> --image <UbuntuイメージID> --key-name my-key --wait
```

### 3. デプロイスクリプトの作成

ローカルにスクリプトファイルを作成する。例：Nginx + Node.jsのセットアップ：

```bash
#!/bin/bash
set -euo pipefail

# パッケージ更新
apt-get update && apt-get upgrade -y

# Nginxインストール
apt-get install -y nginx

# Node.jsインストール（LTS）
curl -fsSL https://deb.nodesource.com/setup_lts.x | bash -
apt-get install -y nodejs

# Nginx設定（リバースプロキシ）
cat > /etc/nginx/sites-available/app <<'NGINX'
server {
    listen 80;
    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
NGINX

ln -sf /etc/nginx/sites-available/app /etc/nginx/sites-enabled/default
systemctl restart nginx

echo "Setup complete"
```

### 4. スクリプト実行

```bash
conoha server deploy my-server --script setup.sh
```

環境変数を渡す場合：

```bash
conoha server deploy my-server --script setup.sh --env DB_HOST=localhost --env DB_PORT=5432
```

スクリプト内で環境変数を参照できる：`$DB_HOST`, `$DB_PORT`

### 5. 動作確認

```bash
# SSH接続して状態を確認する
ssh root@<サーバーIP> "systemctl status nginx"
ssh root@<サーバーIP> "curl -s localhost"
```

## カスタマイズ

### データベースセットアップ

PostgreSQLを追加する場合のスクリプト例：

```bash
#!/bin/bash
set -euo pipefail
apt-get update
apt-get install -y postgresql postgresql-contrib
systemctl enable postgresql
sudo -u postgres createuser --superuser $APP_USER
sudo -u postgres createdb $APP_DB -O $APP_USER
```

```bash
conoha server deploy my-server --script setup-db.sh --env APP_USER=myapp --env APP_DB=myapp_db
```

### 複数スクリプトの段階実行

複雑なセットアップは複数スクリプトに分割して実行する：

```bash
conoha server deploy my-server --script 01-base-setup.sh
conoha server deploy my-server --script 02-app-setup.sh --env APP_PORT=3000
conoha server deploy my-server --script 03-nginx-setup.sh --env APP_PORT=3000
```

## トラブルシューティング

| 問題 | 対処 |
|------|------|
| スクリプトが途中で失敗する | `set -euo pipefail` を先頭に入れて失敗箇所を特定する |
| SSH接続できない | サーバーがACTIVE状態か確認する（`conoha server show <名前>`） |
| 環境変数が展開されない | `--env KEY=VALUE` の形式を確認する（`=` の前後にスペースを入れない） |
```

- [ ] **Step 2: Commit**

```bash
git add recipes/single-server-script.md && git commit -m "Add single-server-script recipe"
```

---

### Task 5: Write k8s-cluster.md

**Files:**
- Create: `recipes/k8s-cluster.md`

- [ ] **Step 1: Create file**

Write the following to `recipes/k8s-cluster.md`:

```markdown
# Kubernetesクラスター構築

## 概要

ConoHa VPS上にk3sを使ったKubernetesクラスターを構築するレシピ。マスターノード1台 + ワーカーノード2台の3ノード構成。

## 基本構成

- **ノード数**: 3（マスター1 + ワーカー2）
- **OS**: Ubuntu
- **k8sディストリビューション**: k3s
- **推奨フレーバー**: 2vCPU / 2GB RAM 以上

```
                    ┌─────────────┐
                    │ k8s-master-1│
                    │   (k3s      │
                    │   server)   │
                    └──────┬──────┘
                           │ K3S_TOKEN
              ┌────────────┼────────────┐
              │                         │
       ┌──────┴──────┐          ┌──────┴──────┐
       │k8s-worker-1 │          │k8s-worker-2 │
       │  (k3s agent)│          │  (k3s agent) │
       └─────────────┘          └──────────────┘
```

## 手順

### 1. 事前準備

セキュリティグループを作成し、必要なポートを開放する：

```bash
# セキュリティグループ作成
conoha network sg create --name k8s-sg

# SSH
conoha network sgr create --security-group-id <SG-ID> --direction ingress --protocol tcp --port-min 22 --port-max 22 --remote-ip 0.0.0.0/0

# Kubernetes API Server
conoha network sgr create --security-group-id <SG-ID> --direction ingress --protocol tcp --port-min 6443 --port-max 6443 --remote-ip 0.0.0.0/0

# kubelet
conoha network sgr create --security-group-id <SG-ID> --direction ingress --protocol tcp --port-min 10250 --port-max 10250 --remote-ip 0.0.0.0/0

# NodePort range
conoha network sgr create --security-group-id <SG-ID> --direction ingress --protocol tcp --port-min 30000 --port-max 32767 --remote-ip 0.0.0.0/0
```

### 2. サーバー作成

3台のサーバーを作成する：

```bash
conoha server create --name k8s-master-1 --flavor <フレーバーID> --image <UbuntuイメージID> --key-name my-key --security-group k8s-sg --wait
conoha server create --name k8s-worker-1 --flavor <フレーバーID> --image <UbuntuイメージID> --key-name my-key --security-group k8s-sg --wait
conoha server create --name k8s-worker-2 --flavor <フレーバーID> --image <UbuntuイメージID> --key-name my-key --security-group k8s-sg --wait
```

サーバーのIPを確認する：

```bash
conoha server show k8s-master-1 -o json
conoha server show k8s-worker-1 -o json
conoha server show k8s-worker-2 -o json
```

### 3. マスターノードセットアップ

k3sサーバーをインストールするスクリプト `k8s-master-setup.sh` を作成する：

```bash
#!/bin/bash
set -euo pipefail

# k3sサーバーインストール
curl -sfL https://get.k3s.io | sh -s - server \
  --write-kubeconfig-mode 644 \
  --tls-san "$MASTER_IP"

# インストール完了待機
until kubectl get nodes; do
  sleep 2
done

echo "k3s server installed successfully"
```

実行する：

```bash
MASTER_IP=$(conoha server show k8s-master-1 -o json | jq -r '.addresses | to_entries[0].value[] | select(.version == 4) | .addr')
conoha server deploy k8s-master-1 --script k8s-master-setup.sh --env MASTER_IP=$MASTER_IP
```

### 4. ジョイントークン取得

マスターからワーカー用のトークンを取得する：

```bash
K3S_TOKEN=$(ssh root@$MASTER_IP "cat /var/lib/rancher/k3s/server/node-token")
```

### 5. ワーカーノードセットアップ

k3sエージェントをインストールするスクリプト `k8s-worker-setup.sh` を作成する：

```bash
#!/bin/bash
set -euo pipefail

# k3sエージェントインストール
curl -sfL https://get.k3s.io | K3S_URL="https://${MASTER_IP}:6443" K3S_TOKEN="$JOIN_TOKEN" sh -s - agent

echo "k3s agent installed successfully"
```

各ワーカーで実行する：

```bash
conoha server deploy k8s-worker-1 --script k8s-worker-setup.sh --env MASTER_IP=$MASTER_IP --env JOIN_TOKEN=$K3S_TOKEN
conoha server deploy k8s-worker-2 --script k8s-worker-setup.sh --env MASTER_IP=$MASTER_IP --env JOIN_TOKEN=$K3S_TOKEN
```

### 6. 動作確認

```bash
ssh root@$MASTER_IP "kubectl get nodes"
```

期待される出力：

```
NAME           STATUS   ROLES                  AGE   VERSION
k8s-master-1   Ready    control-plane,master   5m    v1.xx.x+k3s1
k8s-worker-1   Ready    <none>                 2m    v1.xx.x+k3s1
k8s-worker-2   Ready    <none>                 2m    v1.xx.x+k3s1
```

kubeconfigをローカルにコピーする：

```bash
scp root@$MASTER_IP:/etc/rancher/k3s/k3s.yaml ~/.kube/config
sed -i "s/127.0.0.1/$MASTER_IP/" ~/.kube/config
kubectl get nodes
```

## カスタマイズ

### ワーカーノード数の変更

ワーカーを追加する場合、手順2で追加のサーバーを作成し、手順5を繰り返す。

### kubeadmを使用する場合

k3sの代わりにkubeadmを使用する場合、マスターセットアップスクリプトを以下に変更する：

```bash
#!/bin/bash
set -euo pipefail

# コンテナランタイムとkubeadmインストール
apt-get update
apt-get install -y apt-transport-https ca-certificates curl
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.31/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.31/deb/ /' > /etc/apt/sources.list.d/kubernetes.list
apt-get update
apt-get install -y kubelet kubeadm kubectl containerd
apt-mark hold kubelet kubeadm kubectl

# kubeadm init
kubeadm init --pod-network-cidr=10.244.0.0/16 --apiserver-advertise-address=$MASTER_IP

# kubeconfig設定
mkdir -p /root/.kube
cp /etc/kubernetes/admin.conf /root/.kube/config

# Flannelインストール
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml
```

ジョイントークンの取得コマンドも変更する：

```bash
JOIN_CMD=$(ssh root@$MASTER_IP "kubeadm token create --print-join-command")
```

## トラブルシューティング

| 問題 | 対処 |
|------|------|
| ワーカーがNotReady | `kubectl describe node <名前>` でイベントを確認する |
| k3sインストールが失敗する | サーバーのメモリが2GB以上あるか確認する（`conoha flavor show <ID>`） |
| API Serverに接続できない | セキュリティグループで6443ポートが開放されているか確認する |
| ノード間通信ができない | 全ノードが同じセキュリティグループに属しているか確認する |
```

- [ ] **Step 2: Commit**

```bash
git add recipes/k8s-cluster.md && git commit -m "Add Kubernetes cluster recipe"
```

---

### Task 6: Write openstack-platform.md

**Files:**
- Create: `recipes/openstack-platform.md`

- [ ] **Step 1: Create file**

Write the following to `recipes/openstack-platform.md`:

```markdown
# OpenStackプラットフォーム構築

## 概要

ConoHa VPS上にDevStackを使ったOpenStack環境を構築するレシピ。まずオールインワン（1台）構成で構築し、必要に応じてマルチノードに拡張する。

## 基本構成

- **ノード数**: 1（オールインワン）
- **OS**: Ubuntu
- **OpenStackディストリビューション**: DevStack
- **推奨フレーバー**: 4vCPU / 8GB RAM 以上（OpenStackは多くのリソースを消費する）

```
┌──────────────────────────────┐
│        openstack-aio         │
│  ┌────────┐  ┌────────────┐  │
│  │Keystone│  │  Horizon   │  │
│  ├────────┤  ├────────────┤  │
│  │  Nova  │  │  Neutron   │  │
│  ├────────┤  ├────────────┤  │
│  │ Glance │  │  Cinder    │  │
│  └────────┘  └────────────┘  │
└──────────────────────────────┘
```

## 手順

### 1. 事前準備

セキュリティグループを作成する：

```bash
conoha network sg create --name openstack-sg

# SSH
conoha network sgr create --security-group-id <SG-ID> --direction ingress --protocol tcp --port-min 22 --port-max 22 --remote-ip 0.0.0.0/0

# Horizon (Dashboard)
conoha network sgr create --security-group-id <SG-ID> --direction ingress --protocol tcp --port-min 80 --port-max 80 --remote-ip 0.0.0.0/0

# Keystone (Identity API)
conoha network sgr create --security-group-id <SG-ID> --direction ingress --protocol tcp --port-min 5000 --port-max 5000 --remote-ip 0.0.0.0/0

# Nova (Compute API)
conoha network sgr create --security-group-id <SG-ID> --direction ingress --protocol tcp --port-min 8774 --port-max 8774 --remote-ip 0.0.0.0/0

# VNC Console
conoha network sgr create --security-group-id <SG-ID> --direction ingress --protocol tcp --port-min 6080 --port-max 6080 --remote-ip 0.0.0.0/0
```

### 2. サーバー作成

大きなフレーバーを選択する（8GB RAM以上推奨）：

```bash
conoha flavor list
conoha server create --name openstack-aio --flavor <大型フレーバーID> --image <UbuntuイメージID> --key-name my-key --security-group openstack-sg --wait
```

### 3. DevStackインストール

セットアップスクリプト `openstack-setup.sh` を作成する：

```bash
#!/bin/bash
set -euo pipefail

# stackユーザー作成
useradd -s /bin/bash -d /opt/stack -m stack
chmod +x /opt/stack
echo "stack ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/stack

# DevStackクローン
su - stack -c "git clone https://opendev.org/openstack/devstack /opt/stack/devstack"

# local.conf作成
cat > /opt/stack/devstack/local.conf <<EOF
[[local|localrc]]
ADMIN_PASSWORD=$ADMIN_PASSWORD
DATABASE_PASSWORD=$ADMIN_PASSWORD
RABBIT_PASSWORD=$ADMIN_PASSWORD
SERVICE_PASSWORD=$ADMIN_PASSWORD

HOST_IP=$HOST_IP

# 有効化するサービス
enable_service placement-api
enable_service n-cpu
enable_service n-api
enable_service n-cond
enable_service n-sch
enable_service g-api
enable_service c-sch
enable_service c-api
enable_service c-vol

# Neutron
enable_service q-svc
enable_service q-agt
enable_service q-dhcp
enable_service q-l3
enable_service q-meta

# Horizon
enable_service horizon

LOGFILE=/opt/stack/logs/stack.sh.log
LOGDAYS=1
EOF

chown stack:stack /opt/stack/devstack/local.conf

# DevStack実行（時間がかかる: 20〜40分）
su - stack -c "cd /opt/stack/devstack && ./stack.sh"

echo "DevStack installation complete"
```

実行する：

```bash
HOST_IP=$(conoha server show openstack-aio -o json | jq -r '.addresses | to_entries[0].value[] | select(.version == 4) | .addr')
conoha server deploy openstack-aio --script openstack-setup.sh --env ADMIN_PASSWORD=SecurePass123 --env HOST_IP=$HOST_IP
```

注意: DevStackのインストールには20〜40分かかる。`conoha server deploy` のタイムアウトに注意する。

### 4. 動作確認

```bash
# Horizonダッシュボードにアクセスする
echo "http://$HOST_IP/dashboard"
# ユーザー: admin, パスワード: 上で設定したADMIN_PASSWORD

# OpenStack CLIで確認する
ssh root@$HOST_IP "su - stack -c 'source /opt/stack/devstack/openrc admin admin && openstack service list'"
```

## カスタマイズ

### マルチノード構成

コントローラー1台 + コンピュート2台に拡張する場合：

1. 追加のサーバーを作成する：

```bash
conoha server create --name openstack-compute-1 --flavor <フレーバーID> --image <UbuntuイメージID> --key-name my-key --security-group openstack-sg --wait
conoha server create --name openstack-compute-2 --flavor <フレーバーID> --image <UbuntuイメージID> --key-name my-key --security-group openstack-sg --wait
```

2. コンピュートノード用の `local.conf` でコントローラーのIPを `SERVICE_HOST` に設定する。

### 有効化サービスの変更

`local.conf` の `enable_service` / `disable_service` で調整する。例：Swiftを追加する場合：

```bash
enable_service s-proxy s-object s-container s-account
```

## トラブルシューティング

| 問題 | 対処 |
|------|------|
| `stack.sh` が途中で失敗する | `/opt/stack/logs/stack.sh.log` を確認する |
| メモリ不足 | 最低8GB RAMのフレーバーを使用する |
| Horizonにアクセスできない | セキュリティグループで80ポートを確認する |
| DevStack再実行 | `su - stack -c "cd /opt/stack/devstack && ./unstack.sh && ./stack.sh"` |
```

- [ ] **Step 2: Commit**

```bash
git add recipes/openstack-platform.md && git commit -m "Add OpenStack platform recipe"
```

---

### Task 7: Write slurm-cluster.md

**Files:**
- Create: `recipes/slurm-cluster.md`

- [ ] **Step 1: Create file**

Write the following to `recipes/slurm-cluster.md`:

```markdown
# Slurmクラスター構築

## 概要

ConoHa VPS上にSlurmジョブスケジューラを使ったHPCクラスターを構築するレシピ。コントローラーノード1台 + コンピュートノード2台の構成。

## 基本構成

- **ノード数**: 3（コントローラー1 + コンピュート2）
- **OS**: Ubuntu
- **ジョブスケジューラ**: Slurm
- **共有ストレージ**: NFS（コントローラーがエクスポート）

```
                    ┌──────────────────┐
                    │ slurm-controller │
                    │  ┌────────────┐  │
                    │  │  slurmctld │  │
                    │  │  slurmdbd  │  │
                    │  │  NFS server│  │
                    │  └────────────┘  │
                    │  /shared (NFS)   │
                    └────────┬─────────┘
                             │
                ┌────────────┼────────────┐
                │                         │
       ┌────────┴────────┐       ┌────────┴────────┐
       │ slurm-compute-1 │       │ slurm-compute-2 │
       │  ┌────────────┐ │       │  ┌────────────┐ │
       │  │   slurmd   │ │       │  │   slurmd   │ │
       │  │ NFS client │ │       │  │ NFS client │ │
       │  └────────────┘ │       │  └────────────┘ │
       │  /shared (mount)│       │  /shared (mount)│
       └─────────────────┘       └─────────────────┘
```

## 手順

### 1. 事前準備

セキュリティグループを作成する：

```bash
conoha network sg create --name slurm-sg

# SSH
conoha network sgr create --security-group-id <SG-ID> --direction ingress --protocol tcp --port-min 22 --port-max 22 --remote-ip 0.0.0.0/0

# Slurm通信ポート
conoha network sgr create --security-group-id <SG-ID> --direction ingress --protocol tcp --port-min 6817 --port-max 6819 --remote-ip 0.0.0.0/0

# NFS
conoha network sgr create --security-group-id <SG-ID> --direction ingress --protocol tcp --port-min 2049 --port-max 2049 --remote-ip 0.0.0.0/0

# RPC (NFS関連)
conoha network sgr create --security-group-id <SG-ID> --direction ingress --protocol tcp --port-min 111 --port-max 111 --remote-ip 0.0.0.0/0
```

### 2. サーバー作成

```bash
conoha server create --name slurm-controller --flavor <フレーバーID> --image <UbuntuイメージID> --key-name my-key --security-group slurm-sg --wait
conoha server create --name slurm-compute-1 --flavor <フレーバーID> --image <UbuntuイメージID> --key-name my-key --security-group slurm-sg --wait
conoha server create --name slurm-compute-2 --flavor <フレーバーID> --image <UbuntuイメージID> --key-name my-key --security-group slurm-sg --wait
```

各サーバーのIPを確認する：

```bash
CTRL_IP=$(conoha server show slurm-controller -o json | jq -r '.addresses | to_entries[0].value[] | select(.version == 4) | .addr')
COMP1_IP=$(conoha server show slurm-compute-1 -o json | jq -r '.addresses | to_entries[0].value[] | select(.version == 4) | .addr')
COMP2_IP=$(conoha server show slurm-compute-2 -o json | jq -r '.addresses | to_entries[0].value[] | select(.version == 4) | .addr')
```

### 3. 全ノード共通セットアップ

共通セットアップスクリプト `slurm-common.sh` を作成する：

```bash
#!/bin/bash
set -euo pipefail

# /etc/hostsにエントリ追加
cat >> /etc/hosts <<EOF
$CTRL_IP slurm-controller
$COMP1_IP slurm-compute-1
$COMP2_IP slurm-compute-2
EOF

# Slurmインストール
apt-get update
apt-get install -y slurm-wlm slurm-client munge

# Mungeキー配置（コントローラーから取得、または新規生成）
if [ "$NODE_ROLE" = "controller" ]; then
  create-munge-key
else
  echo "$MUNGE_KEY_BASE64" | base64 -d > /etc/munge/munge.key
fi
chmod 400 /etc/munge/munge.key
chown munge:munge /etc/munge/munge.key
systemctl enable munge
systemctl restart munge

echo "Common setup complete for $NODE_ROLE"
```

全ノードで実行する：

```bash
conoha server deploy slurm-controller --script slurm-common.sh --env CTRL_IP=$CTRL_IP --env COMP1_IP=$COMP1_IP --env COMP2_IP=$COMP2_IP --env NODE_ROLE=controller
```

コントローラーからMungeキーを取得する：

```bash
MUNGE_KEY=$(ssh root@$CTRL_IP "base64 /etc/munge/munge.key")
```

コンピュートノードで共通セットアップを実行する：

```bash
conoha server deploy slurm-compute-1 --script slurm-common.sh --env CTRL_IP=$CTRL_IP --env COMP1_IP=$COMP1_IP --env COMP2_IP=$COMP2_IP --env NODE_ROLE=compute --env MUNGE_KEY_BASE64="$MUNGE_KEY"
conoha server deploy slurm-compute-2 --script slurm-common.sh --env CTRL_IP=$CTRL_IP --env COMP1_IP=$COMP1_IP --env COMP2_IP=$COMP2_IP --env NODE_ROLE=compute --env MUNGE_KEY_BASE64="$MUNGE_KEY"
```

### 4. コントローラーセットアップ

コントローラーセットアップスクリプト `slurm-controller-setup.sh` を作成する：

```bash
#!/bin/bash
set -euo pipefail

# NFSサーバーセットアップ
apt-get install -y nfs-kernel-server
mkdir -p /shared
chmod 777 /shared
echo "/shared *(rw,sync,no_subtree_check,no_root_squash)" > /etc/exports
exportfs -ra
systemctl enable nfs-kernel-server
systemctl restart nfs-kernel-server

# Slurm設定ファイル作成
cat > /etc/slurm/slurm.conf <<EOF
ClusterName=conoha-cluster
SlurmctldHost=slurm-controller

MpiDefault=none
ProctrackType=proctrack/linuxproc
ReturnToService=1
SlurmctldPidFile=/run/slurmctld.pid
SlurmdPidFile=/run/slurmd.pid
SlurmdSpoolDir=/var/spool/slurmd
SlurmUser=slurm
StateSaveLocation=/var/spool/slurmctld

SchedulerType=sched/backfill
SelectType=select/cons_tres

# ノード定義
NodeName=slurm-compute-1 NodeAddr=$COMP1_IP CPUs=$CPUS RealMemory=$MEMORY State=UNKNOWN
NodeName=slurm-compute-2 NodeAddr=$COMP2_IP CPUs=$CPUS RealMemory=$MEMORY State=UNKNOWN

# パーティション定義
PartitionName=batch Nodes=slurm-compute-[1-2] Default=YES MaxTime=INFINITE State=UP
EOF

# slurmctld起動
mkdir -p /var/spool/slurmctld
chown slurm:slurm /var/spool/slurmctld
systemctl enable slurmctld
systemctl restart slurmctld

echo "Controller setup complete"
```

実行する：

```bash
conoha server deploy slurm-controller --script slurm-controller-setup.sh --env COMP1_IP=$COMP1_IP --env COMP2_IP=$COMP2_IP --env CPUS=2 --env MEMORY=2000
```

### 5. コンピュートノードセットアップ

コンピュートセットアップスクリプト `slurm-compute-setup.sh` を作成する：

```bash
#!/bin/bash
set -euo pipefail

# NFSマウント
apt-get install -y nfs-common
mkdir -p /shared
mount $CTRL_IP:/shared /shared
echo "$CTRL_IP:/shared /shared nfs defaults 0 0" >> /etc/fstab

# コントローラーからslurm.confをコピー
scp -o StrictHostKeyChecking=no root@$CTRL_IP:/etc/slurm/slurm.conf /etc/slurm/slurm.conf

# slurmd起動
mkdir -p /var/spool/slurmd
chown slurm:slurm /var/spool/slurmd
systemctl enable slurmd
systemctl restart slurmd

echo "Compute node setup complete"
```

各コンピュートノードで実行する：

```bash
conoha server deploy slurm-compute-1 --script slurm-compute-setup.sh --env CTRL_IP=$CTRL_IP
conoha server deploy slurm-compute-2 --script slurm-compute-setup.sh --env CTRL_IP=$CTRL_IP
```

### 6. 動作確認

```bash
# ノード状態確認
ssh root@$CTRL_IP "sinfo"
```

期待される出力：

```
PARTITION AVAIL TIMELIMIT NODES STATE NODELIST
batch*    up    infinite  2     idle  slurm-compute-[1-2]
```

テストジョブを実行する：

```bash
ssh root@$CTRL_IP "srun --nodes=2 hostname"
```

期待される出力：

```
slurm-compute-1
slurm-compute-2
```

バッチジョブのテスト：

```bash
ssh root@$CTRL_IP "cat > /shared/test.sh << 'SCRIPT'
#!/bin/bash
echo \"Hello from \$(hostname) at \$(date)\" > /shared/output_\$(hostname).txt
SCRIPT
chmod +x /shared/test.sh
sbatch --nodes=2 --ntasks=2 /shared/test.sh"
```

## カスタマイズ

### コンピュートノードの追加

1. 新しいサーバーを作成する
2. 共通セットアップ → コンピュートセットアップを実行する
3. コントローラーの `slurm.conf` にノード定義を追加する
4. `scontrol reconfigure` を実行する

### GPUノードの使用

GPUフレーバーを選択し、`slurm.conf` のノード定義に `Gres=gpu:1` を追加する。

### ジョブスケジューラの調整

`slurm.conf` の `SchedulerType` と `SelectType` を変更する：
- `sched/backfill`: デフォルト、バックフィルスケジューリング
- `select/cons_tres`: CPU/メモリ/GPUの個別割り当て

## トラブルシューティング

| 問題 | 対処 |
|------|------|
| ノードがdown状態 | `scontrol update nodename=<名前> state=idle` で復旧を試みる |
| Munge認証エラー | 全ノードで同じMungeキーが配置されているか確認する |
| NFSマウントが失敗する | セキュリティグループでNFSポート(2049, 111)を確認する |
| ジョブがpending | `squeue` と `sinfo` でノードの状態を確認する |
```

- [ ] **Step 2: Commit**

```bash
git add recipes/slurm-cluster.md && git commit -m "Add Slurm cluster recipe"
```

---

### Task 8: Push and Create PR

- [ ] **Step 1: Push to remote**

```bash
git push -u origin feature/conoha-cli-skill
```

- [ ] **Step 2: Create PR**

```bash
gh pr create --title "Add conoha-cli-skill content (#48)" --body "$(cat <<'EOF'
## Summary
- Add SKILL.md with ConoHa CLI command reference and recipe index
- Add 5 recipe files for infrastructure orchestration:
  - single-server-app: Docker Compose app deployment
  - single-server-script: Custom script deployment
  - k8s-cluster: Kubernetes cluster with k3s
  - openstack-platform: OpenStack with DevStack
  - slurm-cluster: Slurm HPC cluster

## Notes
- All content in Japanese (target audience)
- This PR adds the skill content to conoha-cli repo for review
- Actual skill repo (crowdy/conoha-cli-skill) will be created separately

Closes #48
EOF
)"
```

---

### Task 9: Create conoha-cli-skill Repository

After PR is merged, create the actual skill repository and populate it.

- [ ] **Step 1: Create repo**

```bash
gh repo create crowdy/conoha-cli-skill --public --description "Claude Code skill for ConoHa VPS3 CLI infrastructure orchestration"
```

- [ ] **Step 2: Clone and populate**

```bash
cd /tmp
git clone https://github.com/crowdy/conoha-cli-skill.git
cd conoha-cli-skill
# Copy SKILL.md and recipes/ from the PR branch
cp /home/tkim/dev/crowdy/conoha-cli/SKILL.md .
cp -r /home/tkim/dev/crowdy/conoha-cli/recipes/ .
git add .
git commit -m "Initial skill content: command reference + 5 recipes"
git push
```

- [ ] **Step 3: Verify install works**

```bash
conoha skill install
ls ~/.claude/skills/conoha-cli-skill/
```

Expected: `SKILL.md` and `recipes/` directory present.
