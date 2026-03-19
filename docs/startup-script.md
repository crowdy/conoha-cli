# Startup Script (user_data)

ConoHa VPS3 では、サーバー作成時にスタートアップスクリプトを指定して初期設定を自動化できます。

## 使い方

### ファイル指定

```bash
conoha server create --name my-server --user-data ./init.sh
```

`init.sh` の例：
```bash
#!/bin/bash
apt update && apt install -y nginx
systemctl enable --now nginx
```

### インラインスクリプト

```bash
conoha server create --name my-server --user-data-raw '#!/bin/bash
apt update && apt install -y docker.io'
```

### URL 指定

```bash
conoha server create --name my-server --user-data-url https://example.com/setup.sh
```

内部的に `#include` ディレクティブとしてラップされます：
```
#include
https://example.com/setup.sh
```

## 制限事項

- **最大サイズ**: 16 KiB（base64 エンコード前）
- **Linux のみ**: Windows Server フレーバー（`g2w-*`）では利用不可（警告が表示されます）
- **3 つのフラグは排他**: `--user-data`, `--user-data-raw`, `--user-data-url` は同時に指定できません

## サポートされるヘッダー

| ヘッダー | 説明 |
|---------|------|
| `#!` | シェルスクリプト |
| `#cloud-config` | cloud-init 設定（YAML） |
| `#cloud-boothook` | 起動時に毎回実行 |
| `#include` | URL からスクリプトを取得 |
| `#include-once` | 初回のみ URL から取得 |

## cloud-config の例

```yaml
#cloud-config
packages:
  - nginx
  - certbot
runcmd:
  - systemctl enable --now nginx
```

## エージェント連携

非対話モードでスクリプトからサーバーを作成：

```bash
conoha server create \
  --name web-01 \
  --flavor g2l-t-c2m1 \
  --image <image-id> \
  --key-name my-key \
  --user-data ./setup.sh \
  --yes --no-input --format json
```

## 参考リンク

- [ConoHa スタートアップスクリプト](https://support.conoha.jp/v/startupscript/)
- [ConoHa VPS スタートアップスクリプト機能](https://vps.conoha.jp/function/startupscript/)
- [ConoHa VPS3 API ドキュメント](https://doc.conoha.jp/products/vps-v3/startupscripts-v3/)
