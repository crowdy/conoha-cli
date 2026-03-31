# conoha-cli-app-samples Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create the `conoha-cli-app-samples` repository with 5 sample apps (hello-world, nextjs, fastapi-ai-chatbot, rails-postgresql, wordpress-mysql) that demonstrate `conoha app deploy`.

**Architecture:** Flat directory structure — each sample is a self-contained directory with compose.yml, Dockerfile (where applicable), source code, and Japanese README. All code comments and app output are in English.

**Tech Stack:** Docker, Docker Compose, nginx, Node.js/Next.js, Python/FastAPI, Ruby/Rails, WordPress, MySQL, PostgreSQL, Ollama

---

### Task 1: Repository scaffolding

**Files:**
- Create: `/home/tkim/dev/crowdy/conoha-cli-app-samples/README.md`
- Create: `/home/tkim/dev/crowdy/conoha-cli-app-samples/LICENSE`
- Create: `/home/tkim/dev/crowdy/conoha-cli-app-samples/.gitignore`

- [ ] **Step 1: Create the directory and initialize git**

```bash
mkdir -p /home/tkim/dev/crowdy/conoha-cli-app-samples
cd /home/tkim/dev/crowdy/conoha-cli-app-samples
git init
```

- [ ] **Step 2: Create LICENSE (MIT)**

Create `/home/tkim/dev/crowdy/conoha-cli-app-samples/LICENSE`:

```
MIT License

Copyright (c) 2026 crowdy

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 3: Create .gitignore**

Create `/home/tkim/dev/crowdy/conoha-cli-app-samples/.gitignore`:

```
.env
.env.server
.DS_Store
node_modules/
tmp/
log/
*.log
```

- [ ] **Step 4: Create root README.md**

Create `/home/tkim/dev/crowdy/conoha-cli-app-samples/README.md`:

```markdown
# conoha-cli-app-samples

