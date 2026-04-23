# `conoha app env` 再設計 (blue/green + accessory 対応)

**Date**: 2026-04-23
**Status**: Proposed
**Owner**: t-kim
**Related**: #94 (this issue), #103 (no-proxy mode), #98 (proxy blue/green)

## 1. 背景

`cmd/app/env.go` はサーバ側ファイル `/opt/conoha/<app>.env.server` を介してアプリの環境変数を永続化する。`--no-proxy` モードでは deploy 時に `.env` へ append される (§3.6, 2026-04-21-no-proxy-mode-design.md) が、proxy モードでは現状 **無効** で、`app env set` は警告のみを出して値を書き込むだけ (読み取り側が存在しない)。

この不整合を解消する。

### 1.1 現行の問題点

1. **proxy モードで値が流れない**: compose slot の override にも accessory compose にも `.env.server` は参照されない。ユーザは `app env set` に成功しても実アプリには反映されない。
2. **パスがレイアウト規約に反する**: `/opt/conoha/<app>.env.server` は `/opt/conoha/<app>/` (app work dir) の *外側* にある。`--no-proxy` モード初期化時 (`mkdir -p /opt/conoha/<name>/`) の規約は "app の全資産は `/opt/conoha/<name>/` 配下" であり、一段上に置くのは歴史的経緯。
3. **destroy との結合が暗黙**: `destroy.go` が `rm -f /opt/conoha/${APP_NAME}.env.server` を明示的に実行する必要がある。一般的な "app dir 配下を消せば終わり" 規約から外れる。
4. **accessory との責務分離不明**: accessory が DB パスワードなど独自の env を必要とする場合の受け皿がない。現状は compose ファイルの `environment:` / `env_file:` に直書きするしかない。

## 2. 設計判断

### 2.1 保存場所

**決定**: **単一ファイル** `/opt/conoha/<name>/.env.server`。

選択肢の検討:
- (A) `/opt/conoha/<name>/env.server` (app dir 配下、単一) ← **採用 (dotfile 化した変種)**
- (B) accessory 用に別ファイル `/opt/conoha/<name>/accessories/.env` を追加
- (C) サブコマンドを `app env web` / `app env accessory` に分離

**(A) 採用理由**:
- ユーザ体験の単純さ。`app env set` と `app env accessory set` を使い分けるのは新規ユーザに認知負荷が高く、大半のアプリでは web 側 env だけで足りる。
- accessory 固有の env (POSTGRES_PASSWORD 等) は compose ファイルの `environment:` / `env_file: .env` が既存のベストプラクティス。`app env` は **ランタイム注入用** と位置づけて機能を絞る。
- dotfile 化 (`.env.server`) は `ls /opt/conoha/<name>/` のノイズを減らすため。既存マーカー (`.conoha-mode`) と揃う。

**(B)/(C) を採用しない理由**:
- accessory は blue/green 対象外 (persistent) なので env 変更は即時反映が期待される。これは `app env` の "次回 deploy で反映" セマンティクスと食い違う。別の UX (restart trigger 等) が必要になりスコープが膨らむ。
- 現状 accessory 向けの実ユースケースが未計測 (ConoHa サンプル群は compose に直書きで足りている)。必要になった段階で別 issue で追加する余地を残す。

**再検討トリガ** (この判断を将来ひっくり返す条件):
- accessory 固有 env を compose 直書き / `env_file: .env` で回避できない具体事例が GitHub issue で 1 件でも上がる、または
- 類似要望 (`app env accessory ...`) が cumulative 3 件以上上がった段階で `#94-followup` として別 issue を切り、本節を更新する。

### 2.2 適用タイミング

**決定**: **次回 deploy 時のみ**。ライブ更新は行わない。

- web slot は ephemeral: 次の `app deploy` で新スロットが `.env.server` を読んで起動する。
- 現行 slot への live 反映は blue/green の不変条件 (スロット内容は deploy 時点のスナップショット) を壊す。
- `app env set` 後に即時反映したい場合は `app deploy` を再実行するよう UX で促す (stderr 一行案内)。

### 2.2.1 rollback 時の env 解決

`app rollback` で前スロット (例: slot-a → slot-b) へ戻した時、web コンテナは
**ロールバック実行時点の `/opt/conoha/<name>/.env.server`** を読む。
compose は `env_file` を container start 時に評価するため、デプロイ時点の
スナップショット値ではなく現在の `.env.server` が使われる。

これは web env が「現在の運用値」を反映すべきという原則に合致する。
スナップショット巻き戻しが必要な場合は `app env set` で明示的に復元する
運用 (env の history/diff 機能は本 issue のスコープ外)。

関連テスト: 受け入れ基準 §6 に "rollback 後も同じ KEY が見える" を追加。

### 2.3 proxy / no-proxy での反映方法

