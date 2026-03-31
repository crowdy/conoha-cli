# App Samples Tier 2 Additions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 5 Tier 2 samples (vite-react, sveltekit, go-fiber, nestjs-postgresql, rust-actix-web) to the existing `conoha-cli-app-samples` repo and update the root README.

**Architecture:** Each sample is a flat top-level directory following established patterns — compose.yml + Dockerfile + minimal source code + Japanese README. All code comments and app output in English. Each sample is independently deployable via `conoha app deploy`.

**Tech Stack:** Vite/React, SvelteKit, Go/Fiber, NestJS/TypeScript/PostgreSQL, Rust/Actix-web

**Repo:** `/home/tkim/dev/crowdy/conoha-cli-app-samples`

---

### Task 1: vite-react

Static React SPA built with Vite, served by nginx.

**Files:**
- Create: `vite-react/README.md`
- Create: `vite-react/Dockerfile`
- Create: `vite-react/compose.yml`
- Create: `vite-react/.dockerignore`
- Create: `vite-react/package.json`
- Create: `vite-react/vite.config.ts`
- Create: `vite-react/tsconfig.json`
- Create: `vite-react/index.html`
- Create: `vite-react/src/main.tsx`
- Create: `vite-react/src/App.tsx`
- Create: `vite-react/src/App.css`
- Create: `vite-react/src/vite-env.d.ts`
- Create: `vite-react/nginx.conf`

- [ ] **Step 1: Create vite-react/package.json**

```json
{
  "name": "conoha-vite-react-sample",
  "version": "1.0.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0"
  },
  "devDependencies": {
    "@types/react": "^19.0.0",
    "@types/react-dom": "^19.0.0",
    "@vitejs/plugin-react": "^4.4.1",
    "typescript": "^5.7.0",
    "vite": "^6.3.2"
  }
}
```

- [ ] **Step 2: Create vite-react/vite.config.ts**

```typescript
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
});
```

- [ ] **Step 3: Create vite-react/tsconfig.json**

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "moduleResolution": "bundler",
    "jsx": "react-jsx",
    "strict": true,
    "noEmit": true,
    "skipLibCheck": true,
    "isolatedModules": true
  },
  "include": ["src"]
}
```

- [ ] **Step 4: Create vite-react/index.html**

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>React on ConoHa</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- [ ] **Step 5: Create vite-react/src/vite-env.d.ts**

```typescript
/// <reference types="vite/client" />
```

- [ ] **Step 6: Create vite-react/src/main.tsx**

```tsx
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import "./App.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>
);
```

- [ ] **Step 7: Create vite-react/src/App.tsx**

```tsx
import { useState } from "react";

export default function App() {
  const [count, setCount] = useState(0);

  return (
    <div className="container">
      <h1>React on ConoHa</h1>
      <p>Deployed with <code>conoha app deploy</code></p>
      <div className="card">
        <button onClick={() => setCount((c) => c + 1)}>
          Count: {count}
        </button>
      </div>
    </div>
  );
}
```

- [ ] **Step 8: Create vite-react/src/App.css**

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
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
}

.container {
  text-align: center;
  padding: 2rem;
}

h1 {
  font-size: 2.5rem;
  margin-bottom: 1rem;
}

p {
  font-size: 1.2rem;
  color: #666;
  margin-bottom: 2rem;
}

.card {
  padding: 1rem;
}

button {
  padding: 0.75rem 2rem;
  font-size: 1.1rem;
  background: #1976d2;
  color: #fff;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s;
}

button:hover {
  background: #1565c0;
}
```

- [ ] **Step 9: Create vite-react/nginx.conf**

```nginx
server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

- [ ] **Step 10: Create vite-react/Dockerfile**

```dockerfile
# Stage 1: Build
FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json ./
RUN npm install
COPY . .
RUN npm run build

# Stage 2: Serve with nginx
FROM nginx:alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80
```

- [ ] **Step 11: Create vite-react/compose.yml**

```yaml
services:
  web:
    build: .
    ports:
      - "80:80"
```

- [ ] **Step 12: Create vite-react/.dockerignore**

```
README.md
.git
node_modules
dist
```

- [ ] **Step 13: Create vite-react/README.md**

```markdown
# vite-react

Vite + React で構築した SPA を nginx で配信するサンプルです。フロントエンドプロジェクトのデプロイに最適です。