[conoha-cli](https://github.com/crowdy/conoha-cli) の `app deploy` コマンドで使えるサンプルアプリ集です。

各サンプルディレクトリにはすぐにデプロイできる `compose.yml`、`Dockerfile`、ソースコードが含まれています。

## 前提条件

- [conoha-cli](https://github.com/crowdy/conoha-cli) がインストール済み
- ConoHa VPS3 アカウント
- SSH キーペアが設定済み（`conoha keypair create` で作成可能）

## 使い方

```bash
# 1. このリポジトリをクローン
git clone https://github.com/crowdy/conoha-cli-app-samples.git
cd conoha-cli-app-samples

# 2. サーバーを作成（まだない場合）
conoha server create --name myserver --flavor g2l-t-2 --image ubuntu-24.04 --key mykey

# 3. サンプルを選んでデプロイ
cd hello-world
conoha app init myserver --app-name hello-world
conoha app deploy myserver --app-name hello-world

# 4. 動作確認
conoha app logs myserver --app-name hello-world
```

## サンプル一覧

| サンプル | スタック | 説明 | 推奨フレーバー |
|---------|---------|------|--------------|
| [hello-world](hello-world/) | nginx + 静的HTML | 最もシンプルなサンプル | g2l-t-1 (1GB) |
| [nextjs](nextjs/) | Next.js (standalone) | Next.js デフォルトページ | g2l-t-2 (2GB) |
| [fastapi-ai-chatbot](fastapi-ai-chatbot/) | FastAPI + Ollama | AI チャットボット | g2l-t-4 (4GB) |
| [rails-postgresql](rails-postgresql/) | Rails + PostgreSQL | Rails scaffold アプリ | g2l-t-2 (2GB) |
| [wordpress-mysql](wordpress-mysql/) | WordPress + MySQL | WordPress ブログ | g2l-t-2 (2GB) |

## 自分のアプリをデプロイするには

`compose.yml`（または `docker-compose.yml`）があるディレクトリであれば、同じ手順でデプロイできます。

```bash
cd your-app
conoha app init myserver --app-name your-app
conoha app deploy myserver --app-name your-app
```

`Dockerfile` でビルドする場合は `compose.yml` の `build: .` を使ってください。

## 関連リンク

- [conoha-cli](https://github.com/crowdy/conoha-cli) — ConoHa VPS3 CLI ツール
- [ドキュメント](https://conoha-cli.jp) — チュートリアル・コマンドリファレンス
```

- [ ] **Step 5: Commit**

```bash
git add LICENSE .gitignore README.md
git commit -m "Initial repo scaffolding with README, LICENSE, gitignore"
```

---

### Task 2: hello-world sample

**Files:**
- Create: `hello-world/README.md`
- Create: `hello-world/Dockerfile`
- Create: `hello-world/compose.yml`
- Create: `hello-world/.dockerignore`
- Create: `hello-world/index.html`

- [ ] **Step 1: Create hello-world/Dockerfile**

```dockerfile
FROM nginx:alpine
COPY index.html /usr/share/nginx/html/index.html
```

- [ ] **Step 2: Create hello-world/compose.yml**

```yaml
services:
  web:
    build: .
    ports:
      - "80:80"
```

- [ ] **Step 3: Create hello-world/.dockerignore**

```
README.md
.git
```

- [ ] **Step 4: Create hello-world/index.html**

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Hello ConoHa</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      display: flex;
      justify-content: center;
      align-items: center;
      min-height: 100vh;
      margin: 0;
      background: #f5f5f5;
    }
    .container {
      text-align: center;
      padding: 2rem;
    }
    h1 { color: #333; font-size: 2.5rem; }
    p { color: #666; font-size: 1.2rem; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Hello from ConoHa!</h1>
    <p>Deployed with <code>conoha app deploy</code></p>
  </div>
</body>
</html>
```

- [ ] **Step 5: Create hello-world/README.md**

```markdown
# hello-world

nginx で静的HTMLを配信する最もシンプルなサンプルです。初めて `conoha app deploy` を試す方におすすめです。

## 構成

- nginx (Alpine)
- ポート: 80

## 前提条件

- conoha-cli がインストール済み
- ConoHa VPS3 アカウント
- SSH キーペア設定済み

## デプロイ

```bash
# サーバー作成（まだない場合）
conoha server create --name myserver --flavor g2l-t-1 --image ubuntu-24.04 --key mykey

# アプリ初期化
conoha app init myserver --app-name hello-world

# デプロイ
conoha app deploy myserver --app-name hello-world
```

## 動作確認

ブラウザで `http://<サーバーIP>` にアクセスすると「Hello from ConoHa!」と表示されます。

## カスタマイズ

`index.html` を編集して再度 `conoha app deploy` するだけで更新できます。
```

- [ ] **Step 6: Verify locally**

```bash
cd hello-world
docker compose up -d --build
curl http://localhost
docker compose down
cd ..
```

Expected: HTML page with "Hello from ConoHa!" is returned.

- [ ] **Step 7: Commit**

```bash
git add hello-world/
git commit -m "Add hello-world sample (nginx + static HTML)"
```

---

### Task 3: nextjs sample

**Files:**
- Create: `nextjs/README.md`
- Create: `nextjs/Dockerfile`
- Create: `nextjs/compose.yml`
- Create: `nextjs/.dockerignore`
- Create: `nextjs/package.json`
- Create: `nextjs/next.config.ts`
- Create: `nextjs/tsconfig.json`
- Create: `nextjs/app/layout.tsx`
- Create: `nextjs/app/page.tsx`
- Create: `nextjs/app/globals.css`
- Create: `nextjs/public/` (empty, or favicon)

- [ ] **Step 1: Create nextjs/package.json**

```json
{
  "name": "conoha-nextjs-sample",
  "version": "1.0.0",
  "private": true,
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start"
  },
  "dependencies": {
    "next": "15.3.1",
    "react": "^19.0.0",
    "react-dom": "^19.0.0"
  },
  "devDependencies": {
    "@types/node": "^22.0.0",
    "@types/react": "^19.0.0",
    "@types/react-dom": "^19.0.0",
    "typescript": "^5.7.0"
  }
}
```

- [ ] **Step 2: Create nextjs/next.config.ts**

```typescript
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
};

export default nextConfig;
```

- [ ] **Step 3: Create nextjs/tsconfig.json**

```json
{
  "compilerOptions": {
    "target": "ES2017",
    "lib": ["dom", "dom.iterable", "esnext"],
    "allowJs": true,
    "skipLibCheck": true,
    "strict": true,
    "noEmit": true,
    "esModuleInterop": true,
    "module": "esnext",
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "jsx": "preserve",
    "incremental": true,
    "plugins": [{ "name": "next" }],
    "paths": { "@/*": ["./*"] }
  },
  "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx"],
  "exclude": ["node_modules"]
}
```

- [ ] **Step 4: Create nextjs/app/globals.css**

```css
* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  background: #f5f5f5;
  color: #333;
}
```

- [ ] **Step 5: Create nextjs/app/layout.tsx**

```tsx
import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Next.js on ConoHa",
  description: "Next.js app deployed with conoha app deploy",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