**proxy モード**:
- deploy 時、`/opt/conoha/<name>/.env.server` をスロット work dir にコピーせず、compose override で `env_file` エントリを web service に付与する (常に注入。ファイル存在は §7 の "init 時 `touch`" で保証)。
- override の絶対パス参照は compose v2.20+ でサポート済み。
- 欠点: compose override に絶対パスが入ると `compose config` の可搬性が下がる。受容: ポータビリティより "原本は一箇所" の原則を優先。
- accessory プロジェクトは対象外 (§2.1)。

**no-proxy モード**:
- 既存の "deploy 時に `.env` へ append" を維持するが、ソースのパスを `/opt/conoha/<app>.env.server` → `/opt/conoha/<app>/.env.server` に移す (§4 migration 参照)。
- append タイミング・順序 (repo の `.env` → `.env.server`、last-occurrence wins) は変更しない。

### 2.4 ユーザ視点の UX 変更

変わらないこと:
- サブコマンド面 (`app env set/get/list/unset`) は維持。
- キー validation (`ValidateEnvKey`) は維持。
- proxy モードでの "warning: no effect on proxy-mode deployed slots" 警告は **削除** される (今回で解消するため)。

変わること:
- `app env set` 完了時の一行案内: `"Next: run 'conoha app deploy <server>' to apply"`。
- proxy モードでは compose override 経由で next deploy 時に反映されるようになる (#94 の解消)。
- no-proxy モードの挙動は機能的に変わらないが、保存場所が移る。

## 3. 新レイアウト

```
/opt/conoha/<name>/
├── .conoha-mode       # proxy | no-proxy (既存、#103)
├── .env.server        # <== 新位置
├── CURRENT_SLOT       # proxy mode: active slot id (既存)
├── <slot>/            # proxy mode: blue/green work dir (既存)
│   ├── docker-compose.yml
│   ├── conoha-override.yml
│   └── .env           # 既存: tar extraction 由来 + 旧 .env.server の merge 結果
└── ...                # no-proxy mode: flat work dir の各ファイル
```

proxy モードの compose override が新たに追加するエントリ:
```yaml
services:
  <web>:
    env_file:
      - /opt/conoha/<name>/.env.server
```

compose-spec の短縮形を用いる。`env_file` には `:ro` 接尾辞は存在しない
(compose-spec における `:ro` は volume-mount 限定)。long-form
(`path:` / `required:` / `format:`) は compose 2.24+ 限定でありかつ本設計では
不要 (ファイル存在は init 時の `touch` で保証するため `required: true`
相当の既定挙動で十分)。

## 4. 移行

**自動移行は行わない**。理由: サーバ側状態を CLI が勝手に移動するとサイレント破壊のリスクがあり、ユーザが git 管理外のファイルを「別ツールで書き換えた」可能性を排除できない。

代わりに:

1. `app env set` / `app env list` / `app env get` / `app env unset` が **入口** で以下を実行:
   - 新位置 (`/opt/conoha/<name>/.env.server`) が既に存在: それを使う。
   - 新位置不在 + 旧位置 (`/opt/conoha/<name>.env.server`) 存在: stderr に警告 + 旧位置を読む (後方互換)。
     ```
     warning: legacy env file at /opt/conoha/<name>.env.server detected.
              Run 'conoha app env migrate <server>' to move it to /opt/conoha/<name>/.env.server.
              The legacy location will stop being read in the next minor release.
     ```
   - 両方不在: 新規作成は新位置に。

2. `conoha app env migrate <server>` を新サブコマンドとして追加。旧→新の `mv` を 1 コマンドで実行、成功時に旧ファイルを削除。

3. `destroy` は `rm -rf /opt/conoha/<name>` で新位置を自動清掃。旧位置残骸の `rm -f /opt/conoha/${APP_NAME}.env.server` 明示削除は **v0.2.x の間は残す** (destroy は legacy サーバの掃除経路も兼ねるため — `destroy.go` 既存コメント参照)。**v0.3.x で削除予定** — タイムラインは §5.3 参照。

## 5. 実装アーキテクチャ

### 5.1 変更対象ファイル

- `cmd/app/env.go`:
  - `envFilePath(app)` ヘルパを導入し全サブコマンドで共用。新位置を返す。
  - `legacyEnvFilePath(app)` 追加。migrate コマンドと後方互換 read パスで使用。
  - `maybeWarnProxyEnvMode` を **削除** (redesign で解消するため)。警告を出す代わりに、各サブコマンドが "next: app deploy" の案内を出す。
  - 新サブコマンド `envMigrateCmd` 追加。
  - **レガシー位置検出 helper**: `connectToApp` 直後に `ctx` に `LegacyEnvPath bool` を 1 度だけ stat して cache する (1d1c185 の `ctx.Marker()` パターンに準ずる)。env サブコマンド呼び出し毎に SSH round-trip が増えないようにする。warning 出力はこの cache 値を見て分岐。

- `cmd/app/init.go`:
  - proxy モード初期化時に `touch /opt/conoha/<name>/.env.server` を
    実行する (空ファイルで OK)。§7 の "欠落時 compose が起動失敗" 対策。
    no-proxy モードでも同じ `touch` を行う (挙動統一)。

- `cmd/app/deploy.go`:
  - `runProxyDeploy`: compose override 生成時に `env_file` エントリを **常に注入** する。ファイル存在は `app init` の touch が保証するため分岐不要。
  - `runNoProxyDeploy`: `.env` merge の source パスを新位置に変更。旧位置 fallback は **append しない** (migrate コマンド推奨)。

- `cmd/app/override.go`:
  - `composeOverride` のシグネチャを拡張して env_file 注入フラグを受ける。あるいは `webEnvFile` 引数を追加。

- `cmd/app/destroy.go`:
  - 変更なし (`rm -rf /opt/conoha/<name>` で新位置は自動対応。legacy `rm -f` は §5.3 のスケジュールで削除予定まで据え置き)。

### 5.2 `app env migrate` のふるまい

```
conoha app env migrate <server>
```

1. connectToApp で SSH connect、app 名取得、mode marker 読む。
2. 旧位置 (`/opt/conoha/<app>.env.server`) の存在チェック。
3. 新位置が既に存在 → "both exist" エラー、手動解決を促す。
4. 旧位置のみ存在 → `mv` を SSH で実行、ファイル所有権・モード保持。
5. 旧位置不在 → "nothing to migrate" を stdout、exit 0。

再実行安全。

### 5.3 非互換変更

非互換化のタイムライン (README にも明記する):

| バージョン | レガシー位置の扱い | destroy.go の legacy `rm -f` |
|------------|--------------------|------------------------------|
| v0.2.x     | warning + 後方互換 read (本 PR で実装) | 残す (ユーザが migrate 未実施でも掃除) |
| v0.3.x     | **error** (読まない。migrate を要求) | **削除** (全ユーザ migrate 済と見做す) |

v0.3.x 昇格時は `cmd/app/destroy.go` から `rm -f /opt/conoha/<name>.env.server`
行を削除するタスクも合わせて切る。v0.2.x 期間中に発生した migrate コマンド
実行回数を計測できれば、昇格判断の補助データになる (計測手段は別途検討)。

## 6. 受け入れ基準

各項目は具体的な検証コマンドで表現する (QA / CI が自動検証できる形)。

- [ ] proxy モードで `app env set KEY=VALUE` → `app deploy` 実行後、
      `ssh <server> "docker compose -p <name>-<slot> exec <web> printenv KEY"`
      が `VALUE` を出力する。
- [ ] no-proxy モードで `app env set KEY=VALUE` → `app deploy` 実行後、
      `ssh <server> "grep '^KEY=' /opt/conoha/<name>/.env"` が `KEY=VALUE` を返す。
- [ ] 旧位置 (`/opt/conoha/<name>.env.server`) のみ存在するサーバで
      `app env list <server>` を実行すると、stdout に値が出力されつつ stderr に
      `warning: legacy env file ... migrate` を含む行が出る。
- [ ] `app env migrate <server>` を実行すると、旧位置が消え、新位置に同じ内容
      (`diff` ゼロ) のファイルが `0600` で作成される。再実行時は
      `nothing to migrate` を stdout で返し exit 0。
- [ ] `destroy <server>` 実行後、`ssh <server> "ls /opt/conoha/"` で対象 app
      ディレクトリが存在しない かつ `ls /opt/conoha/<name>.env.server 2>&1`
      が `No such file` を返す。
- [ ] `maybeWarnProxyEnvMode` の既存テスト (`cmd/app/env_test.go` の関連ケース)
      が削除され、代わりに各サブコマンドが "Next: run 'conoha app deploy ...'"
      を stderr に含むことを検証するテストが追加される。
- [ ] `app env set KEY=new` → `app rollback` の後、web コンテナで `printenv KEY`
      が **new** を返す (§2.2.1 の rollback env 挙動の確認)。

## 7. オープン技術判断

- **compose 最低バージョン**: 短縮形 `env_file: - /abs/path` は compose 2.20+
  (ConoHa VMI の compose は v2.35 系なので実質問題なし)。README に最低バージョンを明記。
- **欠落時の compose 挙動**: `env_file` の短縮形で指定されたファイルが存在しないと
  compose は **エラーで起動失敗**する。本設計は空の `.env.server` を `app init` 時に
  `touch` で作成する方針 (§5.1 の init 変更) にコミット。override 生成側は条件分岐
  なしで常に注入。
- **シークレット取り扱い**: 本設計は平文 env のみ対象。DB password / API key 等の
  暗号化管理 (sops / age / Vault 等) は別 issue (`#94-secrets-followup` 予定)。
  本 issue のスコープ外であることを README とリリースノートで明記。
- **accessory の env_file 注入**: 対象外 (§2.1)。ユーザが compose で書く。

## 8. 参考

- 旧仕様: [§3.6 (2026-04-21-no-proxy-mode-design.md)](2026-04-21-no-proxy-mode-design.md)
- 関連 issue: #94 (本 issue), #103 (no-proxy mode)
- 前提: feat/proxy-deploy (#98) landed 2026-04-21 (ed1c5e0)
