# App Samples Tier 1 Additions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 4 Tier 1 samples (spring-boot-postgresql, express-mongodb, laravel-mysql, django-postgresql) to the existing `conoha-cli-app-samples` repo and update the root README.

**Architecture:** Each sample is a flat top-level directory following established patterns — compose.yml + Dockerfile + minimal source code + Japanese README. All code comments and app output in English. Each sample is independently deployable via `conoha app deploy`.

**Tech Stack:** Spring Boot 3 / Java 21, Express.js / MongoDB, Laravel 11 / PHP 8.3 / MySQL, Django 5 / PostgreSQL

**Repo:** `/home/tkim/dev/crowdy/conoha-cli-app-samples`

---

### Task 1: spring-boot-postgresql

**Files:**
- Create: `spring-boot-postgresql/README.md`
- Create: `spring-boot-postgresql/Dockerfile`
- Create: `spring-boot-postgresql/compose.yml`
- Create: `spring-boot-postgresql/.dockerignore`
- Create: `spring-boot-postgresql/pom.xml`
- Create: `spring-boot-postgresql/src/main/java/com/example/app/Application.java`
- Create: `spring-boot-postgresql/src/main/java/com/example/app/Post.java`
- Create: `spring-boot-postgresql/src/main/java/com/example/app/PostRepository.java`
- Create: `spring-boot-postgresql/src/main/java/com/example/app/PostController.java`
- Create: `spring-boot-postgresql/src/main/resources/application.properties`
- Create: `spring-boot-postgresql/src/main/resources/templates/index.html`

- [ ] **Step 1: Create spring-boot-postgresql/pom.xml**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    <parent>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-parent</artifactId>
        <version>3.4.4</version>
    </parent>
    <groupId>com.example</groupId>
    <artifactId>conoha-spring-sample</artifactId>
    <version>1.0.0</version>
    <properties>
        <java.version>21</java.version>
    </properties>
    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
        </dependency>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-data-jpa</artifactId>
        </dependency>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-thymeleaf</artifactId>
        </dependency>
        <dependency>
            <groupId>org.postgresql</groupId>
            <artifactId>postgresql</artifactId>
            <scope>runtime</scope>
        </dependency>
    </dependencies>
    <build>
        <plugins>
            <plugin>
                <groupId>org.springframework.boot</groupId>
                <artifactId>spring-boot-maven-plugin</artifactId>
            </plugin>
        </plugins>
    </build>
</project>
```

- [ ] **Step 2: Create spring-boot-postgresql/src/main/resources/application.properties**

```properties
spring.datasource.url=jdbc:postgresql://${DB_HOST:db}:5432/${DB_NAME:app_production}
spring.datasource.username=${DB_USER:postgres}
spring.datasource.password=${DB_PASSWORD:postgres}
spring.jpa.hibernate.ddl-auto=update
spring.jpa.open-in-view=false
server.port=8080
```

- [ ] **Step 3: Create spring-boot-postgresql/src/main/java/com/example/app/Application.java**

```java
package com.example.app;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

@SpringBootApplication
public class Application {
    public static void main(String[] args) {
        SpringApplication.run(Application.class, args);
    }
}
```

- [ ] **Step 4: Create spring-boot-postgresql/src/main/java/com/example/app/Post.java**

```java
package com.example.app;

import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Table;

@Entity
@Table(name = "posts")
public class Post {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;
    private String title;
    private String body;

    public Post() {}

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getTitle() { return title; }
    public void setTitle(String title) { this.title = title; }
    public String getBody() { return body; }
    public void setBody(String body) { this.body = body; }
}
```

- [ ] **Step 5: Create spring-boot-postgresql/src/main/java/com/example/app/PostRepository.java**

```java
package com.example.app;

import org.springframework.data.jpa.repository.JpaRepository;

public interface PostRepository extends JpaRepository<Post, Long> {
}
```

- [ ] **Step 6: Create spring-boot-postgresql/src/main/java/com/example/app/PostController.java**

```java
package com.example.app;

import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PathVariable;

@Controller
public class PostController {
    private final PostRepository repository;

    public PostController(PostRepository repository) {
        this.repository = repository;
    }

    @GetMapping("/")
    public String index(Model model) {
        model.addAttribute("posts", repository.findAll());
        model.addAttribute("post", new Post());
        return "index";
    }

    @PostMapping("/posts")
    public String create(Post post) {
        repository.save(post);
        return "redirect:/";
    }