```

- [ ] **Step 6: Create nextjs/app/page.tsx**

```tsx
export default function Home() {
  return (
    <main
      style={{
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        minHeight: "100vh",
      }}
    >
      <div style={{ textAlign: "center", padding: "2rem" }}>
        <h1 style={{ fontSize: "2.5rem", marginBottom: "1rem" }}>
          Next.js on ConoHa
        </h1>
        <p style={{ fontSize: "1.2rem", color: "#666" }}>
          Deployed with <code>conoha app deploy</code>
        </p>
      </div>
    </main>
  );
}
```

- [ ] **Step 7: Create nextjs/Dockerfile**

```dockerfile
# Stage 1: Install dependencies
FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json ./
RUN npm install

# Stage 2: Build the application
FROM node:22-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

# Stage 3: Production runner
FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
USER nextjs
EXPOSE 3000
ENV PORT=3000
CMD ["node", "server.js"]
```

- [ ] **Step 8: Create nextjs/compose.yml**

```yaml
services:
  web:
    build: .
    ports:
      - "3000:3000"
```

- [ ] **Step 9: Create nextjs/.dockerignore**

```
README.md
.git
node_modules
.next
```

- [ ] **Step 10: Create nextjs/README.md**

```markdown
# nextjs

Next.js アプリをスタンドアロンモードでデプロイするサンプルです。マルチステージビルドで軽量なイメージを生成します。

## 構成

- Node.js 22 + Next.js 15 (standalone output)
- ポート: 3000

## 前提条件

- conoha-cli がインストール済み
- ConoHa VPS3 アカウント
- SSH キーペア設定済み

## デプロイ

```bash
# サーバー作成（まだない場合）
conoha server create --name myserver --flavor g2l-t-2 --image ubuntu-24.04 --key mykey

# アプリ初期化
conoha app init myserver --app-name nextjs

# デプロイ
conoha app deploy myserver --app-name nextjs
```

## 動作確認

ブラウザで `http://<サーバーIP>:3000` にアクセスすると「Next.js on ConoHa」と表示されます。

## カスタマイズ

- `app/page.tsx` を編集してページ内容を変更
- `app/` 以下にファイルを追加して App Router でルーティング
- `next.config.ts` で Next.js の設定を変更
```

- [ ] **Step 11: Verify locally**

```bash
cd nextjs
docker compose up -d --build
curl http://localhost:3000
docker compose down
cd ..
```

Expected: HTML page with "Next.js on ConoHa" text.

- [ ] **Step 12: Commit**

```bash
git add nextjs/
git commit -m "Add Next.js sample (standalone multi-stage build)"
```

---

### Task 4: fastapi-ai-chatbot sample

**Files:**
- Create: `fastapi-ai-chatbot/README.md`
- Create: `fastapi-ai-chatbot/Dockerfile`
- Create: `fastapi-ai-chatbot/compose.yml`
- Create: `fastapi-ai-chatbot/.dockerignore`
- Create: `fastapi-ai-chatbot/requirements.txt`
- Create: `fastapi-ai-chatbot/main.py`
- Create: `fastapi-ai-chatbot/templates/index.html`

- [ ] **Step 1: Create fastapi-ai-chatbot/requirements.txt**

```
fastapi==0.115.12
uvicorn[standard]==0.34.2
httpx==0.28.1
jinja2==3.1.6
```

- [ ] **Step 2: Create fastapi-ai-chatbot/main.py**

```python
import httpx
from fastapi import FastAPI, Request
from fastapi.responses import HTMLResponse
from fastapi.templating import Jinja2Templates
from pydantic import BaseModel

