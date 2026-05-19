# Docker Python Sandbox

Dockerコンテナ内でPythonコードを安全に実行するPOCシステム

## セキュリティ

### sandbox-container
- **非rootユーザー**: nobodyユーザーで実行（UID 65534）
- **capability削除**: 全ケーパビリティを削除（CapDrop: ALL）
- **ネットワーク分離**: 外部通信を完全に遮断（NetworkMode: none）
- **リソース制限**: CPU/メモリ使用量を制限
- **Tmpfs**: /tmpを一時ファイルシステムとしてマウント
- **読み取り専用ルート**: ファイルシステムへの書き込みを禁止（ReadonlyRootfs: true）
- **権限昇格防止**: setuid/setgidによる権限昇格を防止（no-new-privileges）
- **PID制限**: Fork bomb攻撃を防止（PidsLimit: 100）
- **スワップ無効化**: メモリ制限を厳格化（MemorySwap: -1）
- **入力検証**: コードサイズ制限（1MB）とパラメータ検証
- **Seccomp**: システムコール制限（ネットワーク関連syscall削除）

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

## 開発ツール

### Pre-commit (コード品質自動チェック)

commit時に自動でフォーマットとリントを実行します。

```bash
# インストール（初回のみ）
uv pip install pre-commit
pre-commit install

# 手動実行
pre-commit run --all-files

# スキップしてcommit
git commit --no-verify
```

**チェック内容:**
- Go: `gofmt`, `go vet`
- Python: `ruff` (フォーマット + リント)
- 全体: 末尾空白、ファイル末尾改行、大ファイル警告

### Python (test-client)
- **uv**: 高速なパッケージマネージャー
- **ruff**: 高速なPython linter/formatter

```bash
cd test-client

# 依存関係インストール
uv sync

# Lintチェック
ruff check .

# コードフォーマット
ruff format .
```

### Go (sandbox-controller)
- **gofmt**: 標準のフォーマッタ

```bash
cd sandbox-controller

# フォーマット
gofmt -w .
```

## 既知の問題

### Docker Desktop for Mac
- **Unix Socketプロキシ**: ContainerStart APIが遅延する問題があります
- **回避策**: 非同期実装により即座にレスポンスを返します
- **ポートマッピング**: 一部の環境でポート8080が正常にバインドされない場合があります

## テスト

### Goテスト（sandbox-controller）

```bash
# 単体テスト
cd sandbox-controller
go test ./...

# 統合テスト（Dockerが必要）
go test -v ./... -run Integration

# E2Eテスト（Dockerが必要）
go test -v ./... -run E2E
```

### Pythonテスト（test-client）

```bash
# pytestが必要
uv pip install pytest pytest-asyncio httpx

# 単体テスト
cd test-client
pytest

# E2Eテスト（controllerが必要）
pytest tests/test_e2e.py -v
```

### テストカバレッジ

| テスト種類 | ファイル | 説明 |
|------------|---------|------|
| 単体テスト | `*_test.go` | モックを使った個別機能のテスト |
| 統合テスト | `*_integration_test.go` | 実際のDockerを使ったテスト |
| E2Eテスト | `e2e_test.go` | 完全なフローのエンドツーエンドテスト |