    @PostMapping("/posts/{id}/delete")
    public String delete(@PathVariable Long id) {
        repository.deleteById(id);
        return "redirect:/";
    }
}
```

- [ ] **Step 7: Create spring-boot-postgresql/src/main/resources/templates/index.html**

```html
<!DOCTYPE html>
<html lang="en" xmlns:th="http://www.thymeleaf.org">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Spring Boot on ConoHa</title>
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
    form.inline { display: inline; }
    .form-box { background: #fff; padding: 1rem; border-radius: 8px; margin-bottom: 2rem; }
    input, textarea { width: 100%; padding: 0.5rem; margin-bottom: 0.5rem; border: 1px solid #ddd; border-radius: 4px; font-size: 1rem; box-sizing: border-box; }
    textarea { height: 80px; resize: vertical; }
    button { padding: 0.5rem 1.5rem; background: #1976d2; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 1rem; }
    .delete { background: #d32f2f; font-size: 0.85rem; padding: 0.3rem 0.8rem; }
  </style>
</head>
<body>
  <h1>Spring Boot on ConoHa</h1>
  <div class="form-box">
    <form th:action="@{/posts}" method="post" th:object="${post}">
      <input type="text" th:field="*{title}" placeholder="Title" required>
      <textarea th:field="*{body}" placeholder="Body (optional)"></textarea>
      <button type="submit">Create Post</button>
    </form>
  </div>
  <div th:each="p : ${posts}" class="post">
    <h2 th:text="${p.title}">Title</h2>
    <p th:text="${p.body}">Body</p>
    <form th:action="@{/posts/{id}/delete(id=${p.id})}" method="post" class="inline">
      <button type="submit" class="delete">Delete</button>
    </form>
  </div>
</body>
</html>
```

- [ ] **Step 8: Create spring-boot-postgresql/Dockerfile**

```dockerfile
# Stage 1: Build with Maven
FROM eclipse-temurin:21-jdk-alpine AS builder
WORKDIR /app
COPY pom.xml .
RUN apk add --no-cache maven && mvn dependency:go-offline -B
COPY src ./src
RUN mvn package -DskipTests -B

# Stage 2: Production runner
FROM eclipse-temurin:21-jre-alpine
WORKDIR /app
COPY --from=builder /app/target/*.jar app.jar
EXPOSE 8080
CMD ["java", "-jar", "app.jar"]
```

- [ ] **Step 9: Create spring-boot-postgresql/compose.yml**

```yaml
services:
  web:
    build: .
    ports:
      - "8080:8080"
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

- [ ] **Step 10: Create spring-boot-postgresql/.dockerignore**

```
README.md
.git
target
.idea
*.iml
```

- [ ] **Step 11: Create spring-boot-postgresql/README.md**

```markdown
# spring-boot-postgresql

Spring Boot と PostgreSQL を使ったシンプルな投稿アプリです。JPA による CRUD 機能を持ちます。

## 構成

- Java 21 + Spring Boot 3.4
- PostgreSQL 17
- ポート: 8080

## 前提条件

- conoha-cli がインストール済み
- ConoHa VPS3 アカウント
- SSH キーペア設定済み

## デプロイ

```bash
# サーバー作成（まだない場合）
conoha server create --name myserver --flavor g2l-t-2 --image ubuntu-24.04 --key mykey

# アプリ初期化
conoha app init myserver --app-name spring-app

# デプロイ
conoha app deploy myserver --app-name spring-app
```

初回ビルドは Maven 依存関係のダウンロードに数分かかります。

## 動作確認

ブラウザで `http://<サーバーIP>:8080` にアクセスすると投稿一覧ページが表示されます。

## カスタマイズ

- `src/main/java/com/example/app/` にエンティティやコントローラーを追加
- `src/main/resources/templates/` に Thymeleaf テンプレートを追加
- 本番環境では `DB_PASSWORD` を `.env.server` で管理
```

- [ ] **Step 12: Commit**

```bash
cd /home/tkim/dev/crowdy/conoha-cli-app-samples
git add spring-boot-postgresql/
git commit -m "Add Spring Boot + PostgreSQL sample (JPA CRUD app)"
```

---

### Task 2: express-mongodb

**Files:**
- Create: `express-mongodb/README.md`
- Create: `express-mongodb/Dockerfile`
- Create: `express-mongodb/compose.yml`
- Create: `express-mongodb/.dockerignore`
- Create: `express-mongodb/package.json`
- Create: `express-mongodb/app.js`
- Create: `express-mongodb/views/index.ejs`

- [ ] **Step 1: Create express-mongodb/package.json**

```json
{
  "name": "conoha-express-sample",
  "version": "1.0.0",
  "private": true,
  "scripts": {
    "start": "node app.js"
  },
  "dependencies": {
    "express": "^5.1.0",
    "ejs": "^3.1.10",
    "mongoose": "^8.13.2"
  }
}
```

- [ ] **Step 2: Create express-mongodb/app.js**

```javascript
const express = require("express");
const mongoose = require("mongoose");

const app = express();
const PORT = 3000;

app.set("view engine", "ejs");
app.use(express.urlencoded({ extended: true }));

// Connect to MongoDB
const mongoURL = process.env.MONGO_URL || "mongodb://db:27017/app";
mongoose.connect(mongoURL);

// Post schema
const postSchema = new mongoose.Schema({
  title: { type: String, required: true },
  body: String,
  createdAt: { type: Date, default: Date.now },
});
const Post = mongoose.model("Post", postSchema);

// Routes
app.get("/", async (req, res) => {
  const posts = await Post.find().sort({ createdAt: -1 });
  res.render("index", { posts });
});

app.post("/posts", async (req, res) => {
  await Post.create({ title: req.body.title, body: req.body.body });
  res.redirect("/");
});

app.post("/posts/:id/delete", async (req, res) => {
  await Post.findByIdAndDelete(req.params.id);
  res.redirect("/");
});

app.get("/health", (req, res) => {
  res.json({ status: "ok" });
});

app.listen(PORT, () => {
  console.log(`Server running on port ${PORT}`);
});
```

- [ ] **Step 3: Create express-mongodb/views/index.ejs**

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Express on ConoHa</title>
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
    button { padding: 0.5rem 1.5rem; background: #1976d2; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 1rem; }
    .delete { background: #d32f2f; font-size: 0.85rem; padding: 0.3rem 0.8rem; }
    form.inline { display: inline; }
  </style>
</head>
<body>
  <h1>Express on ConoHa</h1>
  <div class="form-box">
    <form action="/posts" method="post">
      <input type="text" name="title" placeholder="Title" required>
      <textarea name="body" placeholder="Body (optional)"></textarea>
      <button type="submit">Create Post</button>
    </form>
  </div>
  <% posts.forEach(post => { %>
    <div class="post">
      <h2><%= post.title %></h2>
      <p><%= post.body %></p>
      <form action="/posts/<%= post._id %>/delete" method="post" class="inline">
        <button type="submit" class="delete">Delete</button>
      </form>
    </div>
  <% }) %>
</body>
</html>
```

- [ ] **Step 4: Create express-mongodb/Dockerfile**

```dockerfile
FROM node:22-alpine
WORKDIR /app
COPY package.json ./
RUN npm install --omit=dev
COPY . .
EXPOSE 3000
CMD ["node", "app.js"]
```

- [ ] **Step 5: Create express-mongodb/compose.yml**

```yaml
services:
  web:
    build: .
    ports:
      - "3000:3000"
    environment:
      - MONGO_URL=mongodb://db:27017/app
    depends_on:
      - db

  db:
    image: mongo:8
    volumes:
      - db_data:/data/db

volumes:
  db_data:
```

- [ ] **Step 6: Create express-mongodb/.dockerignore**

```
README.md
.git
node_modules
```

- [ ] **Step 7: Create express-mongodb/README.md**

```markdown
# express-mongodb

Express.js と MongoDB を使ったシンプルな投稿アプリです。Mongoose による CRUD 機能を持ちます。

## 構成

- Node.js 22 + Express.js 5
- MongoDB 8
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
conoha app init myserver --app-name express-app

# デプロイ
conoha app deploy myserver --app-name express-app
```

## 動作確認

ブラウザで `http://<サーバーIP>:3000` にアクセスすると投稿一覧ページが表示されます。

## カスタマイズ

- `app.js` にルートを追加して機能を拡張
- `views/` に EJS テンプレートを追加
- MongoDB は認証なしで起動するため、本番環境では認証設定を追加
```

- [ ] **Step 8: Commit**

```bash
cd /home/tkim/dev/crowdy/conoha-cli-app-samples
git add express-mongodb/
git commit -m "Add Express.js + MongoDB sample (Mongoose CRUD app)"
```

---

### Task 3: laravel-mysql

**Files:**
- Create: `laravel-mysql/README.md`
- Create: `laravel-mysql/Dockerfile`
- Create: `laravel-mysql/compose.yml`
- Create: `laravel-mysql/.dockerignore`
- Create: `laravel-mysql/composer.json`
- Create: `laravel-mysql/artisan`
- Create: `laravel-mysql/public/index.php`
- Create: `laravel-mysql/bootstrap/app.php`
- Create: `laravel-mysql/bootstrap/providers.php`
- Create: `laravel-mysql/config/app.php`
- Create: `laravel-mysql/config/database.php`
- Create: `laravel-mysql/routes/web.php`
- Create: `laravel-mysql/app/Models/Post.php`
- Create: `laravel-mysql/app/Http/Controllers/PostController.php`
- Create: `laravel-mysql/app/Providers/AppServiceProvider.php`
- Create: `laravel-mysql/resources/views/posts/index.blade.php`
- Create: `laravel-mysql/database/migrations/2026_01_01_000000_create_posts_table.php`
- Create: `laravel-mysql/bin/docker-entrypoint`

- [ ] **Step 1: Create laravel-mysql/composer.json**

```json
{
    "name": "conoha/laravel-sample",
    "type": "project",
    "require": {
        "php": "^8.3",
        "laravel/framework": "^11.0"
    },
    "autoload": {
        "psr-4": {
            "App\\": "app/"
        }
    },
    "config": {
        "optimize-autoloader": true,
        "preferred-install": "dist",
        "sort-packages": true
    },
    "extra": {
        "laravel": {
            "dont-discover": []
        }
    },
    "minimum-stability": "stable",
    "prefer-stable": true
}
```

- [ ] **Step 2: Create laravel-mysql/artisan**

```php
#!/usr/bin/env php
<?php

use Illuminate\Foundation\Application;

define('LARAVEL_START', microtime(true));

require __DIR__.'/vendor/autoload.php';

$app = require_once __DIR__.'/bootstrap/app.php';
$kernel = $app->make(Illuminate\Contracts\Console\Kernel::class);
$status = $kernel->handle(
    $input = new Symfony\Component\Console\Input\ArgvInput,
    new Symfony\Component\Console\Output\ConsoleOutput
);
$kernel->terminate($input, $status);
exit($status);
```

Make executable: `chmod +x laravel-mysql/artisan`

- [ ] **Step 3: Create laravel-mysql/public/index.php**

```php
<?php

use Illuminate\Http\Request;

define('LARAVEL_START', microtime(true));

require __DIR__.'/../vendor/autoload.php';

$app = require_once __DIR__.'/../bootstrap/app.php';
$kernel = $app->make(Illuminate\Contracts\Http\Kernel::class);
$response = $kernel->handle($request = Request::capture());
$response->send();
$kernel->terminate($request, $response);
```

- [ ] **Step 4: Create laravel-mysql/bootstrap/app.php**

```php
<?php

use Illuminate\Foundation\Application;
use Illuminate\Foundation\Configuration\Exceptions;
use Illuminate\Foundation\Configuration\Middleware;

return Application::configure(basePath: dirname(__DIR__))
    ->withRouting(web: __DIR__.'/../routes/web.php')
    ->withMiddleware(function (Middleware $middleware) {})
    ->withExceptions(function (Exceptions $exceptions) {})
    ->create();
```

- [ ] **Step 5: Create laravel-mysql/bootstrap/providers.php**

```php
<?php

return [
    App\Providers\AppServiceProvider::class,
];
```

- [ ] **Step 6: Create laravel-mysql/config/app.php**

```php
<?php

return [
    'name' => 'Laravel on ConoHa',
    'env' => env('APP_ENV', 'production'),
    'debug' => (bool) env('APP_DEBUG', false),
    'url' => env('APP_URL', 'http://localhost'),
    'timezone' => 'Asia/Tokyo',
    'locale' => 'en',
    'key' => env('APP_KEY'),
    'maintenance' => ['driver' => 'file'],
];
```

- [ ] **Step 7: Create laravel-mysql/config/database.php**

```php
<?php

return [
    'default' => 'mysql',
    'connections' => [
        'mysql' => [
            'driver' => 'mysql',
            'host' => env('DB_HOST', 'db'),
            'port' => env('DB_PORT', '3306'),
            'database' => env('DB_DATABASE', 'laravel'),
            'username' => env('DB_USERNAME', 'laravel'),
            'password' => env('DB_PASSWORD', 'laravel'),
            'charset' => 'utf8mb4',
            'collation' => 'utf8mb4_unicode_ci',
            'prefix' => '',
        ],
    ],
    'migrations' => [
        'table' => 'migrations',
    ],
];
```

- [ ] **Step 8: Create laravel-mysql/app/Providers/AppServiceProvider.php**

```php
<?php

namespace App\Providers;

use Illuminate\Support\ServiceProvider;

class AppServiceProvider extends ServiceProvider
{
    public function register(): void {}
    public function boot(): void {}
}
```

- [ ] **Step 9: Create laravel-mysql/app/Models/Post.php**

```php
<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

class Post extends Model
{
    protected $fillable = ['title', 'body'];
}
```

- [ ] **Step 10: Create laravel-mysql/app/Http/Controllers/PostController.php**

```php
<?php

namespace App\Http\Controllers;

use App\Models\Post;
use Illuminate\Http\Request;

class PostController
{
    public function index()
    {
        $posts = Post::orderBy('created_at', 'desc')->get();
        return view('posts.index', compact('posts'));
    }

    public function store(Request $request)
    {
        $request->validate(['title' => 'required|string|max:255']);
        Post::create($request->only('title', 'body'));
        return redirect('/');
    }

    public function destroy(Post $post)
    {
        $post->delete();
        return redirect('/');
    }
}
```

- [ ] **Step 11: Create laravel-mysql/routes/web.php**

```php
<?php

use App\Http\Controllers\PostController;
use Illuminate\Support\Facades\Route;

Route::get('/', [PostController::class, 'index']);
Route::post('/posts', [PostController::class, 'store']);
Route::delete('/posts/{post}', [PostController::class, 'destroy']);
```

- [ ] **Step 12: Create laravel-mysql/resources/views/posts/index.blade.php**

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Laravel on ConoHa</title>
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
    button { padding: 0.5rem 1.5rem; background: #1976d2; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 1rem; }
    .delete { background: #d32f2f; font-size: 0.85rem; padding: 0.3rem 0.8rem; }
    form.inline { display: inline; }
  </style>
</head>
<body>
  <h1>Laravel on ConoHa</h1>
  <div class="form-box">
    <form action="/posts" method="post">
      @csrf
      <input type="text" name="title" placeholder="Title" required>
      <textarea name="body" placeholder="Body (optional)"></textarea>
      <button type="submit">Create Post</button>
    </form>
  </div>
  @foreach ($posts as $post)
    <div class="post">
      <h2>{{ $post->title }}</h2>
      <p>{{ $post->body }}</p>
      <form action="/posts/{{ $post->id }}" method="post" class="inline">
        @csrf
        @method('DELETE')
        <button type="submit" class="delete">Delete</button>
      </form>
    </div>
  @endforeach
</body>
</html>
```

- [ ] **Step 13: Create laravel-mysql/database/migrations/2026_01_01_000000_create_posts_table.php**

```php
<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('posts', function (Blueprint $table) {
            $table->id();
            $table->string('title');
            $table->text('body')->nullable();
            $table->timestamps();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('posts');
    }
};
```

- [ ] **Step 14: Create laravel-mysql/bin/docker-entrypoint**

```bash
#!/bin/bash
set -e

# Generate app key if not set
if [ -z "$APP_KEY" ]; then
    php artisan key:generate --force
fi

# Run database migrations
php artisan migrate --force

exec "$@"
```

Make executable: `chmod +x laravel-mysql/bin/docker-entrypoint`

- [ ] **Step 15: Create laravel-mysql/Dockerfile**

```dockerfile
FROM composer:2 AS deps
WORKDIR /app
COPY composer.json ./
RUN composer install --no-dev --no-scripts --prefer-dist

FROM php:8.3-apache
RUN docker-php-ext-install pdo_mysql
RUN a2enmod rewrite
ENV APACHE_DOCUMENT_ROOT=/var/www/html/public
RUN sed -ri -e 's!/var/www/html!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/sites-available/*.conf \
    && sed -ri -e 's!/var/www/!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/apache2.conf /etc/apache2/conf-available/*.conf \
    && sed -ri -e 's/AllowOverride None/AllowOverride All/g' /etc/apache2/apache2.conf
WORKDIR /var/www/html
COPY --from=deps /app/vendor ./vendor
COPY . .
RUN chown -R www-data:www-data storage bootstrap/cache 2>/dev/null || true
RUN chmod +x bin/docker-entrypoint
EXPOSE 80
ENTRYPOINT ["bin/docker-entrypoint"]
CMD ["apache2-foreground"]
```

- [ ] **Step 16: Create laravel-mysql/compose.yml**

```yaml
services:
  web:
    build: .
    ports:
      - "80:80"
    environment:
      - APP_ENV=production
      - APP_DEBUG=false
      - DB_HOST=db
      - DB_DATABASE=laravel
      - DB_USERNAME=laravel
      - DB_PASSWORD=laravel
    depends_on:
      db:
        condition: service_healthy

  db:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=rootpassword
      - MYSQL_DATABASE=laravel
      - MYSQL_USER=laravel
      - MYSQL_PASSWORD=laravel
    volumes:
      - db_data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  db_data:
```

- [ ] **Step 17: Create laravel-mysql/.dockerignore**

```
README.md
.git
vendor
node_modules
storage/logs/*
```

- [ ] **Step 18: Create required Laravel directories**

```bash
mkdir -p laravel-mysql/storage/framework/{sessions,views,cache}
mkdir -p laravel-mysql/storage/logs
mkdir -p laravel-mysql/bootstrap/cache
touch laravel-mysql/storage/framework/sessions/.gitkeep
touch laravel-mysql/storage/framework/views/.gitkeep
touch laravel-mysql/storage/framework/cache/.gitkeep
touch laravel-mysql/storage/logs/.gitkeep
touch laravel-mysql/bootstrap/cache/.gitkeep
```

- [ ] **Step 19: Create laravel-mysql/README.md**

```markdown
# laravel-mysql

Laravel と MySQL を使ったシンプルな投稿アプリです。Eloquent ORM による CRUD 機能を持ちます。

## 構成

- PHP 8.3 + Laravel 11
- MySQL 8.0
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
conoha app init myserver --app-name laravel-app

# デプロイ
conoha app deploy myserver --app-name laravel-app
```

DB マイグレーションと APP_KEY 生成はコンテナ起動時に自動実行されます。

## 動作確認

ブラウザで `http://<サーバーIP>` にアクセスすると投稿一覧ページが表示されます。

## カスタマイズ

- `app/Http/Controllers/` にコントローラーを追加
- `resources/views/` に Blade テンプレートを追加
- `database/migrations/` にマイグレーションを追加してスキーマを変更
- 本番環境では `DB_PASSWORD` を `.env.server` で管理
```

- [ ] **Step 20: Commit**

```bash
cd /home/tkim/dev/crowdy/conoha-cli-app-samples
git add laravel-mysql/
git commit -m "Add Laravel + MySQL sample (Eloquent CRUD app)"
```

---

### Task 4: django-postgresql

**Files:**
- Create: `django-postgresql/README.md`
- Create: `django-postgresql/Dockerfile`
- Create: `django-postgresql/compose.yml`
- Create: `django-postgresql/.dockerignore`
- Create: `django-postgresql/requirements.txt`
- Create: `django-postgresql/manage.py`
- Create: `django-postgresql/config/settings.py`
- Create: `django-postgresql/config/urls.py`
- Create: `django-postgresql/config/wsgi.py`
- Create: `django-postgresql/posts/models.py`
- Create: `django-postgresql/posts/views.py`
- Create: `django-postgresql/posts/urls.py`
- Create: `django-postgresql/posts/forms.py`
- Create: `django-postgresql/posts/admin.py`
- Create: `django-postgresql/posts/apps.py`
- Create: `django-postgresql/templates/posts/index.html`
- Create: `django-postgresql/bin/docker-entrypoint`

- [ ] **Step 1: Create django-postgresql/requirements.txt**

```
django==5.2
psycopg[binary]==3.2.6
gunicorn==23.0.0
```

- [ ] **Step 2: Create django-postgresql/manage.py**

```python
#!/usr/bin/env python
import os
import sys

def main():
    os.environ.setdefault("DJANGO_SETTINGS_MODULE", "config.settings")
    from django.core.management import execute_from_command_line
    execute_from_command_line(sys.argv)

if __name__ == "__main__":
    main()
```

Make executable: `chmod +x django-postgresql/manage.py`

- [ ] **Step 3: Create django-postgresql/config/settings.py**

```python
import os
from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent.parent

SECRET_KEY = os.environ.get("SECRET_KEY", "change-me-in-production")
DEBUG = os.environ.get("DEBUG", "0") == "1"
ALLOWED_HOSTS = ["*"]

INSTALLED_APPS = [
    "django.contrib.admin",
    "django.contrib.auth",
    "django.contrib.contenttypes",
    "django.contrib.sessions",
    "django.contrib.messages",
    "django.contrib.staticfiles",
    "posts",
]

MIDDLEWARE = [
    "django.middleware.security.SecurityMiddleware",
    "django.contrib.sessions.middleware.SessionMiddleware",
    "django.middleware.common.CommonMiddleware",
    "django.middleware.csrf.CsrfViewMiddleware",
    "django.contrib.auth.middleware.AuthenticationMiddleware",
    "django.contrib.messages.middleware.MessageMiddleware",
]

ROOT_URLCONF = "config.urls"

TEMPLATES = [
    {
        "BACKEND": "django.template.backends.django.DjangoTemplates",
        "DIRS": [BASE_DIR / "templates"],
        "APP_DIRS": True,
        "OPTIONS": {
            "context_processors": [
                "django.template.context_processors.request",
                "django.contrib.auth.context_processors.auth",
                "django.contrib.messages.context_processors.messages",
            ],
        },
    },
]

WSGI_APPLICATION = "config.wsgi.application"

DATABASES = {
    "default": {
        "ENGINE": "django.db.backends.postgresql",
        "HOST": os.environ.get("DB_HOST", "db"),
        "NAME": os.environ.get("DB_NAME", "app_production"),
        "USER": os.environ.get("DB_USER", "postgres"),
        "PASSWORD": os.environ.get("DB_PASSWORD", "postgres"),
    }
}

STATIC_URL = "static/"
STATIC_ROOT = BASE_DIR / "staticfiles"
DEFAULT_AUTO_FIELD = "django.db.models.BigAutoField"
```

- [ ] **Step 4: Create django-postgresql/config/urls.py**

```python
from django.contrib import admin
from django.urls import include, path

urlpatterns = [
    path("admin/", admin.site.urls),
    path("", include("posts.urls")),
]
```

- [ ] **Step 5: Create django-postgresql/config/wsgi.py**

```python
import os
from django.core.wsgi import get_wsgi_application

os.environ.setdefault("DJANGO_SETTINGS_MODULE", "config.settings")
application = get_wsgi_application()
```

- [ ] **Step 6: Create django-postgresql/posts/apps.py**

```python
from django.apps import AppConfig

class PostsConfig(AppConfig):
    default_auto_field = "django.db.models.BigAutoField"
    name = "posts"
```

- [ ] **Step 7: Create django-postgresql/posts/models.py**

```python
from django.db import models

class Post(models.Model):
    title = models.CharField(max_length=255)
    body = models.TextField(blank=True)
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        ordering = ["-created_at"]

    def __str__(self):
        return self.title
```

- [ ] **Step 8: Create django-postgresql/posts/forms.py**

```python
from django import forms
from .models import Post

class PostForm(forms.ModelForm):
    class Meta:
        model = Post
        fields = ["title", "body"]
        widgets = {
            "title": forms.TextInput(attrs={"placeholder": "Title"}),
            "body": forms.Textarea(attrs={"placeholder": "Body (optional)", "rows": 3}),
        }
```

- [ ] **Step 9: Create django-postgresql/posts/views.py**

```python
from django.shortcuts import redirect, get_object_or_404
from django.views.generic import ListView
from .forms import PostForm
from .models import Post

class PostListView(ListView):
    model = Post
    template_name = "posts/index.html"
    context_object_name = "posts"

    def get_context_data(self, **kwargs):
        context = super().get_context_data(**kwargs)
        context["form"] = PostForm()
        return context

def post_create(request):
    if request.method == "POST":
        form = PostForm(request.POST)
        if form.is_valid():
            form.save()
    return redirect("/")

def post_delete(request, pk):
    if request.method == "POST":
        post = get_object_or_404(Post, pk=pk)
        post.delete()
    return redirect("/")
```

- [ ] **Step 10: Create django-postgresql/posts/urls.py**

```python
from django.urls import path
from . import views

urlpatterns = [
    path("", views.PostListView.as_view(), name="post_list"),
    path("posts/create/", views.post_create, name="post_create"),
    path("posts/<int:pk>/delete/", views.post_delete, name="post_delete"),
]
```

- [ ] **Step 11: Create django-postgresql/posts/admin.py**

```python
from django.contrib import admin
from .models import Post

admin.site.register(Post)
```

- [ ] **Step 12: Create django-postgresql/templates/posts/index.html**

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Django on ConoHa</title>
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
    button { padding: 0.5rem 1.5rem; background: #1976d2; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 1rem; }
    .delete { background: #d32f2f; font-size: 0.85rem; padding: 0.3rem 0.8rem; }
    form.inline { display: inline; }
  </style>
</head>
<body>
  <h1>Django on ConoHa</h1>
  <div class="form-box">
    <form action="{% url 'post_create' %}" method="post">
      {% csrf_token %}
      {{ form.title }}
      {{ form.body }}
      <button type="submit">Create Post</button>
    </form>
  </div>
  {% for post in posts %}
    <div class="post">
      <h2>{{ post.title }}</h2>
      <p>{{ post.body }}</p>
      <form action="{% url 'post_delete' post.pk %}" method="post" class="inline">
        {% csrf_token %}
        <button type="submit" class="delete">Delete</button>
      </form>
    </div>
  {% endfor %}
</body>
</html>
```

- [ ] **Step 13: Create django-postgresql/bin/docker-entrypoint**

```bash
#!/bin/bash
set -e

# Run database migrations
python manage.py migrate --noinput

# Collect static files
python manage.py collectstatic --noinput

exec "$@"
```

Make executable: `chmod +x django-postgresql/bin/docker-entrypoint`

- [ ] **Step 14: Create django-postgresql/Dockerfile**

```dockerfile
FROM python:3.12-slim
WORKDIR /app
RUN apt-get update -qq && apt-get install -y libpq5 && rm -rf /var/lib/apt/lists/*
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
RUN chmod +x bin/docker-entrypoint manage.py
EXPOSE 8000
ENTRYPOINT ["bin/docker-entrypoint"]
CMD ["gunicorn", "config.wsgi:application", "--bind", "0.0.0.0:8000"]
```

- [ ] **Step 15: Create django-postgresql/compose.yml**

```yaml
services:
  web:
    build: .
    ports:
      - "8000:8000"
    environment:
      - DB_HOST=db
      - DB_USER=postgres
      - DB_PASSWORD=postgres
      - DB_NAME=app_production
      - SECRET_KEY=change-me-in-production
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

- [ ] **Step 16: Create django-postgresql/.dockerignore**

```
README.md
.git
__pycache__
*.pyc
.venv
staticfiles
```

- [ ] **Step 17: Create empty __init__.py files for Python packages**

```bash
touch django-postgresql/config/__init__.py
touch django-postgresql/posts/__init__.py
touch django-postgresql/posts/migrations/__init__.py
```

- [ ] **Step 18: Create django-postgresql/README.md**

```markdown
# django-postgresql

Django と PostgreSQL を使ったシンプルな投稿アプリです。Django ORM による CRUD 機能と管理画面を持ちます。

## 構成

- Python 3.12 + Django 5.2
- PostgreSQL 17
- Gunicorn（アプリサーバー）
- ポート: 8000

## 前提条件

- conoha-cli がインストール済み
- ConoHa VPS3 アカウント
- SSH キーペア設定済み

## デプロイ

```bash
# サーバー作成（まだない場合）
conoha server create --name myserver --flavor g2l-t-2 --image ubuntu-24.04 --key mykey

# アプリ初期化
conoha app init myserver --app-name django-app

# デプロイ
conoha app deploy myserver --app-name django-app
```

DB マイグレーションはコンテナ起動時に自動実行されます。

## 動作確認

ブラウザで `http://<サーバーIP>:8000` にアクセスすると投稿一覧ページが表示されます。

Django 管理画面は `http://<サーバーIP>:8000/admin/` からアクセスできます（スーパーユーザーの作成が必要）。

## カスタマイズ

- `posts/` アプリを編集して機能を追加
- `python manage.py startapp <name>` で新しいアプリを追加
- `python manage.py createsuperuser` で管理画面のユーザーを作成
- 本番環境では `SECRET_KEY` と `DB_PASSWORD` を `.env.server` で管理
```

- [ ] **Step 19: Commit**

```bash
cd /home/tkim/dev/crowdy/conoha-cli-app-samples
git add django-postgresql/
git commit -m "Add Django + PostgreSQL sample (ORM CRUD app with admin)"
```

---

### Task 5: Update root README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update the samples table in root README.md**

Add 4 new rows to the サンプル一覧 table in `/home/tkim/dev/crowdy/conoha-cli-app-samples/README.md`. The table should now be:

```markdown
| サンプル | スタック | 説明 | 推奨フレーバー |
|---------|---------|------|--------------|
| [hello-world](hello-world/) | nginx + 静的HTML | 最もシンプルなサンプル | g2l-t-1 (1GB) |
| [nextjs](nextjs/) | Next.js (standalone) | Next.js デフォルトページ | g2l-t-2 (2GB) |
| [fastapi-ai-chatbot](fastapi-ai-chatbot/) | FastAPI + Ollama | AI チャットボット | g2l-t-4 (4GB) |
| [rails-postgresql](rails-postgresql/) | Rails + PostgreSQL | Rails scaffold アプリ | g2l-t-2 (2GB) |
| [wordpress-mysql](wordpress-mysql/) | WordPress + MySQL | WordPress ブログ | g2l-t-2 (2GB) |
| [spring-boot-postgresql](spring-boot-postgresql/) | Spring Boot + PostgreSQL | JPA CRUD アプリ | g2l-t-2 (2GB) |
| [express-mongodb](express-mongodb/) | Express.js + MongoDB | Mongoose CRUD アプリ | g2l-t-2 (2GB) |
| [laravel-mysql](laravel-mysql/) | Laravel + MySQL | Eloquent CRUD アプリ | g2l-t-2 (2GB) |
| [django-postgresql](django-postgresql/) | Django + PostgreSQL | Django ORM アプリ + 管理画面 | g2l-t-2 (2GB) |
```

- [ ] **Step 2: Commit**

```bash
cd /home/tkim/dev/crowdy/conoha-cli-app-samples
git add README.md
git commit -m "Update root README with Tier 1 samples"
```

---

### Task 6: Push to GitHub

- [ ] **Step 1: Push all new commits**

```bash
cd /home/tkim/dev/crowdy/conoha-cli-app-samples
git push origin main
```
