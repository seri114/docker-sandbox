# Sandbox Controller

DockerコンテナでPythonコードを安全に実行するGoベースのコントローラー。

## 開発

### 依存関係

```bash
go mod download
```

### ビルド

```bash
go build -o controller .
```

### フォーマット

```bash
gofmt -w .
```

### テスト

```bash
# 単体テスト
go test ./...

# 統合テスト（Dockerが必要）
go test -v ./... -run Integration

# E2Eテスト（Dockerが必要）
go test -v ./... -run E2E
```

### 実行

```bash
# ローカル実行（Dockerが必要）
./controller

# ポート指定
DOCKER_HOST=unix:///var/run/docker.sock ./controller
```