## 構成

- Vite 6 + React 19 + TypeScript
- nginx（静的ファイル配信）
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
conoha app init myserver --app-name react-app

# デプロイ
conoha app deploy myserver --app-name react-app
```

## 動作確認

ブラウザで `http://<サーバーIP>` にアクセスするとカウンターアプリが表示されます。

## カスタマイズ

- `src/App.tsx` を編集してコンポーネントを変更
- `npm install` で追加パッケージをインストール
- `nginx.conf` でキャッシュ設定やリバースプロキシを追加
```

- [ ] **Step 14: Commit**

```bash
cd /home/tkim/dev/crowdy/conoha-cli-app-samples
git add vite-react/
git commit -m "Add Vite + React sample (static SPA with nginx)"
```

---

### Task 2: sveltekit

SvelteKit app with adapter-node for server-side rendering.

**Files:**
- Create: `sveltekit/README.md`
- Create: `sveltekit/Dockerfile`
- Create: `sveltekit/compose.yml`
- Create: `sveltekit/.dockerignore`
- Create: `sveltekit/package.json`
- Create: `sveltekit/svelte.config.js`
- Create: `sveltekit/vite.config.ts`
- Create: `sveltekit/tsconfig.json`
- Create: `sveltekit/src/app.html`
- Create: `sveltekit/src/app.css`
- Create: `sveltekit/src/routes/+page.svelte`
- Create: `sveltekit/src/routes/+layout.svelte`

- [ ] **Step 1: Create sveltekit/package.json**

```json
{
  "name": "conoha-sveltekit-sample",
  "version": "1.0.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite dev",
    "build": "vite build",
    "preview": "vite preview"
  },
  "devDependencies": {
    "@sveltejs/adapter-node": "^5.2.12",
    "@sveltejs/kit": "^2.20.7",
    "@sveltejs/vite-plugin-svelte": "^5.0.3",
    "svelte": "^5.25.12",
    "typescript": "^5.7.0",
    "vite": "^6.3.2"
  }
}
```

- [ ] **Step 2: Create sveltekit/svelte.config.js**

```javascript
import adapter from "@sveltejs/adapter-node";

/** @type {import('@sveltejs/kit').Config} */
const config = {
  kit: {
    adapter: adapter(),
  },
};

export default config;
```

- [ ] **Step 3: Create sveltekit/vite.config.ts**

```typescript
import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [sveltekit()],
});
```

- [ ] **Step 4: Create sveltekit/tsconfig.json**

```json
{
  "extends": "./.svelte-kit/tsconfig.json",
  "compilerOptions": {
    "allowJs": true,
    "checkJs": true,
    "esModuleInterop": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true,
    "skipLibCheck": true,
    "sourceMap": true,
    "strict": true,
    "moduleResolution": "bundler"
  }
}
```

- [ ] **Step 5: Create sveltekit/src/app.html**

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    %sveltekit.head%
  </head>
  <body>
    <div>%sveltekit.body%</div>
  </body>
</html>
```

- [ ] **Step 6: Create sveltekit/src/app.css**

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
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
}
```

- [ ] **Step 7: Create sveltekit/src/routes/+layout.svelte**

```svelte
<script>
  import "../app.css";
  let { children } = $props();
</script>

{@render children()}
```

- [ ] **Step 8: Create sveltekit/src/routes/+page.svelte**

```svelte
<script lang="ts">
  let count = $state(0);
</script>

<svelte:head>
  <title>SvelteKit on ConoHa</title>
</svelte:head>

<div class="container">
  <h1>SvelteKit on ConoHa</h1>
  <p>Deployed with <code>conoha app deploy</code></p>
  <div class="card">
    <button onclick={() => count++}>
      Count: {count}
    </button>
  </div>
</div>

<style>
  .container {
    text-align: center;
    padding: 2rem;
  }
  h1 {
    font-size: 2.5rem;
    margin-bottom: 1rem;
  }
  p {
    font-size: 1.2rem;
    color: #666;
    margin-bottom: 2rem;
  }
  .card {
    padding: 1rem;
  }
  button {
    padding: 0.75rem 2rem;
    font-size: 1.1rem;
    background: #ff3e00;
    color: #fff;
    border: none;
    border-radius: 8px;
    cursor: pointer;
    transition: background 0.2s;
  }
  button:hover {
    background: #d63600;
  }
