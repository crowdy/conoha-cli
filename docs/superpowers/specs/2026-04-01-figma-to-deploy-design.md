# Figma-to-Deploy Recipe Design Spec

## Overview

New recipe for conoha-cli-skill: `recipes/figma-to-deploy.md`

Workflow that fetches a Figma design via Figma MCP server, generates React (Vite) frontend code, packages it with Docker Compose (Nginx), and deploys to ConoHa VPS using `conoha app deploy`.

- **Target user**: Designer creates in Figma, developer uses Claude Code for code generation + ConoHa deployment
- **Language**: Japanese (consistent with existing recipes)
- **Related issue**: #49

## Scope

### In scope
1. New recipe file: `recipes/figma-to-deploy.md`
2. SKILL.md update: add Figma recipe entry to レシピ一覧 table

### Out of scope
- Figma MCP server implementation (third-party, assumed available)
- Backend API generation (frontend-only workflow)
- CI/CD pipeline setup (mentioned as customization option only)

## Prerequisites

- Figma account + API token (personal access token)
- Figma MCP server configured in Claude Code (`~/.claude/settings.json`)
- conoha-cli installed and authenticated (`conoha auth login`)
- Keypair registered (`conoha keypair list`)

> Note: Figma MCP server stability requires validation. Recipe states this in prerequisites.

## Repository Changes

### conoha-cli-skill repository

```
crowdy/conoha-cli-skill/
├── SKILL.md                          # Update: add recipe entry
└── recipes/
    ├── single-server-app.md
    ├── single-server-script.md
    ├── k8s-cluster.md
    ├── openstack-platform.md
    ├── slurm-cluster.md
    └── figma-to-deploy.md            # New
```

## Recipe Structure

Follows existing recipe template with adjustments for the code generation step.

### 概要 (Overview)
- What: Figma design → React(Vite) → Docker Compose → ConoHa deploy
- When: Designer hands off Figma design, developer wants rapid deployment

### 基本構成 (Basic Configuration)
- Source: Figma MCP server
- Frontend: React (Vite) + Nginx serving static files
- Infrastructure: Ubuntu, 1 node, Docker Compose
- Container architecture: multi-stage build (npm build → Nginx)

### 手順 (Steps)

#### 1. 事前準備 (Preparation)
- Verify Figma MCP server connectivity
- Confirm ConoHa auth and keypair
- Identify target Figma file URL and frame/page

#### 2. Figmaデザイン取得 (Fetch Figma Design)
- Use Figma MCP to fetch file structure
- Identify target frames/components
- Extract design tokens: colors, fonts, spacing, layout

#### 3. コード生成 (Code Generation)
- Scaffold React (Vite) project: `npm create vite@latest`
- Generate React components from Figma design
- Apply design tokens as CSS variables / Tailwind config
- Ensure responsive layout
- Generate `index.html` with proper meta tags

#### 4. Docker Compose構成 (Docker Compose Setup)
- Dockerfile: multi-stage build
  - Stage 1: `node:lts-alpine` → `npm ci && npm run build`
  - Stage 2: `nginx:alpine` → copy build output to `/usr/share/nginx/html`
- `docker-compose.yml`: single nginx service, port 80
- `nginx.conf`: SPA routing (`try_files $uri $uri/ /index.html`)

#### 5. ConoHaデプロイ (Deploy to ConoHa)
- Create server if needed: `conoha server create`
- Initialize app environment: `conoha app init <server> --app-name <name>`
- Deploy: `conoha app deploy <server> --app-name <name>`

#### 6. 動作確認 (Verification)
- `conoha app status <server> --app-name <name>`
- Browser access to server IP
- Visual comparison with original Figma design

### カスタマイズ (Customization)
- Alternative frameworks: Next.js, Vue (Nuxt)
- SSL/TLS with Let's Encrypt (certbot container)
- Custom domain via ConoHa DNS (`conoha dns`)
- CI/CD pipeline addition

### トラブルシューティング (Troubleshooting)
- Figma MCP connection errors: token validity, MCP server config
- Build errors: Node.js version, dependency conflicts
- Deploy failures: SSH connectivity, Docker not initialized (`conoha app init`)
- Nginx 404: SPA routing misconfiguration

## SKILL.md Changes

Add entry to レシピ一覧 table:

| ユーザーの要望 | レシピ |
|---|---|
| FigmaデザインからWebアプリを作ってデプロイして | `recipes/figma-to-deploy.md` |

Add trigger phrases to SKILL.md description:
- 「Figmaからデプロイ」
- 「デザインからアプリを作って」
- 「Figmaのデザインを実装して」

## Design Decisions

1. **Single recipe, not split**: The Figma → deploy flow is one continuous workflow. Splitting would break the narrative.
2. **React (Vite) as default**: Most universal choice for static frontend. Alternatives listed in customization.
3. **Nginx for serving**: Simple, production-ready static file server. Matches Docker Compose deployment model.
4. **Multi-stage Docker build**: Keeps image small, no Node.js in production.
5. **Figma MCP assumed working**: Recipe states prerequisite clearly. No fallback to REST API — keeps recipe focused.
