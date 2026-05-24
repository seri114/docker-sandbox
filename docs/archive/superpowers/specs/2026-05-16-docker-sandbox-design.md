# Docker Python Sandbox Design

**Date**: 2026-05-16
**Status**: Approved (Security Hardened)
**Author**: Claude + User

## Overview

Dockerコンテナ内でPythonコードを安全に実行するPOCシステム。ネットワーク分離、ファイルシステム分離、悪意あるコード対策を備え、REST API経由で操作可能。

**セキュリティ主眼のPOC**: 可能な限りセキュアな構成で実装する。

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Docker Network: sandbox-net                  │
│                                                                  │
│  ┌─────────────┐    ┌──────────────────┐    ┌──────────────────┐│
│  │    nginx    │    │  test-client     │    │ sandbox-controller│
│  │  :80        │───▶│  :8000           │───▶│  (Go, secured)   ││
│  │  /static/*  │    │  (FastAPI)       │    │  :ro /var/run/...││
│  └─────────────┘    └──────────────────┘    └──────────────────┘│
│                                                    │            │
│                           stdinでコード送信  ▼            │
│                                          ┌──────────────────┐  │
│                                          │ sandbox-container│  │
│                                          │ network=none     │  │
│                                          │ mem=128MB        │  │
│                                          │ cpu=0.5          │  │
│                                          │ read-only        │  │
│                                          └──────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Components

### 1. sandbox-controller

**Language**: Go
**Base Image**: `gcr.io/distroless/static:nonroot`
**Security**: 多層防御

#### Security Configuration

| Layer | Setting | Effect |
|-------|---------|--------|
| **Image** | distroless static:nonroot | shellなし、非rootユーザー |
| **User** | nonroot:65532 | root権限なし |
| **Rootfs** | `--read-only` + `--tmpfs /tmp` | 書き込み不可能 |
| **Capabilities** | `--cap-drop=ALL` | 全Linux権限削除 |
| **Privileges** | `--security-opt=no-new-privileges` | 権限昇格防止 |
| **Seccomp** | ホワイトリストプロファイル | 不要なsyscall禁止 |
| **AppArmor** | docker-default以上 | ファイルアクセス制限 |
| **Socket** | `/var/run/docker.sock:ro` | 読取専用マウント |

#### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/containers/create` | POST | コンテナ作成、container_id返却 |
| `/containers/start` | POST | stdin経由でコード送信＆実行開始 |
| `/containers/stop` | POST | コンテナ停止・削除 |
| `/containers/logs` | GET (SSE) | stdout/stderrのリアルタイム配信 |

#### Request/Response

```json
// POST /containers/create
// Request: {}
// Response: { "container_id": "abc123" }

// POST /containers/start
// Request: { "container_id": "abc123", "code": "print('hello')", "timeout": 30 }
// Response: { "status": "running" }

// POST /containers/stop
// Request: { "container_id": "abc123" }
// Response: { "status": "stopped" }

// GET /containers/logs?id=abc123
// Response: SSE stream (data: { "stream": "stdout|stderr", "data": "..." })
```

#### Go Code Security

| 項目 | 実装 |
|------|------|
| 入力バリデーション | コードサイズ上限(1MB)、形式チェック |
| 操作ホワイトリスト | 許可するDocker APIを限定 |
| タイムアウト | 全API呼び出しにタイムアウト設定 |
| ログ/監査 | 全操作をログ記録 |

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

### 3. sandbox-container

**Base Image**: python:3.12-alpine
**Lifecycle**: 都度起動・実行後削除
**Input**: stdin経由でPythonコードを受信

#### Security Configuration

| Resource | Setting |
|----------|---------|
| **Network** | `none` |
| **Rootfs** | `--read-only` + `--tmpfs /tmp` |
| **User** | `nonroot` |
| **Capabilities** | `--cap-drop=ALL` |
| **Memory** | 128MB |
| **CPU** | 0.5 cores |
| **Timeout** | 0-60 seconds |
| **Seccomp** | Python実行に必要なsyscallのみ |

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
  const msg = JSON.parse(e.data);
  output.textContent += msg.data + '\n';
};
evtSource.onerror = () => { evtSource.close(); };
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

## Security Measures (Deep Dive)

### sandbox-controller 多層防御

```
┌─────────────────────────────────────────────────────────────┐
│ Layer 1: Container Configuration                            │
│ - distroless image (no shell)                               │
│ - nonroot user                                              │
│ - read-only rootfs                                          │
│ - capabilities drop ALL                                     │
├─────────────────────────────────────────────────────────────┤
│ Layer 2: Runtime Security                                   │
│ - seccomp whitelist profile                                 │
│ - AppArmor profile                                          │
│ - no-new-privileges                                         │
├─────────────────────────────────────────────────────────────┤
│ Layer 3: Socket Access                                      │
│ - /var/run/docker.sock:ro (read-only mount)                │
├─────────────────────────────────────────────────────────────┤
│ Layer 4: Code Security                                      │
│ - Input validation                                          │
│ - API whitelist                                             │
│ - Timeout on all operations                                 │
│ - Audit logging                                             │
└─────────────────────────────────────────────────────────────┘
```

### sandbox-container 隔離

```
┌─────────────────────────────────────────────────────────────┐
│ Network Isolation: network=none                             │
│ - No external communication                                 │
│ - No inter-container communication                          │
├─────────────────────────────────────────────────────────────┤
│ Filesystem Isolation:                                       │
│ - Read-only rootfs                                          │
│ - Only /tmp is writable (tmpfs)                             │
│ - No volume mounts                                          │
├─────────────────────────────────────────────────────────────┤
│ Process Isolation:                                          │
│ - Non-root user                                             │
│ - No capabilities                                           │
│ - Seccomp filter                                            │
├─────────────────────────────────────────────────────────────┤
│ Resource Limits:                                            │
│ - Memory: 128MB hard limit                                  │
│ - CPU: 0.5 cores quota                                      │
│ - Timeout: 60s max                                          │
└─────────────────────────────────────────────────────────────┘
```

### Residual Risks

| 脅威 | 緩和策 | 残リスク |
|------|--------|---------|
| controller乗っ取り | 多層防御 | リスク低減但不能ゼロ |
| ホストDocker操作 | 読取専用socket | 依然として可能 |
| 情報漏洩 | ネットワーク分離 | ネットワーク経由は防止 |
| DoS | リソース制限 | ホストリソース保護 |

## Docker Compose Configuration

```yaml
services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./webui:/usr/share/nginx/html:ro
    networks:
      - sandbox-net
    read_only: true
    security_opt:
      - no-new-privileges:true

  test-client:
    build: ./test-client
    ports:
      - "8000:8000"
    networks:
      - sandbox-net
    environment:
      - SANDBOX_CONTROLLER_URL=http://sandbox-controller:8080
    depends_on:
      - sandbox-controller

  sandbox-controller:
    build: ./sandbox-controller
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    networks:
      - sandbox-net
    read_only: true
    tmpfs:
      - /tmp:mode=1777
    security_opt:
      - no-new-privileges:true
      - seccomp:seccomp-profile.json
    cap_drop:
      - ALL
    user: "65532:65532"

networks:
  sandbox-net:
    driver: bridge
    internal: true  # 外部ネットワーク分離
```

## Error Handling

| Error | Handling |
|-------|----------|
| タイムアウト | コンテナ強制停止、SSEでtimeoutイベント送信 |
| メモリ不足 | コンテナOOMで停止、エラー通知 |
| 構文エラー | stderrに出力、SSEで配信 |
| コンテナ作成失敗 | 500エラー返却 |
| APIタイムアウト | 適切なHTTPステータスコード返却 |

## Success Criteria

- [x] PythonコードをWeb UIから入力・実行できる
- [x] 実行結果をリアルタイムに表示できる
- [x] ネットワークアクセスがブロックされている
- [x] リソース制限が適用されている
- [x] タイムアウトで確実に停止する
- [x] sandbox-controllerはセキュアに構成される
- [x] 残留リスクが文書化されている
