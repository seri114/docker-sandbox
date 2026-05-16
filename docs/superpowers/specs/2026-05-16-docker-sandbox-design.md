# Docker Python Sandbox Design

**Date**: 2026-05-16
**Status**: Approved
**Author**: Claude + User

## Overview

Dockerコンテナ内でPythonコードを安全に実行するPOCシステム。ネットワーク分離、ファイルシステム分離、悪意あるコード対策を備え、REST API経由で操作可能。

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Docker Network: sandbox-net                  │
│                                                                  │
│  ┌─────────────┐    ┌──────────────────┐    ┌──────────────────┐│
│  │    nginx    │    │  test-client     │    │ sandbox-controller│
│  │  :80        │───▶│  :8000           │───▶│  :8080 (内部)    ││
│  │  /static/*  │    │  (FastAPI)       │    │  /var/run/docker.sock│
│  └─────────────┘    └──────────────────┘    └──────────────────┘│
│                                                    │            │
│                           stdinでコード送信  ▼            │
│                                          ┌──────────────────┐  │
│                                          │ sandbox-container│  │
│                                          │ network=none     │  │
│                                          │ mem=128MB        │  │
│                                          │ cpu=0.5          │  │
│                                          └──────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Components

### 1. sandbox-controller

**Language**: Go
**Base Image**: distroless (shell-less)
**Responsibilities**:
- Docker Unix Socket経由のコンテナ操作
- REST APIの提供
- stdin経由でのコード送信
- ログのSSEストリーミング

**API Endpoints**:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/containers/create` | POST | コンテナ作成、container_id返却 |
| `/containers/start` | POST | stdin経由でコード送信＆実行開始 |
| `/containers/stop` | POST | コンテナ停止・削除 |
| `/containers/logs` | GET (SSE) | stdout/stderrのリアルタイム配信 |

**Request/Response**:

```json
// POST /containers/create
// Request: {}
// Response: { "container_id": "abc123" }

// POST /containers/start
// Request: { "container_id": "abc123", "code": "print('hello')" }
// Response: { "status": "running" }

// POST /containers/stop
// Request: { "container_id": "abc123" }
// Response: { "status": "stopped" }

// GET /containers/logs?id=abc123
// Response: SSE stream
```

### 2. test-client

**Language**: Python (FastAPI)
**Base Image**: python:3.12-slim
**Responsibilities**:
- Web UIとsandbox-controllerの仲介
- Pythonコードの受付
- SSE出力のフォワード
- タイムアウト管理

**API Endpoints**:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/static/*` | GET | Web UI静的ファイル |
| `/api/execute` | POST | 実行リクエスト受付 |
| `/api/execute/stream` | GET (SSE) | リアルタイム出力配信 |

**Request/Response**:

```json
// POST /api/execute
// Request: { "code": "print('hello')", "timeout": 30 }
// Response: { "execution_id": "exec_123" }

// GET /api/execute/stream?id=exec_123
// Response: SSE stream (stdout/stderr)
```

### 3. sandbox-container

**Base Image**: python:3.12-alpine
**Lifecycle**: 都度起動・実行後削除
**Input**: stdin経由でPythonコードを受信

**Resource Limits**:

| Resource | Value |
|----------|-------|
| Network | `none` |
| Memory | 128MB |
| CPU | 0.5 cores |
| Timeout | 0-60 seconds (configurable) |

### 4. Web UI

**Technology**: HTML + CSS + Vanilla JavaScript
**Features**:
- コード入力エリア（textarea）
- タイムアウト指定（input: number, 0-60）
- 実行ボタン
- リアルタイム出力表示（pre: white-space: pre-wrap）

**JavaScript (EventSource)**:

```javascript
const evtSource = new EventSource('/api/execute/stream?id=' + executionId);
evtSource.onmessage = (e) => {
  output.textContent += e.data + '\n';
};
```

## Data Flow

```
1. User → Web UI: コード入力、実行ボタンクリック
2. Web UI → test-client: POST /api/execute
3. test-client → sandbox-controller: POST /containers/create
4. test-client → sandbox-controller: POST /containers/start (with code)
5. sandbox-controller → sandbox-container: stdin経由でコード送信
6. sandbox-container → stdout/stderr 出力
7. sandbox-controller → test-client: SSE /containers/logs
8. test-client → Web UI: SSE /api/execute/stream
9. Web UI: リアルタイム表示
10. 終了時: test-client → sandbox-controller: POST /containers/stop
11. コンテナ削除
```

## Security Measures

| Threat | Mitigation |
|--------|------------|
| ネットワークアクセス | `network=none` で完全分離 |
| ファイルシステムアクセス | 読み取り専用ルートfs、一時ボリュームのみ |
| リソース枯渇 | メモリ128MB、CPU 0.5コア、タイムアウト60秒 |
| 無限ループ | タイムアウトで強制停止 |
| sandbox-controller乗っ取り | shell-lessイメージ、最小構成 |

## Docker Compose Configuration

```yaml
services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./webui:/usr/share/nginx/html
    networks:
      - sandbox-net

  test-client:
    build: ./test-client
    ports:
      - "8000:8000"
    networks:
      - sandbox-net

  sandbox-controller:
    build: ./sandbox-controller
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    networks:
      - sandbox-net

networks:
  sandbox-net:
    driver: bridge
```

## Error Handling

| Error | Handling |
|-------|----------|
| タイムアウト | コンテナ強制停止、SSEでtimeoutイベント送信 |
| メモリ不足 | コンテナOOMで停止、エラー通知 |
| 構文エラー | stderrに出力、SSEで配信 |
| コンテナ作成失敗 | 500エラー返却 |

## Success Criteria

- [x] PythonコードをWeb UIから入力・実行できる
- [x] 実行結果をリアルタイムに表示できる
- [x] ネットワークアクセスがブロックされている
- [x] リソース制限が適用されている
- [x] タイムアウトで確実に停止する