</style>
```

- [ ] **Step 9: Create sveltekit/Dockerfile**

```dockerfile
# Stage 1: Build
FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json ./
RUN npm install
COPY . .
RUN npm run build

# Stage 2: Production runner
FROM node:22-alpine
WORKDIR /app
COPY --from=builder /app/build ./build
COPY --from=builder /app/package.json ./
RUN npm install --omit=dev
EXPOSE 3000
ENV PORT=3000
CMD ["node", "build"]
```

- [ ] **Step 10: Create sveltekit/compose.yml**

```yaml
services:
  web:
    build: .
    ports:
      - "3000:3000"
```

- [ ] **Step 11: Create sveltekit/.dockerignore**

```
README.md
.git
node_modules
build
.svelte-kit
```

- [ ] **Step 12: Create sveltekit/README.md**

```markdown
# sveltekit

SvelteKit アプリを adapter-node でデプロイするサンプルです。SSR 対応のモダンフレームワークです。

## 構成

- SvelteKit 2 + Svelte 5 + TypeScript
- adapter-node（Node.js サーバー）
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
conoha app init myserver --app-name sveltekit-app

# デプロイ
conoha app deploy myserver --app-name sveltekit-app
```

## 動作確認

ブラウザで `http://<サーバーIP>:3000` にアクセスするとカウンターアプリが表示されます。

## カスタマイズ

- `src/routes/` にファイルを追加してルーティング
- `svelte.config.js` で SvelteKit の設定を変更
- 静的サイトにしたい場合は `adapter-static` に変更
```

- [ ] **Step 13: Commit**

```bash
cd /home/tkim/dev/crowdy/conoha-cli-app-samples
git add sveltekit/
git commit -m "Add SvelteKit sample (SSR with adapter-node)"
```

---

### Task 3: go-fiber

Minimal Go REST API with Fiber framework.

**Files:**
- Create: `go-fiber/README.md`
- Create: `go-fiber/Dockerfile`
- Create: `go-fiber/compose.yml`
- Create: `go-fiber/.dockerignore`
- Create: `go-fiber/go.mod`
- Create: `go-fiber/go.sum` (empty)
- Create: `go-fiber/main.go`

- [ ] **Step 1: Create go-fiber/go.mod**

```
module conoha-go-fiber-sample

go 1.24

require github.com/gofiber/fiber/v2 v2.52.6

require (
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.58.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
)
```

- [ ] **Step 2: Create go-fiber/go.sum (empty)**

Create an empty file. Docker build will run `go mod download` which generates the real sum.

- [ ] **Step 3: Create go-fiber/main.go**

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Message struct {
	ID        int       `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

var messages []Message
var nextID = 1