app = FastAPI()
templates = Jinja2Templates(directory="templates")

OLLAMA_URL = "http://ollama:11434"
MODEL = "tinyllama"


class ChatRequest(BaseModel):
    message: str


@app.get("/", response_class=HTMLResponse)
async def index(request: Request):
    return templates.TemplateResponse("index.html", {"request": request})


@app.post("/chat")
async def chat(req: ChatRequest):
    async with httpx.AsyncClient(timeout=120.0) as client:
        response = await client.post(
            f"{OLLAMA_URL}/api/generate",
            json={"model": MODEL, "prompt": req.message, "stream": False},
        )
        response.raise_for_status()
        data = response.json()
    return {"response": data.get("response", "")}


@app.get("/health")
async def health():
    return {"status": "ok"}
```

- [ ] **Step 3: Create fastapi-ai-chatbot/templates/index.html**

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>AI Chatbot</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      max-width: 600px;
      margin: 2rem auto;
      padding: 0 1rem;
      background: #f5f5f5;
    }
    h1 { color: #333; }
    #messages {
      border: 1px solid #ddd;
      border-radius: 8px;
      padding: 1rem;
      min-height: 300px;
      max-height: 500px;
      overflow-y: auto;
      background: #fff;
      margin-bottom: 1rem;
    }
    .msg { margin-bottom: 0.5rem; padding: 0.5rem; border-radius: 4px; }
    .user { background: #e3f2fd; }
    .bot { background: #f5f5f5; }
    form { display: flex; gap: 0.5rem; }
    input {
      flex: 1;
      padding: 0.5rem;
      border: 1px solid #ddd;
      border-radius: 4px;
      font-size: 1rem;
    }
    button {
      padding: 0.5rem 1.5rem;
      background: #1976d2;
      color: #fff;
      border: none;
      border-radius: 4px;
      cursor: pointer;
      font-size: 1rem;
    }
    button:disabled { opacity: 0.5; }
  </style>
</head>
<body>
  <h1>AI Chatbot</h1>
  <p>Powered by Ollama (tinyllama)</p>
  <div id="messages"></div>
  <form id="chatForm">
    <input type="text" id="input" placeholder="Type a message..." autocomplete="off" required>
    <button type="submit" id="btn">Send</button>
  </form>
  <script>
    const form = document.getElementById("chatForm");
    const input = document.getElementById("input");
    const messages = document.getElementById("messages");
    const btn = document.getElementById("btn");

    function addMessage(text, cls) {
      const div = document.createElement("div");
      div.className = "msg " + cls;
      div.textContent = text;
      messages.appendChild(div);
      messages.scrollTop = messages.scrollHeight;
    }

    form.addEventListener("submit", async (e) => {
      e.preventDefault();
      const msg = input.value.trim();
      if (!msg) return;
      addMessage(msg, "user");
      input.value = "";
      btn.disabled = true;
      try {
        const res = await fetch("/chat", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ message: msg }),
        });
        const data = await res.json();
        addMessage(data.response, "bot");
      } catch {
        addMessage("Error: could not reach the server.", "bot");
      }
      btn.disabled = false;
      input.focus();
    });
  </script>
</body>
</html>
```

- [ ] **Step 4: Create fastapi-ai-chatbot/Dockerfile**

```dockerfile
FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8000
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
```

- [ ] **Step 5: Create fastapi-ai-chatbot/compose.yml**

```yaml
services:
  app:
    build: .
    ports:
      - "8000:8000"
    depends_on:
      - ollama

  ollama:
    image: ollama/ollama:latest
    volumes:
      - ollama_data:/root/.ollama
    # Pull the model on first start
    entrypoint: ["/bin/sh", "-c", "ollama serve & sleep 5 && ollama pull tinyllama && wait"]

volumes:
  ollama_data:
```

