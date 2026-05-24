# Docker Sandbox - Streaming Examples

このディレクトリには、Docker Sandboxのストリーミング機能を活用したサンプルプログラムが含まれています。

## 特徴

- **`time.sleep` を活用したポーリング**: 定期的にログを取得してリアルタイム表示
- **実用的なストリーミングパターン**: コンテナ実行中のログを逐次表示
- **同期・非同期の両方に対応**: ユースケースに合わせて選択可能

## サンプルプログラム

### 1. streaming_example.py (同期版)

単一のコンテナのログを `time.sleep` を使って定期的にポーリングしながら表示します。

**使用例:**
```bash
cd test-client
python examples/streaming_example.py
```

**含まれる例:**
- 例1: シンプルな遅延出力
- 例2: 進行状況を示すプログレッシブ出力
- 例3: タイムスタンプ付きのリアルタイムモニタリング

**主な機能:**
- `time.sleep()` による定期的なログ取得
- SSE (Server-Sent Events) 形式の解析
- タイムスタンプ付きのログ表示
- キーボード割り込み (Ctrl+C) による graceful shutdown

### 2. async_streaming_example.py (非同期版)

非同期I/Oを使用して、複数のコンテナを同時にモニタリングします。

**使用例:**
```bash
cd test-client
python examples/async_streaming_example.py
```

**含まれる例:**
- 例1: 単一コンテナの非同期ストリーミング
- 例2: 複数コンテナの並行モニタリング

**主な機能:**
- `asyncio.sleep()` による非同期ポーリング
- 複数コンテナの同時モニタリング
- 非同期コンテキストマネージャーの使用
- 並行タスク管理

## コード例

### 同期版の基本的な使用方法

```python
from app.client import SandboxClient
import time
import json

client = SandboxClient("http://localhost:8000")

# コンテナ作成と実行
container_id = client.create_container(
    image="python:3.12-alpine",
    code="print('Hello World')"
)

client.start_container(container_id, code="print('Hello World')", timeout=30)

# ログのストリーミング
time.sleep(0.5)  # コンテナの起動を待機

with client.stream_logs(container_id) as response:
    for line in response.iter_lines():
        if line.startswith("data: "):
            msg = json.loads(line[6:])
            if "data" in msg and msg["data"]:
                print(msg["data"].strip())

# クリーンアップ
client.stop_container(container_id)
client.close()
```

### 非同期版の基本的な使用方法

```python
from app.client import AsyncSandboxClient
import asyncio
import json

async def main():
    client = AsyncSandboxClient("http://localhost:8000")

    # コンテナ作成と実行
    container_id = await client.create_container(
        image="python:3.12-alpine",
        code="print('Hello World')"
    )

    # ログのストリーミング
    response, http_client = await client.stream_logs(container_id)

    try:
        async for line in response.aiter_lines():
            if line.startswith("data: "):
                msg = json.loads(line[6:])
                if "data" in msg and msg["data"]:
                    print(msg["data"].strip())
    finally:
        await http_client.aclose()

if __name__ == "__main__":
    asyncio.run(main())
```

## 改善点

元の実装からの主な改善点:

1. **実際のストリーミング処理の実装**
   - `stream_logs()` はコンテキストマネージャーとして使用可能
   - `read_stream_lines()` メソッドで行ごとの読み取りをサポート

2. **time.sleep の活用**
   - 定期的なポーリングによるリアルタイムログ表示
   - 過度なリクエストを防ぐための適切な間隔設定

3. **実用的なエラーハンドリング**
   - キーボード割り込みへの対応
   - 適切なリソースクリーンアップ

4. **タイムスタンプと進行状況の表示**
   - 実行時間の計測
   - 視覚的に分かりやすい出力形式

## 依存関係

- httpx
- asyncio (非同期版)

## 実行環境

- Docker Sandbox Controller が実行されていること
- デフォルトでは `http://localhost:8000` で接続

## 注意点

- コンテナの実行には時間がかかる場合があるため、適切な `time.sleep()` 間隔を設定してください
- 長時間実行するコードの場合、`timeout` パラメータを調整してください
- 非同期版は複数のコンテナを同時にモニタリングする場合に特に有効です
