# conoha-cli.jp Documentation Site Design

## Overview

GitHub Pages documentation site for conoha-cli, targeting Japanese developers who already use ConoHa VPS3. Built with VitePress, hosted at `conoha-cli.jp`, in a separate repository (`crowdy/conoha-cli-pages`).

**Goals:**
- Extremely easy to understand (とにかく分かりやすい)
- Tutorials + command reference + real-world deployment examples
- Three languages: Japanese (primary), English, Korean

## Project Structure

**Repository:** `crowdy/conoha-cli-pages` (separate from `crowdy/conoha-cli`)

**Why separate:**
- Custom domain (`conoha-cli.jp`) management is cleaner per-repo
- Docs build (VitePress/Node.js) and CLI build (Go/goreleaser) don't interfere
- Doc changes don't trigger CLI CI

```
crowdy/conoha-cli-pages/
├── .github/
│   └── workflows/
│       └── deploy.yml            # GitHub Actions: build + deploy to gh-pages
├── docs/
│   ├── .vitepress/
│   │   └── config.ts             # VitePress config (i18n, sidebar, nav)
│   ├── ja/                       # Japanese (default locale)
│   │   ├── index.md              # Landing page
│   │   ├── guide/                # Tutorials
│   │   │   ├── getting-started.md
│   │   │   ├── server.md
│   │   │   ├── app-deploy.md
│   │   │   └── app-management.md
│   │   ├── examples/             # Real-world deployment cases
│   │   │   ├── nextjs.md
│   │   │   ├── fastapi-ai-chatbot.md
│   │   │   ├── rails-postgresql.md
│   │   │   └── wordpress.md
│   │   └── reference/            # Command reference
│   │       ├── auth.md
│   │       ├── server.md
│   │       └── app.md
│   ├── en/                       # English (same structure)
│   └── ko/                       # Korean (same structure)
├── package.json
└── CNAME                         # conoha-cli.jp
```

## Target Audience

Japanese developers already using ConoHa VPS3 who want to manage their VPS via CLI instead of the web console.

## Content Design

### Tutorials (guide/)

Sequential, follow-along guides. A beginner can go from installation to a running app by following them in order.

| Order | Page | Content |
|-------|------|---------|
| 1 | getting-started | Install conoha-cli, `conoha auth login`, basic config |
| 2 | server | `server create/list/start/stop`, flavor selection, keypair |
| 3 | app-deploy | `app init` → `app deploy` → `app logs`, Dockerfile basics |
| 4 | app-management | `app env set/get`, `app destroy`, `app list` |

### Real-World Examples (examples/)

Each example follows a unified flow: **What you'll build → Prerequisites → Dockerfile → app init → app deploy → Verify it works**

**Tier 1 (initial release):**

| Example | Framework | Use Case | Why |
|---------|-----------|----------|-----|
| nextjs | Next.js | Full-stack web app | Most popular frontend in Japan, Vercel cost-saving alternative |
| fastapi-ai-chatbot | FastAPI + Ollama | AI chatbot | LLM self-hosting demand, existing business case docs |
| rails-postgresql | Rails + PostgreSQL | Web service | Standard stack for Japanese startups/indie devs |
| wordpress | WordPress + MySQL | Blog/corporate site | Rental server → VPS migration demand |

**Tier 2 (second release):**
- Django + PostgreSQL (admin/API)
- Go Echo/Gin (lightweight API)
- Spring Boot (enterprise Java)
- Dify / LangChain (AI agent platform)

**Tier 3 (incremental):**
- Discord.js (Discord Bot)
- Redmine / GitLab (self-hosted internal tools)
- Minecraft server (popular in Japan)
- Mastodon (Fediverse instance)

### Command Reference (reference/)

One page per command group. Each command entry follows a consistent format:

```
## command name

Description (one sentence)

### Usage
conoha <command> [flags]

### Flags
| Flag | Description | Default |
|------|-------------|---------|

### Examples
$ conoha server list
$ conoha server create --name myserver --flavor g2l-t-2 --image ubuntu-24.04
```

**Initial release:** auth, server, app (3 pages)
**Later:** flavor, keypair, volume, image, network, dns, config

## Internationalization (i18n)

- **Default locale:** Japanese (`ja`)
- Root URL (`/`) redirects to Japanese content
- English at `/en/`, Korean at `/ko/`
- VitePress `locales` config provides per-language sidebar, nav, and search
- Language switcher in the navbar

**Translation priority:** Write Japanese first, then English, then Korean. Not all pages need to exist in all languages from day one.

## Tech Stack

- **SSG:** VitePress (latest)
- **Runtime:** Node.js
- **Hosting:** GitHub Pages
- **Domain:** conoha-cli.jp (custom domain via CNAME)
- **CI/CD:** GitHub Actions — push to `main` → VitePress build → deploy to `gh-pages` branch
- **Search:** VitePress built-in local search (supports Japanese)

## GitHub Actions Deploy Pipeline

Trigger: push to `main` branch

Steps:
1. Checkout
2. Setup Node.js
3. Install dependencies (`npm ci`)
4. Build VitePress (`npm run docs:build`)
5. Deploy to `gh-pages` branch

## Integration with conoha-cli

- conoha-cli README.md: add link to `https://conoha-cli.jp`
- Docs site: link to GitHub repo (`crowdy/conoha-cli`) and releases page
- Future: CLI `--help` output includes "詳しくは https://conoha-cli.jp を参照"

## Initial Release Scope

**Phase 1 (MVP):**
- VitePress project scaffold + GitHub Actions deploy
- Japanese only: landing page + 4 tutorials + 4 examples + 3 reference pages
- Custom domain setup

**Phase 2:**
- English translation
- Korean translation
- Tier 2 examples (4 more)

**Phase 3:**
- Remaining command reference pages
- Tier 3 examples
- CLI `--help` integration