- [ ] **Step 6: Create fastapi-ai-chatbot/.dockerignore**

```
README.md
.git
__pycache__
*.pyc
.venv
```

- [ ] **Step 7: Create fastapi-ai-chatbot/README.md**

```markdown
# fastapi-ai-chatbot

FastAPI と Ollama を使ったシンプルな AI チャットボットです。ブラウザから質問すると LLM が回答します。

## 構成

- Python 3.12 + FastAPI（アプリサーバー）
- Ollama + tinyllama モデル（LLM）
- ポート: 8000

## 前提条件

- conoha-cli がインストール済み
- ConoHa VPS3 アカウント
- SSH キーペア設定済み

## デプロイ

```bash
# サーバー作成（4GB以上推奨）
conoha server create --name myserver --flavor g2l-t-4 --image ubuntu-24.04 --key mykey

# アプリ初期化
conoha app init myserver --app-name chatbot

# デプロイ
conoha app deploy myserver --app-name chatbot
```

初回起動時に tinyllama モデルのダウンロード（約600MB）が自動で行われます。完了まで数分かかります。

## 動作確認

ブラウザで `http://<サーバーIP>:8000` にアクセスするとチャット画面が表示されます。

## カスタマイズ

- `compose.yml` の `ollama pull tinyllama` を別のモデル（例: `gemma3:1b`）に変更
- `main.py` の `MODEL` 変数を合わせて変更
- より大きなモデルを使う場合はメモリの多いフレーバーを選択
```

- [ ] **Step 8: Commit**

```bash
git add fastapi-ai-chatbot/
git commit -m "Add FastAPI + Ollama AI chatbot sample"
```

---

### Task 5: rails-postgresql sample

**Files:**
- Create: `rails-postgresql/README.md`
- Create: `rails-postgresql/Dockerfile`
- Create: `rails-postgresql/compose.yml`
- Create: `rails-postgresql/.dockerignore`
- Create: `rails-postgresql/Gemfile`
- Create: `rails-postgresql/Gemfile.lock` (empty)
- Create: `rails-postgresql/Rakefile`
- Create: `rails-postgresql/config.ru`
- Create: `rails-postgresql/bin/rails`
- Create: `rails-postgresql/bin/docker-entrypoint`
- Create: `rails-postgresql/config/application.rb`
- Create: `rails-postgresql/config/environment.rb`
- Create: `rails-postgresql/config/database.yml`
- Create: `rails-postgresql/config/routes.rb`
- Create: `rails-postgresql/config/environments/production.rb`
- Create: `rails-postgresql/app/controllers/application_controller.rb`
- Create: `rails-postgresql/app/controllers/posts_controller.rb`
- Create: `rails-postgresql/app/models/application_record.rb`
- Create: `rails-postgresql/app/models/post.rb`
- Create: `rails-postgresql/app/views/layouts/application.html.erb`
- Create: `rails-postgresql/app/views/posts/index.html.erb`
- Create: `rails-postgresql/app/views/posts/_form.html.erb`
- Create: `rails-postgresql/db/migrate/20260101000000_create_posts.rb`
- Create: `rails-postgresql/db/schema.rb`

- [ ] **Step 1: Create rails-postgresql/Gemfile**

```ruby
source "https://rubygems.org"

gem "rails", "~> 8.0"
gem "pg", "~> 1.5"
gem "puma", "~> 6.5"
```

- [ ] **Step 2: Create rails-postgresql/Gemfile.lock**

Create an empty file. The Docker build will run `bundle install` which generates the real lock file.

```
(empty file)
```

- [ ] **Step 3: Create rails-postgresql/Rakefile**

```ruby
require_relative "config/application"
Rails.application.load_tasks
```

- [ ] **Step 4: Create rails-postgresql/config.ru**

```ruby
require_relative "config/environment"
run Rails.application
```

- [ ] **Step 5: Create rails-postgresql/bin/rails**

```ruby
#!/usr/bin/env ruby
APP_PATH = File.expand_path("../config/application", __dir__)
require_relative "../config/application"
require "rails/command"
Rails::Command.invoke("application", ARGV)
```

Make executable: `chmod +x rails-postgresql/bin/rails`

- [ ] **Step 6: Create rails-postgresql/bin/docker-entrypoint**

```bash
#!/bin/bash
set -e

