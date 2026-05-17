# Test Client

FastAPIベースのSandbox Controllerテストクライアント

## 開発

### 依存関係インストール

```bash
uv sync
```

### Lint & Format

```bash
# Lintチェック
ruff check .

# 自動修正
ruff check --fix .

# フォーマット
ruff format .
```

### テスト

```bash
# 全テスト
uv run pytest

# E2Eテスト（Dockerが必要）
uv run pytest tests/test_e2e.py -v
```

### ローカル実行

```bash
uv run dev
```

http://localhost:8000 でアクセス可能。
