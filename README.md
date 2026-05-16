# Docker Python Sandbox

Dockerコンテナ内でPythonコードを安全に実行するPOCシステム

## セキュリティ

### sandbox-controller
- **distrolessイメージ**: 最小攻撃表面積のベースイメージを使用
- **読み取り専用rootfs**: 書き込み不可のファイルシステムで保護
- **capability削除**: 最小限の権限のみに制限

### sandbox-container
- **ネットワーク分離**: 外部通信を完全に遮断
- **リソース制限**: CPU/メモリ使用量を制限

### Docker Socket
- **読み取り専用マウント**: 書き込み権限なしでマウント

## 起動

```bash
docker compose up --build
```

## アクセス

ブラウザで以下のURLを開きます:

```
http://localhost
```

## 停止

```bash
docker compose down
```