func main() {
	app := fiber.New()

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// List messages
	app.Get("/api/messages", func(c *fiber.Ctx) error {
		return c.JSON(messages)
	})

	// Create message
	app.Post("/api/messages", func(c *fiber.Ctx) error {
		var body struct {
			Text string `json:"text"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
		}
		if body.Text == "" {
			return c.Status(400).JSON(fiber.Map{"error": "text is required"})
		}
		msg := Message{
			ID:        nextID,
			Text:      body.Text,
			CreatedAt: time.Now(),
		}
		nextID++
		messages = append(messages, msg)
		return c.Status(201).JSON(msg)
	})

	// Delete message
	app.Delete("/api/messages/:id", func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
		}
		for i, msg := range messages {
			if msg.ID == id {
				messages = append(messages[:i], messages[i+1:]...)
				return c.SendStatus(204)
			}
		}
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	})

	// Serve index page
	app.Get("/", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		return c.SendString(indexHTML)
	})

	fmt.Println("Server running on port 3000")
	log.Fatal(app.Listen(":3000"))
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Go Fiber on ConoHa</title>
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
    .msg { background: #fff; padding: 1rem; border-radius: 8px; margin-bottom: 0.5rem; display: flex; justify-content: space-between; align-items: center; }
    .form-box { background: #fff; padding: 1rem; border-radius: 8px; margin-bottom: 2rem; display: flex; gap: 0.5rem; }
    input { flex: 1; padding: 0.5rem; border: 1px solid #ddd; border-radius: 4px; font-size: 1rem; }
    button { padding: 0.5rem 1.5rem; background: #00acd7; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 1rem; }
    .delete { background: #d32f2f; font-size: 0.85rem; padding: 0.3rem 0.8rem; }
  </style>
</head>
<body>
  <h1>Go Fiber on ConoHa</h1>
  <div class="form-box">
    <input type="text" id="input" placeholder="Type a message..." required>
    <button onclick="send()">Send</button>
  </div>
  <div id="list"></div>
  <script>
    async function load() {
      const res = await fetch("/api/messages");
      const msgs = await res.json();
      const list = document.getElementById("list");
      list.innerHTML = (msgs || []).map(m =>
        '<div class="msg"><span>' + m.text + '</span>' +
        '<button class="delete" onclick="del(' + m.id + ')">Delete</button></div>'
      ).join("");
    }
    async function send() {
      const input = document.getElementById("input");
      const text = input.value.trim();
      if (!text) return;
      await fetch("/api/messages", {
        method: "POST",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify({text})
      });
      input.value = "";
      load();
    }
    async function del(id) {
      await fetch("/api/messages/" + id, {method: "DELETE"});
      load();
    }
    document.getElementById("input").addEventListener("keydown", e => {
      if (e.key === "Enter") send();
    });
    load();
  </script>
</body>
</html>`
```

- [ ] **Step 4: Create go-fiber/Dockerfile**

```dockerfile
# Stage 1: Build
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server .

# Stage 2: Production runner
FROM alpine:3.21
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 3000
CMD ["./server"]
```

- [ ] **Step 5: Create go-fiber/compose.yml**

```yaml
services:
  web:
    build: .
    ports:
      - "3000:3000"
```

- [ ] **Step 6: Create go-fiber/.dockerignore**

```
README.md
.git
server
```

- [ ] **Step 7: Create go-fiber/README.md**

```markdown
# go-fiber

Go と Fiber フレームワークで構築した高速 REST API サーバーです。インメモリでメッセージの CRUD を行います。

## 構成

- Go 1.24 + Fiber v2
- ポート: 3000

## 前提条件

- conoha-cli がインストール済み
- ConoHa VPS3 アカウント
- SSH キーペア設定済み

## デプロイ

```bash
# サーバー作成（まだない場合）
conoha server create --name myserver --flavor g2l-t-1 --image ubuntu-24.04 --key mykey

# アプリ初期化
conoha app init myserver --app-name go-api

# デプロイ
conoha app deploy myserver --app-name go-api
```

## 動作確認

ブラウザで `http://<サーバーIP>:3000` にアクセスするとメッセージボードが表示されます。

API エンドポイント:
- `GET /api/messages` — メッセージ一覧
- `POST /api/messages` — メッセージ作成（`{"text": "hello"}`）
- `DELETE /api/messages/:id` — メッセージ削除
- `GET /health` — ヘルスチェック

## カスタマイズ

- `main.go` にルートを追加して機能を拡張
- データベースを追加する場合は GORM などの ORM を導入
- バイナリサイズが小さいため起動が非常に高速
```

- [ ] **Step 8: Commit**

```bash
cd /home/tkim/dev/crowdy/conoha-cli-app-samples
git add go-fiber/
git commit -m "Add Go Fiber sample (high-performance REST API)"
```

---

### Task 4: nestjs-postgresql

NestJS TypeScript backend with TypeORM and PostgreSQL.

**Files:**
- Create: `nestjs-postgresql/README.md`
- Create: `nestjs-postgresql/Dockerfile`
- Create: `nestjs-postgresql/compose.yml`
- Create: `nestjs-postgresql/.dockerignore`
- Create: `nestjs-postgresql/package.json`
- Create: `nestjs-postgresql/tsconfig.json`
- Create: `nestjs-postgresql/tsconfig.build.json`
- Create: `nestjs-postgresql/nest-cli.json`
- Create: `nestjs-postgresql/src/main.ts`
- Create: `nestjs-postgresql/src/app.module.ts`
- Create: `nestjs-postgresql/src/post.entity.ts`
- Create: `nestjs-postgresql/src/posts.controller.ts`
- Create: `nestjs-postgresql/src/posts.service.ts`
- Create: `nestjs-postgresql/views/index.hbs`

- [ ] **Step 1: Create nestjs-postgresql/package.json**

```json
{
  "name": "conoha-nestjs-sample",
  "version": "1.0.0",
  "private": true,
  "scripts": {
    "build": "nest build",
    "start": "nest start",
    "start:prod": "node dist/main"
  },
  "dependencies": {
    "@nestjs/common": "^11.0.0",
    "@nestjs/core": "^11.0.0",
    "@nestjs/platform-express": "^11.0.0",
    "@nestjs/typeorm": "^11.0.0",
    "hbs": "^4.2.0",
    "pg": "^8.13.0",
    "reflect-metadata": "^0.2.2",
    "rxjs": "^7.8.0",
    "typeorm": "^0.3.20"
  },
  "devDependencies": {
    "@nestjs/cli": "^11.0.0",
    "typescript": "^5.7.0"
  }
}
```

- [ ] **Step 2: Create nestjs-postgresql/tsconfig.json**

```json
{
  "compilerOptions": {
    "module": "commonjs",
    "declaration": true,
    "removeComments": true,
    "emitDecoratorMetadata": true,
    "experimentalDecorators": true,
    "target": "ES2021",
    "sourceMap": true,
    "outDir": "./dist",
    "baseUrl": "./",
    "incremental": true,
    "skipLibCheck": true,
    "strict": true,
    "forceConsistentCasingInFileNames": true,
    "noFallthroughCasesInSwitch": true
  }
}
```

- [ ] **Step 3: Create nestjs-postgresql/tsconfig.build.json**

```json
{
  "extends": "./tsconfig.json",
  "exclude": ["node_modules", "dist", "test"]
}
```

- [ ] **Step 4: Create nestjs-postgresql/nest-cli.json**

```json
{
  "$schema": "https://json.schemastore.org/nest-cli",
  "collection": "@nestjs/schematics",
  "sourceRoot": "src"
}
```

- [ ] **Step 5: Create nestjs-postgresql/src/post.entity.ts**

```typescript
import { Entity, PrimaryGeneratedColumn, Column, CreateDateColumn } from "typeorm";

@Entity("posts")
export class Post {
  @PrimaryGeneratedColumn()
  id: number;

  @Column()
  title: string;

  @Column({ type: "text", nullable: true })
  body: string;

  @CreateDateColumn()
  createdAt: Date;
}
```

- [ ] **Step 6: Create nestjs-postgresql/src/posts.service.ts**

```typescript
import { Injectable } from "@nestjs/common";
import { InjectRepository } from "@nestjs/typeorm";
import { Repository } from "typeorm";
import { Post } from "./post.entity";

@Injectable()
export class PostsService {
  constructor(
    @InjectRepository(Post)
    private readonly repo: Repository<Post>,
  ) {}

  findAll(): Promise<Post[]> {
    return this.repo.find({ order: { createdAt: "DESC" } });
  }

  create(title: string, body: string): Promise<Post> {
    const post = this.repo.create({ title, body });
    return this.repo.save(post);
  }

  async remove(id: number): Promise<void> {
    await this.repo.delete(id);
  }
}
```

- [ ] **Step 7: Create nestjs-postgresql/src/posts.controller.ts**

```typescript
import { Body, Controller, Get, Post as HttpPost, Param, Render, Redirect } from "@nestjs/common";
import { PostsService } from "./posts.service";

@Controller()
export class PostsController {
  constructor(private readonly postsService: PostsService) {}

  @Get()
  @Render("index")
  async index() {
    const posts = await this.postsService.findAll();
    return { posts };
  }

  @HttpPost("posts")
  @Redirect("/")
  async create(@Body() body: { title: string; body: string }) {
    await this.postsService.create(body.title, body.body);
  }

  @HttpPost("posts/:id/delete")
  @Redirect("/")
  async remove(@Param("id") id: string) {
    await this.postsService.remove(Number(id));
  }
}
```

- [ ] **Step 8: Create nestjs-postgresql/src/app.module.ts**

```typescript
import { Module } from "@nestjs/common";
import { TypeOrmModule } from "@nestjs/typeorm";
import { Post } from "./post.entity";
import { PostsController } from "./posts.controller";
import { PostsService } from "./posts.service";

@Module({
  imports: [
    TypeOrmModule.forRoot({
      type: "postgres",
      host: process.env.DB_HOST || "db",
      port: 5432,
      username: process.env.DB_USER || "postgres",
      password: process.env.DB_PASSWORD || "postgres",
      database: process.env.DB_NAME || "app_production",
      entities: [Post],
      synchronize: true,
    }),
    TypeOrmModule.forFeature([Post]),
  ],
  controllers: [PostsController],
  providers: [PostsService],
})
export class AppModule {}
```

- [ ] **Step 9: Create nestjs-postgresql/src/main.ts**

```typescript
import { NestFactory } from "@nestjs/core";
import { NestExpressApplication } from "@nestjs/platform-express";
import { join } from "path";
import { AppModule } from "./app.module";

async function bootstrap() {
  const app = await NestFactory.create<NestExpressApplication>(AppModule);
  app.setBaseViewsDir(join(__dirname, "..", "views"));
  app.setViewEngine("hbs");
  await app.listen(3000);
  console.log("Server running on port 3000");
}
bootstrap();
```

- [ ] **Step 10: Create nestjs-postgresql/views/index.hbs**

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>NestJS on ConoHa</title>
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
    .form-box { background: #fff; padding: 1rem; border-radius: 8px; margin-bottom: 2rem; }
    input, textarea { width: 100%; padding: 0.5rem; margin-bottom: 0.5rem; border: 1px solid #ddd; border-radius: 4px; font-size: 1rem; box-sizing: border-box; }
    textarea { height: 80px; resize: vertical; }
    button { padding: 0.5rem 1.5rem; background: #e0234e; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 1rem; }
    .delete { background: #d32f2f; font-size: 0.85rem; padding: 0.3rem 0.8rem; }
    form.inline { display: inline; }
  </style>
</head>
<body>
  <h1>NestJS on ConoHa</h1>
  <div class="form-box">
    <form action="/posts" method="post">
      <input type="text" name="title" placeholder="Title" required>
      <textarea name="body" placeholder="Body (optional)"></textarea>
      <button type="submit">Create Post</button>
    </form>
  </div>
  {{#each posts}}
    <div class="post">
      <h2>{{this.title}}</h2>
      <p>{{this.body}}</p>
      <form action="/posts/{{this.id}}/delete" method="post" class="inline">
        <button type="submit" class="delete">Delete</button>
      </form>
    </div>
  {{/each}}
</body>
</html>
```

- [ ] **Step 11: Create nestjs-postgresql/Dockerfile**

```dockerfile
# Stage 1: Build
FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json ./
RUN npm install
COPY . .
RUN npm run build

# Stage 2: Production runner
FROM node:22-alpine
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/views ./views
COPY --from=builder /app/package.json ./
RUN npm install --omit=dev
EXPOSE 3000
CMD ["node", "dist/main"]
```

- [ ] **Step 12: Create nestjs-postgresql/compose.yml**

```yaml
services:
  web:
    build: .
    ports:
      - "3000:3000"
    environment:
      - DB_HOST=db
      - DB_USER=postgres
      - DB_PASSWORD=postgres
      - DB_NAME=app_production
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

- [ ] **Step 13: Create nestjs-postgresql/.dockerignore**

```
README.md
.git
node_modules
dist
```

- [ ] **Step 14: Create nestjs-postgresql/README.md**

```markdown
# nestjs-postgresql

NestJS と PostgreSQL を使ったシンプルな投稿アプリです。TypeORM による CRUD 機能を持ちます。

## 構成

- Node.js 22 + NestJS 11 + TypeScript
- TypeORM + PostgreSQL 17
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
conoha app init myserver --app-name nestjs-app

# デプロイ
conoha app deploy myserver --app-name nestjs-app
```

テーブルはアプリ起動時に自動作成されます（TypeORM synchronize）。

## 動作確認

ブラウザで `http://<サーバーIP>:3000` にアクセスすると投稿一覧ページが表示されます。

## カスタマイズ

- `src/` にモジュール・コントローラー・サービスを追加
- `nest g resource <name>` でリソース一式を自動生成
- 本番環境では `DB_PASSWORD` を `.env.server` で管理
- `synchronize: true` は開発用。本番ではマイグレーションを使用
```

- [ ] **Step 15: Commit**

```bash
cd /home/tkim/dev/crowdy/conoha-cli-app-samples
git add nestjs-postgresql/
git commit -m "Add NestJS + PostgreSQL sample (TypeORM CRUD app)"
```

---

### Task 5: rust-actix-web

Minimal Rust REST API with Actix-web.

**Files:**
- Create: `rust-actix-web/README.md`
- Create: `rust-actix-web/Dockerfile`
- Create: `rust-actix-web/compose.yml`
- Create: `rust-actix-web/.dockerignore`
- Create: `rust-actix-web/Cargo.toml`
- Create: `rust-actix-web/src/main.rs`

- [ ] **Step 1: Create rust-actix-web/Cargo.toml**

```toml
[package]
name = "conoha-rust-sample"
version = "1.0.0"
edition = "2024"

[dependencies]
actix-web = "4"
serde = { version = "1", features = ["derive"] }
serde_json = "1"
tokio = { version = "1", features = ["macros", "rt-multi-thread"] }
```

- [ ] **Step 2: Create rust-actix-web/src/main.rs**

```rust
use actix_web::{web, App, HttpServer, HttpResponse, Responder};
use serde::{Deserialize, Serialize};
use std::sync::Mutex;

#[derive(Serialize, Clone)]
struct Message {
    id: u32,
    text: String,
}

#[derive(Deserialize)]
struct CreateMessage {
    text: String,
}

struct AppState {
    messages: Mutex<Vec<Message>>,
    next_id: Mutex<u32>,
}

async fn index() -> impl Responder {
    HttpResponse::Ok().content_type("text/html").body(INDEX_HTML)
}

async fn list_messages(data: web::Data<AppState>) -> impl Responder {
    let messages = data.messages.lock().unwrap();
    HttpResponse::Ok().json(&*messages)
}

async fn create_message(
    data: web::Data<AppState>,
    body: web::Json<CreateMessage>,
) -> impl Responder {
    if body.text.is_empty() {
        return HttpResponse::BadRequest().json(serde_json::json!({"error": "text is required"}));
    }
    let mut messages = data.messages.lock().unwrap();
    let mut next_id = data.next_id.lock().unwrap();
    let msg = Message {
        id: *next_id,
        text: body.text.clone(),
    };
    *next_id += 1;
    messages.push(msg.clone());
    HttpResponse::Created().json(msg)
}

async fn delete_message(
    data: web::Data<AppState>,
    path: web::Path<u32>,
) -> impl Responder {
    let id = path.into_inner();
    let mut messages = data.messages.lock().unwrap();
    if let Some(pos) = messages.iter().position(|m| m.id == id) {
        messages.remove(pos);
        HttpResponse::NoContent().finish()
    } else {
        HttpResponse::NotFound().json(serde_json::json!({"error": "not found"}))
    }
}

async fn health() -> impl Responder {
    HttpResponse::Ok().json(serde_json::json!({"status": "ok"}))
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    let data = web::Data::new(AppState {
        messages: Mutex::new(Vec::new()),
        next_id: Mutex::new(1),
    });

    println!("Server running on port 3000");

    HttpServer::new(move || {
        App::new()
            .app_data(data.clone())
            .route("/", web::get().to(index))
            .route("/health", web::get().to(health))
            .route("/api/messages", web::get().to(list_messages))
            .route("/api/messages", web::post().to(create_message))
            .route("/api/messages/{id}", web::delete().to(delete_message))
    })
    .bind("0.0.0.0:3000")?
    .run()
    .await
}

const INDEX_HTML: &str = r#"<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Rust Actix on ConoHa</title>
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
    .msg { background: #fff; padding: 1rem; border-radius: 8px; margin-bottom: 0.5rem; display: flex; justify-content: space-between; align-items: center; }
    .form-box { background: #fff; padding: 1rem; border-radius: 8px; margin-bottom: 2rem; display: flex; gap: 0.5rem; }
    input { flex: 1; padding: 0.5rem; border: 1px solid #ddd; border-radius: 4px; font-size: 1rem; }
    button { padding: 0.5rem 1.5rem; background: #b7410e; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 1rem; }
    .delete { background: #d32f2f; font-size: 0.85rem; padding: 0.3rem 0.8rem; }
  </style>
</head>
<body>
  <h1>Rust Actix on ConoHa</h1>
  <div class="form-box">
    <input type="text" id="input" placeholder="Type a message..." required>
    <button onclick="send()">Send</button>
  </div>
  <div id="list"></div>
  <script>
    async function load() {
      const res = await fetch("/api/messages");
      const msgs = await res.json();
      document.getElementById("list").innerHTML = msgs.map(m =>
        '<div class="msg"><span>' + m.text + '</span>' +
        '<button class="delete" onclick="del(' + m.id + ')">Delete</button></div>'
      ).join("");
    }
    async function send() {
      const input = document.getElementById("input");
      const text = input.value.trim();
      if (!text) return;
      await fetch("/api/messages", {
        method: "POST",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify({text})
      });
      input.value = "";
      load();
    }
    async function del(id) {
      await fetch("/api/messages/" + id, {method: "DELETE"});
      load();
    }
    document.getElementById("input").addEventListener("keydown", e => {
      if (e.key === "Enter") send();
    });
    load();
  </script>
</body>
</html>"#;
```

- [ ] **Step 3: Create rust-actix-web/Dockerfile**

```dockerfile
# Stage 1: Build
FROM rust:1.86-alpine AS builder
WORKDIR /app
RUN apk add --no-cache musl-dev
COPY Cargo.toml ./
RUN mkdir src && echo "fn main() {}" > src/main.rs && cargo build --release && rm -rf src
COPY src ./src
RUN touch src/main.rs && cargo build --release

# Stage 2: Production runner
FROM alpine:3.21
WORKDIR /app
COPY --from=builder /app/target/release/conoha-rust-sample ./server
EXPOSE 3000
CMD ["./server"]
```

- [ ] **Step 4: Create rust-actix-web/compose.yml**

```yaml
services:
  web:
    build: .
    ports:
      - "3000:3000"
```

- [ ] **Step 5: Create rust-actix-web/.dockerignore**

```
README.md
.git
target
```

- [ ] **Step 6: Create rust-actix-web/README.md**

```markdown
# rust-actix-web

Rust と Actix-web で構築した高速 REST API サーバーです。インメモリでメッセージの CRUD を行います。

## 構成

- Rust 1.86 + Actix-web 4
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
conoha app init myserver --app-name rust-api

# デプロイ
conoha app deploy myserver --app-name rust-api
```

初回ビルドは Rust コンパイルに数分かかります。2回目以降はキャッシュで高速化されます。

## 動作確認

ブラウザで `http://<サーバーIP>:3000` にアクセスするとメッセージボードが表示されます。

API エンドポイント:
- `GET /api/messages` — メッセージ一覧
- `POST /api/messages` — メッセージ作成（`{"text": "hello"}`）
- `DELETE /api/messages/:id` — メッセージ削除
- `GET /health` — ヘルスチェック

## カスタマイズ

- `src/main.rs` にルートを追加して機能を拡張
- データベースを追加する場合は Diesel や SQLx を導入
- バイナリサイズが非常に小さくメモリ効率が高い
```

- [ ] **Step 7: Commit**

```bash
cd /home/tkim/dev/crowdy/conoha-cli-app-samples
git add rust-actix-web/
git commit -m "Add Rust Actix-web sample (high-performance REST API)"
```

---

### Task 6: Update root README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add 5 new rows to the samples table**

Add after the django-postgresql row:

```markdown
| [vite-react](vite-react/) | Vite + React (静的SPA) | カウンターアプリ | g2l-t-1 (1GB) |
| [sveltekit](sveltekit/) | SvelteKit (SSR) | カウンターアプリ | g2l-t-2 (2GB) |
| [go-fiber](go-fiber/) | Go + Fiber | 高速 REST API | g2l-t-1 (1GB) |
| [nestjs-postgresql](nestjs-postgresql/) | NestJS + PostgreSQL | TypeORM CRUD アプリ | g2l-t-2 (2GB) |
| [rust-actix-web](rust-actix-web/) | Rust + Actix-web | 高速 REST API | g2l-t-2 (2GB) |
```

- [ ] **Step 2: Commit**

```bash
cd /home/tkim/dev/crowdy/conoha-cli-app-samples
git add README.md
git commit -m "Update root README with Tier 2 samples"
```

---

### Task 7: Push to GitHub

- [ ] **Step 1: Push all new commits**

```bash
cd /home/tkim/dev/crowdy/conoha-cli-app-samples
git push origin main
```