# Run database migrations automatically
rails db:prepare

exec "$@"
```

Make executable: `chmod +x rails-postgresql/bin/docker-entrypoint`

- [ ] **Step 7: Create rails-postgresql/config/application.rb**

```ruby
require_relative "boot" rescue nil
require "rails"
require "active_model/railtie"
require "active_record/railtie"
require "action_controller/railtie"
require "action_view/railtie"

module ConohaRailsSample
  class Application < Rails::Application
    config.load_defaults 8.0
    config.eager_load = true
    config.secret_key_base = ENV.fetch("SECRET_KEY_BASE") { SecureRandom.hex(64) }
  end
end
```

- [ ] **Step 8: Create rails-postgresql/config/environment.rb**

```ruby
require_relative "application"
Rails.application.initialize!
```

- [ ] **Step 9: Create rails-postgresql/config/database.yml**

```yaml
default: &default
  adapter: postgresql
  encoding: unicode
  pool: 5
  host: <%= ENV.fetch("DB_HOST", "db") %>
  username: <%= ENV.fetch("DB_USER", "postgres") %>
  password: <%= ENV.fetch("DB_PASSWORD", "postgres") %>

production:
  <<: *default
  database: <%= ENV.fetch("DB_NAME", "app_production") %>

development:
  <<: *default
  database: app_development
```

- [ ] **Step 10: Create rails-postgresql/config/routes.rb**

```ruby
Rails.application.routes.draw do
  root "posts#index"
  resources :posts, only: [:index, :create, :destroy]
end
```

- [ ] **Step 11: Create rails-postgresql/config/environments/production.rb**

```ruby
require "active_support/core_ext/integer/time"

Rails.application.configure do
  config.enable_reloading = false
  config.eager_load = true
  config.consider_all_requests_local = false
  config.public_file_server.enabled = true
  config.log_level = :info
  config.log_tags = [:request_id]
  config.action_controller.perform_caching = true
end
```

- [ ] **Step 12: Create rails-postgresql/app/controllers/application_controller.rb**

```ruby
class ApplicationController < ActionController::Base
  allow_browser versions: :modern
end
```

- [ ] **Step 13: Create rails-postgresql/app/controllers/posts_controller.rb**

```ruby
class PostsController < ApplicationController
  def index
    @posts = Post.order(created_at: :desc)
    @post = Post.new
  end

  def create
    Post.create!(post_params)
    redirect_to root_path
  end

  def destroy
    Post.find(params[:id]).destroy
    redirect_to root_path
  end

  private

  def post_params
    params.require(:post).permit(:title, :body)
  end
end
```

- [ ] **Step 14: Create rails-postgresql/app/models/application_record.rb**

```ruby
class ApplicationRecord < ActiveRecord::Base
  self.abstract_class = true
end
```

- [ ] **Step 15: Create rails-postgresql/app/models/post.rb**

```ruby
class Post < ApplicationRecord
  validates :title, presence: true
