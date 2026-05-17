# Docker Python Sandbox

Dockerコンテナ内でPythonコードを安全に実行するPOCシステム

## セキュリティ

### sandbox-controller
- **Alpineベース**: 最小攻撃表面積のベースイメージを使用
- **非rootユーザー**: nobodyユーザーで実行

### sandbox-container
- **非rootユーザー**: nobodyユーザーで実行（UID 65534）
- **capability削除**: 全ケーパビリティを削除（CapDrop: ALL）
- **ネットワーク分離**: 外部通信を完全に遮断（NetworkMode: none）
- **リソース制限**: CPU/メモリ使用量を制限
- **Tmpfs**: /tmpを一時ファイルシステムとしてマウント

### Docker Socket
- **読み取り専用マウント**: 書き込み権限なしでマウント

## アーキテクチャ

```
┌─────────┐     ┌──────────────────┐     ┌─────────────┐
│  WebUI  │────▶│ Test Client      │────▶│ Controller  │
└─────────┘     │ (FastAPI)        │     │ (Go)        │
                └──────────────────┘     └──────┬──────┘
                                                 │
                                                 ▼
                                          ┌──────────┐
                                          │  Docker  │
                                          │  Socket  │
                                          └──────────┘
```

## 起動

```bash
docker compose up --build
```

**注意**: Docker Desktop for Macの場合、ポート8080で競合が発生する可能性があります。
ポート18080が使用されます。

## アクセス

- **Web UI**: http://localhost
- **Controller API**: http://localhost:18080
- **Test Client**: http://localhost:8000

## API エンドポイント

### コンテナ作成
```bash
curl -X POST http://localhost:18080/containers/create \
  -H "Content-Type: application/json" \
  -d '{
    "image": "python:3.12-alpine",
    "code": "print(\"Hello\")",
    "memory": 134217768,
    "cpu": 0.5
  }'
```

### コンテナ起動（非同期）
```bash
curl -X POST http://localhost:18080/containers/start \
  -H "Content-Type: application/json" \
  -d '{
    "container_id": "...",
    "code": "print(\"Hello\")",
    "timeout": 30
  }'
```

### ログ取得（SSE）
```bash
curl "http://localhost:18080/containers/logs?id=..."
```

## 停止

```bash
docker compose down
```

## 既知の問題

### Docker Desktop for Mac
- **Unix Socketプロキシ**: ContainerStart APIが遅延する問題があります
- **回避策**: 非同期実装により即座にレスポンスを返します
- **ポートマッピング**: 一部の環境でポート8080が正常にバインドされない場合があります
