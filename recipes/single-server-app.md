# シングルサーバーアプリデプロイ（conoha-proxy blue/green）

## 概要

Docker Composeアプリをサーバー1台にデプロイするレシピ。`conoha proxy` で TLS リバースプロキシを立ち上げ、`conoha app` で blue/green デプロイを行う。Let's Encrypt 自動発行、Host ヘッダールーティング、drain 窓内ロールバックをカバーする。

## 基本構成

- **ノード数**: 1
- **OS**: Ubuntu
- **DNS**: 使用するドメインの A レコードが VPS に向いていること（Let's Encrypt HTTP-01 検証用）
- **必須**: レポルートに `conoha.yml` と Docker Compose ファイルがあること

```
[Internet :80/:443]
        │
        ▼
[ConoHa VPS]
 ├── conoha-proxy container           ← TLS 終端 + blue/green スワップ
 │    ├── /var/lib/conoha-proxy/state.db
 │    └── /var/lib/conoha-proxy/admin.sock
 │
 └── /opt/conoha/<name>/
      ├── CURRENT_SLOT                ← 現在 active なスロット
      ├── <slot>/                     ← blue/green ごとの作業ディレクトリ
      │   ├── compose.yml
      │   └── conoha-override.yml     ← 127.0.0.1:0:<port> に動的バインド
      └── accessories は独立プロジェクト `<name>-accessories` で永続化
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

### 3. `conoha.yml` を用意

レポルートに以下の内容で作成：

```yaml
name: myapp
hosts:
  - app.example.com
web:
  service: web       # compose 内の Web サービス名
  port: 8080         # コンテナ内のリスニングポート
# 任意
# accessories: [db, redis]
# deploy:
#   drain_ms: 30000
# health:
#   path: /up
```

compose ファイルは `compose.yml` / `docker-compose.yml` / `conoha-docker-compose.yml` のいずれか。web サービスの `ports:` にホスト側ポートをハードコードしないこと（deploy 時に `127.0.0.1:0:<port>` へ再マッピングされる）。

### 4. プロキシを VPS に起動

```bash
conoha proxy boot my-app-server --acme-email ops@example.com
```

初回は Docker インストールと conoha-proxy コンテナ起動が行われる。`/var/lib/conoha-proxy/` が永続ボリュームとして確保される。

### 5. DNS を設定

`hosts:` に記載したドメインの A レコードを VPS に向ける。Let's Encrypt HTTP-01 検証がこの後で走る。

### 6. アプリを proxy に登録

```bash
conoha app init my-app-server
```

`conoha.yml` を読み取り、proxy の Admin API (`POST /v1/services`) で service を登録する。

### 7. デプロイ

カレントディレクトリで実行：

```bash
conoha app deploy my-app-server
```

内部で行われる処理：
- レポの tar.gz アップロード
- 新スロット（git short SHA もしくはタイムスタンプ）ディレクトリに展開
- compose override を生成して web サービスの host ポートを動的バインド
- 初回のみ accessory (DB 等) を `<name>-accessories` プロジェクトで起動
- 新スロットで `docker compose up -d --build`
- `docker port` で kernel 割当の host ポートを取得
- `POST /v1/services/<name>/deploy { target_url, drain_ms }` を proxy に投げる
- probe 成功で active swap、旧スロットは drain 後にバックグラウンドで `compose down`

### 8. 動作確認

```bash
# HTTPS 接続確認
curl -I https://app.example.com/

# 現在の状態（compose 各 slot + proxy の phase / active target）
conoha app status my-app-server

# 旧 slot が落ちたか確認
conoha proxy services my-app-server
```

## ロールバック

drain 窓（既定 30 秒）の内であれば即時：

```bash
conoha app rollback my-app-server
```

drain 窓が切れていれば `no_drain_target` エラー。旧バージョンをもう一度 deploy する（`--slot <old-sha>` で同じ slot 名を指定可）。同じ slot ID を再利用する際は既存の作業ディレクトリが `rm -rf` されるため、直前の deploy から `drain_ms` 経過前でも安全（teardown スクリプトは実行直前に `CURRENT_SLOT` を再確認し、該当 slot が再 active 化されていれば自動的にスキップする）。

## proxy の運用

```bash
conoha proxy details my-app-server     # バージョンと登録 service 数
conoha proxy logs my-app-server --follow
conoha proxy reboot my-app-server --acme-email ops@example.com  # イメージ更新時
conoha proxy restart my-app-server     # 単純な再起動
```

## v0.1.x からの移行

1. サーバーはそのまま（データ損失なし）。
2. レポに `conoha.yml` を追加。
3. `conoha proxy boot <server> --acme-email you@example.com`
4. `conoha app init <server>`（新しい init は proxy に登録するだけ。既存の `/opt/conoha/<name>.git` があれば警告のみ出して触らない）。
5. `conoha app deploy <server>`（初回の blue/green デプロイ）。
6. （任意）旧 bare repo を削除：
   ```bash
   ssh <user>@<server> rm -rf /opt/conoha/<name>.git /opt/conoha/<name>.env.server
   ```

## トラブルシューティング

| 問題 | 対処 |
|------|------|
| `proxy boot` で 'Container already exists' | 既に動いている。`conoha proxy details` で確認するか、`conoha proxy reboot` で再起動 |
| `app init` が 'legacy git bare repo exists' を警告 | v0.1.x 由来の repo。削除しなくても動くが、上記の移行手順で掃除推奨 |
| `app deploy` が `probe_failed` で失敗 | 新 slot の `/up` が 200 を返していない。`docker compose -p <name>-<slot> logs` で調査 |
| `app rollback` が `no_drain_target` エラー | drain 窓が切れた。旧 slot を `--slot <sha>` で再 deploy |
| HTTPS が繋がらない | DNS 反映前 or Let's Encrypt Rate Limit。`conoha proxy logs` で certmagic のログを確認 |
| 複数アプリを同じ VPS に載せたい | `conoha.yml` を各レポに用意し、各レポで `app init` + `app deploy` するだけ（proxy は共有） |
