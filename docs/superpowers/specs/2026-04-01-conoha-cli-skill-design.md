# conoha-cli-skill Design Spec

## Overview

Claude Code skill for ConoHa VPS3 CLI infrastructure orchestration. Provides recipes for single-server deployments and multi-server cluster setups (Kubernetes, OpenStack, Slurm).

- **Repository**: `crowdy/conoha-cli-skill` (new, public)
- **Install**: `conoha skill install` clones to `~/.claude/skills/conoha-cli-skill/`
- **Language**: Japanese (target audience is Japanese ConoHa users)
- **Related issue**: #48

## Scope

- Phase 2: Single-server deployment recipes (app deploy, server deploy)
- Phase 3: Multi-server orchestration (k8s, OpenStack, Slurm)
- Out of scope: Multi-cloud integration (Phase 4, separate issue)

## Repository Structure

```
crowdy/conoha-cli-skill/
├── SKILL.md                          # Main entry point for Claude Code
└── recipes/
    ├── single-server-app.md          # Docker Compose app deployment
    ├── single-server-script.md       # Custom script deployment
    ├── k8s-cluster.md                # Kubernetes cluster (k3s)
    ├── openstack-platform.md         # OpenStack platform (DevStack)
    └── slurm-cluster.md              # Slurm HPC cluster
```

No README.md or auxiliary docs — skill contains only what Claude Code needs.

## SKILL.md Structure

YAML frontmatter (required for Claude Code skill triggering):

```yaml
---
name: conoha-cli
description: >
  ConoHa VPS3 CLIによるインフラ構築スキル。サーバー作成、アプリデプロイ、
  Kubernetesクラスター、OpenStackプラットフォーム、Slurmクラスターの構築を支援。
  「ConoHaでサーバーを作って」「k8sクラスターを構築して」「アプリをデプロイして」
  などのリクエストでトリガー。
---
```

Body contents (under 500 lines, Japanese, imperative form):

1. **前提条件** — conoha-cli installed, auth configured, keypair registered
2. **基本操作** — core command summary (server, flavor, image, keypair, app)
3. **レシピ一覧** — table mapping user intent to recipe files
4. **共通パターン** — shared patterns across recipes:
   - Multi-server creation with naming conventions
   - Private network configuration
   - Security group setup
   - Deploy script execution order

## Recipe Template

Each recipe follows a consistent structure:

```markdown
# レシピ名

## 概要
What it builds, when to use it

## 基本構成
Default node count, flavor, OS, network topology (text diagram)

## 手順
### 1. 事前準備 (security groups, keypairs, networks)
### 2. サーバー作成 (conoha server create commands)
### 3. 初期設定スクリプト (conoha server deploy scripts)
### 4. クラスター構成 (cluster-specific setup)
### 5. 動作確認 (verification commands)

## カスタマイズ
Variations: node count, OS, flavor, alternative tools

## トラブルシューティング
Common issues and solutions
```

## Recipe Specifications

### Phase 2: Single Server

#### single-server-app.md
- **Basic config**: Ubuntu, 1 node
- **Flow**: server create → app init → app deploy
- **Content**: Docker Compose app deployment using `conoha app` commands
- **Variations**: different app types (Node.js, Python, Go), environment variables

#### single-server-script.md
- **Basic config**: Ubuntu, 1 node
- **Flow**: server create → server deploy --script
- **Content**: Custom script deployment using `conoha server deploy`
- **Variations**: different runtimes, database setup, Nginx reverse proxy

### Phase 3: Multi-Server Orchestration

#### k8s-cluster.md
- **Basic config**: Ubuntu, 3 nodes (1 server + 2 agents), k3s
- **Flow**: create security group → create 3 servers → deploy k3s server script → deploy k3s agent scripts → verify cluster
- **Network**: private network for inter-node communication
- **Variations**: node count scaling, kubeadm instead of k3s, different CNI plugins

#### openstack-platform.md
- **Basic config**: Ubuntu, 1 node (all-in-one DevStack)
- **Flow**: create server (large flavor) → deploy DevStack install script → configure → verify
- **Variations**: multi-node (controller + compute nodes), specific OpenStack services

#### slurm-cluster.md
- **Basic config**: Ubuntu, 3 nodes (1 controller + 2 compute), Slurm
- **Flow**: create security group → create 3 servers → deploy Slurm controller script → deploy Slurm compute scripts → verify cluster
- **Network**: private network, shared NFS storage
- **Variations**: node count scaling, GPU flavors, job scheduler tuning

## Design Approach: Hybrid

Each recipe provides:
1. **Concrete default path** — specific conoha commands with real parameters, ready to execute
2. **Variation guide** — concise notes on how to customize (node count, OS, tools)

Progressive disclosure: Claude Code loads SKILL.md (~500 lines), then reads only the relevant recipe file on demand.

## Claude Code Integration Flow

1. User requests infrastructure setup (e.g., "k8sクラスターを作って")
2. Claude Code reads SKILL.md, matches request to recipe index
3. Claude Code reads the specific recipe file
4. Executes steps sequentially, capturing outputs (server IDs, IPs) for subsequent steps
5. Applies customizations if user specifies variations
6. Runs verification commands to confirm success

## Skill Design Guidelines (from skill-creator)

- **Concise**: Only include knowledge Claude doesn't already have
- **Progressive disclosure**: SKILL.md as index, recipes as references loaded on demand
- **No auxiliary files**: No README.md, CHANGELOG.md, etc.
- **Frontmatter**: `name` + `description` fields for skill triggering
- **Imperative form**: All instructions written in imperative/infinitive form
- **Under 500 lines**: SKILL.md body stays lean, detail goes to recipe files