end
```

- [ ] **Step 16: Create rails-postgresql/app/views/layouts/application.html.erb**

```erb
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Rails on ConoHa</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      max-width: 700px;
      margin: 2rem auto;
      padding: 0 1rem;
      background: #f5f5f5;
      color: #333;
    }
    h1 { margin-bottom: 1rem; }
    .post { background: #fff; padding: 1rem; border-radius: 8px; margin-bottom: 1rem; }
    .post h2 { margin: 0 0 0.5rem; font-size: 1.2rem; }
    .post p { margin: 0; color: #666; }
    form { background: #fff; padding: 1rem; border-radius: 8px; margin-bottom: 2rem; }
    input, textarea { width: 100%; padding: 0.5rem; margin-bottom: 0.5rem; border: 1px solid #ddd; border-radius: 4px; font-size: 1rem; box-sizing: border-box; }
    textarea { height: 80px; resize: vertical; }
    button { padding: 0.5rem 1.5rem; background: #1976d2; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 1rem; }
    .delete { background: #d32f2f; font-size: 0.85rem; padding: 0.3rem 0.8rem; }
  </style>
</head>
<body>
  <%= yield %>
</body>
</html>
```

- [ ] **Step 17: Create rails-postgresql/app/views/posts/index.html.erb**

```erb
<h1>Rails on ConoHa</h1>

<%= render "form", post: @post %>

<% @posts.each do |post| %>
  <div class="post">
    <h2><%= post.title %></h2>
    <p><%= post.body %></p>
    <%= button_to "Delete", post_path(post), method: :delete, class: "delete" %>
  </div>
<% end %>
```

- [ ] **Step 18: Create rails-postgresql/app/views/posts/_form.html.erb**

```erb
<%= form_with model: post do |f| %>
  <%= f.text_field :title, placeholder: "Title" %>
  <%= f.text_area :body, placeholder: "Body (optional)" %>
  <%= f.submit "Create Post" %>
<% end %>
```

- [ ] **Step 19: Create rails-postgresql/db/migrate/20260101000000_create_posts.rb**

```ruby
class CreatePosts < ActiveRecord::Migration[8.0]
  def change
    create_table :posts do |t|
      t.string :title, null: false
      t.text :body
      t.timestamps
    end
  end
end
```

- [ ] **Step 20: Create rails-postgresql/db/schema.rb**

```ruby
ActiveRecord::Schema[8.0].define(version: 2026_01_01_000000) do
  create_table "posts", force: :cascade do |t|
    t.string "title", null: false
    t.text "body"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
  end
end
```

- [ ] **Step 21: Create rails-postgresql/Dockerfile**

```dockerfile
FROM ruby:3.3-slim AS builder
WORKDIR /app
RUN apt-get update -qq && apt-get install -y build-essential libpq-dev
COPY Gemfile Gemfile.lock ./
RUN bundle install --jobs 4

FROM ruby:3.3-slim
WORKDIR /app
RUN apt-get update -qq && apt-get install -y libpq5 && rm -rf /var/lib/apt/lists/*
COPY --from=builder /usr/local/bundle /usr/local/bundle
COPY . .
RUN chmod +x bin/docker-entrypoint
ENV RAILS_ENV=production
EXPOSE 3000
ENTRYPOINT ["bin/docker-entrypoint"]
CMD ["bundle", "exec", "puma", "-C", "config/puma.rb", "-b", "tcp://0.0.0.0:3000"]
```

Note: No puma.rb config needed — command-line flags are sufficient. Replace CMD:

```dockerfile
CMD ["bundle", "exec", "puma", "-b", "tcp://0.0.0.0:3000"]
```

- [ ] **Step 22: Create rails-postgresql/compose.yml**

```yaml
services:
  web:
    build: .
    ports:
      - "3000:3000"
    environment:
      - RAILS_ENV=production
      - DB_HOST=db
      - DB_USER=postgres
      - DB_PASSWORD=postgres
      - DB_NAME=app_production
      - SECRET_KEY_BASE=placeholder_change_me_in_production
    depends_on:
      db:
        condition: service_healthy

  db:
    image: postgres:17-alpine
    environment:
      - POSTGRES_PASSWORD=postgres
    volumes:
      - db_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  db_data:
```

- [ ] **Step 23: Create rails-postgresql/.dockerignore**

```
README.md
.git
tmp
log
node_modules
```

- [ ] **Step 24: Create rails-postgresql/README.md**

```markdown
# rails-postgresql

Rails と PostgreSQL を使ったシンプルな投稿アプリです。scaffold 相当の CRUD 機能を持ちます。

## 構成

- Ruby 3.3 + Rails 8.0
- PostgreSQL 17
- ポート: 3000

## 前提条件

- conoha-cli がインストール済み
- ConoHa VPS3 アカウント
- SSH キーペア設定済み

## デプロイ

```bash
# サーバー作成（まだない場合）
conoha server create --name myserver --flavor g2l-t-2 --image ubuntu-24.04 --key mykey

# アプリ初期化
conoha app init myserver --app-name rails-app

# デプロイ
conoha app deploy myserver --app-name rails-app
```

DB マイグレーションはコンテナ起動時に自動実行されます。

## 動作確認

ブラウザで `http://<サーバーIP>:3000` にアクセスすると投稿一覧ページが表示されます。

## カスタマイズ

- `app/controllers/` と `app/views/` を編集して機能を追加
- `db/migrate/` に新しいマイグレーションを追加してスキーマを変更
- 本番環境では `compose.yml` の `SECRET_KEY_BASE` と `DB_PASSWORD` を `.env.server` で管理
```

- [ ] **Step 25: Commit**

```bash
git add rails-postgresql/
git commit -m "Add Rails + PostgreSQL sample (CRUD posts app)"
```

---

### Task 6: wordpress-mysql sample

**Files:**
- Create: `wordpress-mysql/README.md`
- Create: `wordpress-mysql/compose.yml`
- Create: `wordpress-mysql/.dockerignore`

- [ ] **Step 1: Create wordpress-mysql/compose.yml**

```yaml
services:
  wordpress:
    image: wordpress:latest
    ports:
      - "80:80"
    environment:
      - WORDPRESS_DB_HOST=db
      - WORDPRESS_DB_USER=wordpress
      - WORDPRESS_DB_PASSWORD=${MYSQL_PASSWORD:-wordpress}
      - WORDPRESS_DB_NAME=wordpress
    volumes:
      - wp_data:/var/www/html
    depends_on:
      db:
        condition: service_healthy

  db:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD:-rootpassword}
      - MYSQL_DATABASE=wordpress
      - MYSQL_USER=wordpress
      - MYSQL_PASSWORD=${MYSQL_PASSWORD:-wordpress}
    volumes:
      - db_data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  wp_data:
  db_data:
```

- [ ] **Step 2: Create wordpress-mysql/.dockerignore**

```
README.md
.git
```

- [ ] **Step 3: Create wordpress-mysql/README.md**

```markdown
# wordpress-mysql

WordPress と MySQL の公式 Docker イメージを使ったサンプルです。Dockerfile 不要で、`compose.yml` だけでデプロイできます。

## 構成

- WordPress (公式イメージ)
- MySQL 8.0 (公式イメージ)
- ポート: 80

## 前提条件

- conoha-cli がインストール済み
- ConoHa VPS3 アカウント
- SSH キーペア設定済み

## デプロイ

```bash
# サーバー作成（まだない場合）
conoha server create --name myserver --flavor g2l-t-2 --image ubuntu-24.04 --key mykey

# アプリ初期化
conoha app init myserver --app-name wordpress

# 環境変数を設定（パスワードを変更してください）
conoha app env set myserver --app-name wordpress \
  MYSQL_ROOT_PASSWORD=your_root_password \
  MYSQL_PASSWORD=your_wp_password

# デプロイ
conoha app deploy myserver --app-name wordpress
```

## 動作確認

ブラウザで `http://<サーバーIP>` にアクセスすると WordPress のセットアップ画面が表示されます。

## カスタマイズ

- テーマやプラグインは WordPress 管理画面からインストール
- 本番環境では必ず `conoha app env set` でパスワードを変更してください
- HTTPS が必要な場合はリバースプロキシ（nginx）を追加
```

- [ ] **Step 4: Commit**

```bash
git add wordpress-mysql/
git commit -m "Add WordPress + MySQL sample (official images only)"
```

---

### Task 7: Create GitHub repository and push

- [ ] **Step 1: Create the remote repository**

```bash
cd /home/tkim/dev/crowdy/conoha-cli-app-samples
gh repo create crowdy/conoha-cli-app-samples --public --source=. --description "Sample apps for conoha-cli app deploy"
```

- [ ] **Step 2: Push all commits**

```bash
git push -u origin main
```

- [ ] **Step 3: Verify on GitHub**

```bash
gh repo view crowdy/conoha-cli-app-samples --web
```
