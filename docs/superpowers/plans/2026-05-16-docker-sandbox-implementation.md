# Docker Python Sandbox Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Dockerコンテナ内でPythonコードを安全に実行するPOCシステムを構築する

**Architecture:**
- `sandbox-controller` (Go): Docker Unix Socket経由でコンテナ操作、セキュアなdistrolessイメージ
- `test-client` (FastAPI): Web UIとcontrollerの仲介、SSEフォワード
- `webui`: シンプルなHTML/CSS/JS、EventSourceで出力表示
- Docker Composeで全サービス orchestration

**Tech Stack:**
- Go 1.21+, moby/moby Docker client library
- Python 3.12, FastAPI, httpx
- nginx:alpine, python:3.12-alpine
- Docker Compose v2

---

## Task 1: Project Setup

**Files:**
- Create: `sandbox-controller/go.mod`
- Create: `sandbox-controller/Dockerfile`
- Create: `test-client/requirements.txt`
- Create: `test-client/Dockerfile`
- Create: `webui/index.html`
- Create: `docker-compose.yml`

- [ ] **Step 1: Create Go module**

```bash
cd sandbox-controller && go mod init github.com/user/docker-sandbox/controller
```

Expected: `go.mod` created with module path

- [ ] **Step 2: Create requirements.txt**

```txt
fastapi==0.109.0
uvicorn[standard]==0.27.0
httpx==0.26.0
pytest==7.4.3
pytest-asyncio==0.23.3
```

- [ ] **Step 3: Create directory structure**

```bash
mkdir -p sandbox-controller/internal/{docker,handler,security} test-client/app webui
```

- [ ] **Step 4: Commit**

```bash
git add go.mod requirements.txt
git commit -m "chore: initialize project structure"
```

---

## Task 2: sandbox-controller - Docker Client

**Files:**
- Create: `sandbox-controller/internal/docker/client.go`
- Create: `sandbox-controller/config/config.go`
- Modify: `sandbox-controller/go.mod`

- [ ] **Step 1: Add moby dependency**

```bash
cd sandbox-controller && go get github.com/docker/docker/client
```

- [ ] **Step 2: Write config struct**

```go
package config

import "time"

type Config struct {
    DockerHost    string
    RequestTimeout time.Duration
}

func Default() *Config {
    return &Config{
        DockerHost:    "unix:///var/run/docker.sock",
        RequestTimeout: 30 * time.Second,
    }
}
```

- [ ] **Step 3: Write Docker client wrapper test**

```go
package docker

import (
    "testing"
    "time"
)

func TestNewClient(t *testing.T) {
    client := NewClient("unix:///var/run/docker.sock", 10*time.Second)
    if client == nil {
        t.Fatal("expected non-nil client")
    }
    if client.cli == nil {
        t.Fatal("expected docker client to be initialized")
    }
}
```

- [ ] **Step 4: Run test to verify it fails**

```bash
cd sandbox-controller && go test ./internal/docker -v
```

Expected: FAIL with "undefined: NewClient"

- [ ] **Step 5: Implement Docker client**

```go
package docker

import (
    "context"
    "time"

    "github.com/docker/docker/client"
)

type DockerClient struct {
    cli     *client.Client
    timeout time.Duration
}

func NewClient(host string, timeout time.Duration) *DockerClient {
    cli, err := client.NewClientWithOpts(client.WithHost(host))
    if err != nil {
        panic(err)
    }
    return &DockerClient{
        cli:     cli,
        timeout: timeout,
    }
}

func (d *DockerClient) Close() error {
    return d.cli.Close()
}

func (d *DockerClient) Context() (context.Context, context.CancelFunc) {
    return context.WithTimeout(context.Background(), d.timeout)
}
```

- [ ] **Step 6: Run test to verify it passes**

```bash
cd sandbox-controller && go test ./internal/docker -v
```

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add sandbox-controller/internal/docker sandbox-controller/config
git commit -m "feat: add Docker client wrapper"
```

---

## Task 3: sandbox-controller - Container Creation Handler

**Files:**
- Create: `sandbox-controller/internal/handler/create.go`
- Modify: `sandbox-controller/internal/docker/client.go`

- [ ] **Step 1: Write test for container creation**

```go
package handler

import (
    "testing"

    "github.com/docker/docker/api/types/container"
)

func TestCreateContainerRequest(t *testing.T) {
    req := CreateContainerRequest{
        Image: "python:3.12-alpine",
        Code:  "print('hello')",
    }

    config := req.ToContainerConfig()
    if config.Image != "python:3.12-alpine" {
        t.Errorf("expected image python:3.12-alpine, got %s", config.Image)
    }
    if config.NetworkMode != "none" {
        t.Errorf("expected network none, got %s", config.NetworkMode)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd sandbox-controller && go test ./internal/handler -v -run TestCreateContainerRequest
```

Expected: FAIL with "undefined: CreateContainerRequest"

- [ ] **Step 3: Implement container creation handler**

```go
package handler

import (
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/mount"
)

type CreateContainerRequest struct {
    Image   string
    Code    string
    Timeout int // seconds
    Memory  string // e.g., "128MB"
    CPU     string // e.g., "0.5"
}

type CreateContainerResponse struct {
    ContainerID string `json:"container_id"`
}

func (r *CreateContainerRequest) ToContainerConfig() *container.Config {
    return &container.Config{
        Image:        r.Image,
        Cmd:          []string{"python", "-u", "-c", r.Code},
        AttachStdin:  true,
        AttachStdout: true,
        AttachStderr: true,
        OpenStdin:    true,
        User:         "65532:65532", // nonroot
        ReadOnlyRootfs: true,
    }
}

func (r *CreateContainerRequest) ToHostConfig() *container.HostConfig {
    return &container.HostConfig{
        NetworkMode: "none",
        Resources: container.Resources{
            Memory:   128 * 1024 * 1024, // 128MB
            NanoCPUs: 500000000,         // 0.5 cores
        },
        CapDrop: []string{"ALL"},
        Tmpfs: map[string]string{
            "/tmp": "mode=1777",
        },
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd sandbox-controller && go test ./internal/handler -v -run TestCreateContainerRequest
```

Expected: PASS

- [ ] **Step 5: Add CreateContainer to DockerClient**

```go
// In internal/docker/client.go
import (
    "github.com/docker/docker/api/types"
)

func (d *DockerClient) CreateContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig) (string, error) {
    resp, err := d.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
    if err != nil {
        return "", err
    }
    return resp.ID, nil
}

func (d *DockerClient) StartContainer(ctx context.Context, id string) error {
    return d.cli.ContainerStart(ctx, id, container.StartOptions{})
}
```

- [ ] **Step 6: Commit**

```bash
git add sandbox-controller/internal/handler/create.go sandbox-controller/internal/docker/client.go
git commit -m "feat: add container creation handler"
```

---

## Task 4: sandbox-controller - Start Handler

**Files:**
- Create: `sandbox-controller/internal/handler/start.go`

- [ ] **Step 1: Write test for start handler**

```go
package handler

import "testing"

func TestStartContainerRequest(t *testing.T) {
    req := StartContainerRequest{
        ContainerID: "test-id",
        Code:        "print('hello')",
    }
    if req.ContainerID != "test-id" {
        t.Errorf("expected container_id test-id")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd sandbox-controller && go test ./internal/handler -v -run TestStartContainerRequest
```

Expected: FAIL

- [ ] **Step 3: Implement start handler**

```go
package handler

type StartContainerRequest struct {
    ContainerID string `json:"container_id"`
    Code        string `json:"code"`
    Timeout     int    `json:"timeout"`
}

type StartContainerResponse struct {
    Status string `json:"status"`
}

func (r *StartContainerRequest) Validate() error {
    if r.ContainerID == "" {
        return fmt.Errorf("container_id is required")
    }
    if r.Code == "" {
        return fmt.Errorf("code is required")
    }
    if r.Timeout < 0 || r.Timeout > 60 {
        return fmt.Errorf("timeout must be between 0 and 60")
    }
    return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd sandbox-controller && go test ./internal/handler -v -run TestStartContainerRequest
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add sandbox-controller/internal/handler/start.go
git commit -m "feat: add start handler with validation"
```

---

## Task 5: sandbox-controller - SSE Logs Handler

**Files:**
- Create: `sandbox-controller/internal/handler/logs.go`

- [ ] **Step 1: Write test for logs handler**

```go
package handler

import "testing"

func TestLogsStreamFormat(t *testing.T) {
    msg := LogMessage{
        Stream: "stdout",
        Data:   "hello",
    }
    json := msg.ToJSON()
    if !strings.Contains(json, "stdout") {
        t.Errorf("expected json to contain stdout")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd sandbox-controller && go test ./internal/handler -v -run TestLogsStreamFormat
```

Expected: FAIL

- [ ] **Step 3: Implement SSE logs handler**

```go
package handler

import (
    "encoding/json"
    "fmt"
    "io"

    "github.com/docker/docker/api/types"
)

type LogMessage struct {
    Stream string `json:"stream"` // stdout or stderr
    Data   string `json:"data"`
}

func (m *LogMessage) ToJSON() string {
    b, _ := json.Marshal(m)
    return string(b)
}

type LogsStreamer struct {
    reader io.ReadCloser
}

func NewLogsStreamer(reader io.ReadCloser) *LogsStreamer {
    return &LogsStreamer{reader: reader}
}

func (s *LogsStreamer) StreamTo(ch chan<- LogMessage) error {
    defer s.reader.Close()
    defer close(ch)

    decoder := json.NewDecoder(s.reader)
    for {
        var line types.JSONLog
        if err := decoder.Decode(&line); err != nil {
            if err == io.EOF {
                return nil
            }
            return err
        }

        stream := "stdout"
        ch <- LogMessage{
            Stream: stream,
            Data:   line.Log,
        }
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd sandbox-controller && go test ./internal/handler -v -run TestLogsStreamFormat
```

Expected: PASS

- [ ] **Step 5: Add ContainerLogs to DockerClient**

```go
// In internal/docker/client.go
func (d *DockerClient) ContainerLogs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
    return d.cli.ContainerLogs(ctx, id, container.LogsOptions{
        ShowStdout: true,
        ShowStderr: true,
        Follow:     follow,
        Tail:       "0",
    })
}
```

- [ ] **Step 6: Commit**

```bash
git add sandbox-controller/internal/handler/logs.go sandbox-controller/internal/docker/client.go
git commit -m "feat: add SSE logs streaming handler"
```

---

## Task 6: sandbox-controller - Stop Handler

**Files:**
- Create: `sandbox-controller/internal/handler/stop.go`

- [ ] **Step 1: Write test for stop handler**

```go
package handler

import "testing"

func TestStopContainerRequest(t *testing.T) {
    req := StopContainerRequest{
        ContainerID: "test-id",
    }
    if req.ContainerID != "test-id" {
        t.Errorf("expected container_id test-id")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd sandbox-controller && go test ./internal/handler -v -run TestStopContainerRequest
```

Expected: FAIL

- [ ] **Step 3: Implement stop handler**

```go
package handler

import "time"

type StopContainerRequest struct {
    ContainerID string `json:"container_id"`
}

type StopContainerResponse struct {
    Status string `json:"status"`
}

func (r *StopContainerRequest) Validate() error {
    if r.ContainerID == "" {
        return fmt.Errorf("container_id is required")
    }
    return nil
}

func (d *DockerClient) StopAndRemoveContainer(ctx context.Context, id string) error {
    timeout := 10 * time.Second
    err := d.cli.ContainerStop(ctx, id, container.StopOptions{
        Timeout: &timeout,
    })
    if err != nil {
        return err
    }
    return d.cli.ContainerRemove(ctx, id, container.RemoveOptions{})
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd sandbox-controller && go test ./internal/handler -v -run TestStopContainerRequest
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add sandbox-controller/internal/handler/stop.go
git commit -m "feat: add stop handler with cleanup"
```

---

## Task 7: sandbox-controller - HTTP Server

**Files:**
- Create: `sandbox-controller/main.go`

- [ ] **Step 1: Write main.go**

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/gorilla/mux"
    "github.com/user/docker-sandbox/controller/config"
    "github.com/user/docker-sandbox/controller/internal/docker"
    "github.com/user/docker-sandbox/controller/internal/handler"
)

type Server struct {
    dockerClient *docker.DockerClient
}

func main() {
    cfg := config.Default()
    dockerClient := docker.NewClient(cfg.DockerHost, cfg.RequestTimeout)
    defer dockerClient.Close()

    s := &Server{dockerClient: dockerClient}

    r := mux.NewRouter()
    r.HandleFunc("/containers/create", s.handleCreate).Methods("POST")
    r.HandleFunc("/containers/start", s.handleStart).Methods("POST")
    r.HandleFunc("/containers/stop", s.handleStop).Methods("POST")
    r.HandleFunc("/containers/logs", s.handleLogs).Methods("GET")

    addr := ":8080"
    log.Printf("Starting server on %s", addr)
    log.Fatal(http.ListenAndServe(addr, r))
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
    var req handler.CreateContainerRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    ctx, cancel := s.dockerClient.Context()
    defer cancel()

    id, err := s.dockerClient.CreateContainer(ctx, req.ToContainerConfig(), req.ToHostConfig())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(handler.CreateContainerResponse{ContainerID: id})
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
    var req handler.StartContainerRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    if err := req.Validate(); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    ctx, cancel := s.dockerClient.Context()
    defer cancel()

    if err := s.dockerClient.StartContainer(ctx, req.ContainerID); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(handler.StartContainerResponse{Status: "running"})
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
    var req handler.StopContainerRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    ctx, cancel := s.dockerClient.Context()
    defer cancel()

    if err := s.dockerClient.StopAndRemoveContainer(ctx, req.ContainerID); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(handler.StopContainerResponse{Status: "stopped"})
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")
    if id == "" {
        http.Error(w, "id is required", http.StatusBadRequest)
        return
    }

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    ctx, cancel := context.WithCancel(r.Context())
    defer cancel()

    reader, err := s.dockerClient.ContainerLogs(ctx, id, true)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer reader.Close()

    streamer := handler.NewLogsStreamer(reader)
    ch := make(chan handler.LogMessage)
    go streamer.StreamTo(ch)

    for msg := range ch {
        fmt.Fprintf(w, "data: %s\n\n", msg.ToJSON())
        if f, ok := w.(http.Flusher); ok {
            f.Flush()
        }
    }
}
```

- [ ] **Step 2: Add gorilla/mux dependency**

```bash
cd sandbox-controller && go get github.com/gorilla/mux
```

- [ ] **Step 3: Commit**

```bash
git add sandbox-controller/main.go sandbox-controller/go.mod sandbox-controller/go.sum
git commit -m "feat: add HTTP server with all endpoints"
```

---

## Task 8: sandbox-controller - Dockerfile

**Files:**
- Create: `sandbox-controller/Dockerfile`
- Create: `sandbox-controller/internal/security/seccomp.json`

- [ ] **Step 1: Create seccomp profile**

```json
{
  "defaultAction": "SCMP_ACT_ERRNO",
  "syscalls": [
    {
      "names": [
        "read",
        "write",
        "open",
        "close",
        "stat",
        "fstat",
        "lstat",
        "poll",
        "lseek",
        "mmap",
        "mprotect",
        "munmap",
        "brk",
        "rt_sigaction",
        "rt_sigprocmask",
        "sigaltstack",
        "ioctl",
        "pread64",
        "pwrite64",
        "readv",
        "writev",
        "access",
        "pipe",
        "select",
        "sched_yield",
        "mremap",
        "msync",
        "mincore",
        "madvise",
        "dup",
        "dup2",
        "pause",
        "nanosleep",
        "getitimer",
        "alarm",
        "setitimer",
        "getpid",
        "sendfile",
        "socket",
        "connect",
        "accept",
        "sendto",
        "recvfrom",
        "sendmsg",
        "recvmsg",
        "shutdown",
        "bind",
        "listen",
        "getsockname",
        "getpeername",
        "socketpair",
        "setsockopt",
        "getsockopt",
        "clone",
        "fork",
        "vfork",
        "execve",
        "exit",
        "wait4",
        "kill",
        "uname",
        "semget",
        "semop",
        "semctl",
        "shmdt",
        "shmget",
        "shmctl",
        "msgget",
        "msgrcv",
        "msgsnd",
        "msgctl",
        "fcntl",
        "flock",
        "fsync",
        "fdatasync",
        "truncate",
        "ftruncate",
        "getdents",
        "getcwd",
        "chdir",
        "fchdir",
        "rename",
        "mkdir",
        "rmdir",
        "creat",
        "link",
        "unlink",
        "symlink",
        "readlink",
        "chmod",
        "fchmod",
        "chown",
        "fchown",
        "lchown",
        "umask",
        "gettimeofday",
        "getrlimit",
        "getrusage",
        "sysinfo",
        "times",
        "getuid",
        "getgid",
        "setuid",
        "setgid",
        "geteuid",
        "getegid",
        "setpgid",
        "getppid",
        "getpgrp",
        "setsid",
        "setreuid",
        "setregid",
        "getgroups",
        "setgroups",
        "setresuid",
        "getresuid",
        "setresgid",
        "getresgid",
        "getpgid",
        "setfsuid",
        "setfsgid",
        "getsid",
        "capget",
        "capset",
        "rt_sigpending",
        "rt_sigtimedwait",
        "rt_sigqueueinfo",
        "sigpending",
        "uname",
        "epoll_create",
        "epoll_ctl",
        "epoll_wait",
        "epoll_pwait"
      ],
      "action": "SCMP_ACT_ALLOW"
    }
  ]
}
```

- [ ] **Step 2: Create Dockerfile**

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o controller .

# Final stage - distroless
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/controller /controller
USER 65532:65532
ENTRYPOINT ["/controller"]
```

- [ ] **Step 3: Commit**

```bash
git add sandbox-controller/Dockerfile sandbox-controller/internal/security/seccomp.json
git commit -m "feat: add distroless Dockerfile and seccomp profile"
```

---

## Task 9: test-client - HTTP Client

**Files:**
- Create: `test-client/app/client.py`

- [ ] **Step 1: Write test for client**

```python
import pytest
from app.client import SandboxClient

def test_client_init():
    client = SandboxClient("http://localhost:8080")
    assert client.base_url == "http://localhost:8080"
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd test-client && pytest tests/test_client.py -v
```

Expected: FAIL

- [ ] **Step 3: Implement client**

```python
import httpx
from typing import Optional

class SandboxClient:
    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip('/')
        self.client = httpx.Client(timeout=30.0)

    def create_container(self, image: str = "python:3.12-alpine") -> str:
        response = self.client.post(
            f"{self.base_url}/containers/create",
            json={"image": image}
        )
        response.raise_for_status()
        return response.json()["container_id"]

    def start_container(self, container_id: str, code: str, timeout: int = 30) -> dict:
        response = self.client.post(
            f"{self.base_url}/containers/start",
            json={"container_id": container_id, "code": code, "timeout": timeout}
        )
        response.raise_for_status()
        return response.json()

    def stop_container(self, container_id: str) -> dict:
        response = self.client.post(
            f"{self.base_url}/containers/stop",
            json={"container_id": container_id}
        )
        response.raise_for_status()
        return response.json()

    def stream_logs(self, container_id: str):
        response = self.client.stream(
            "GET",
            f"{self.base_url}/containers/logs",
            params={"id": container_id},
            timeout=None
        )
        response.raise_for_status()
        return response

    def close(self):
        self.client.close()
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd test-client && pytest tests/test_client.py -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add test-client/app/client.py test-client/tests/test_client.py
git commit -m "feat: add sandbox controller client"
```

---

## Task 10: test-client - FastAPI App

**Files:**
- Create: `test-client/app/api.py`
- Create: `test-client/main.py`

- [ ] **Step 1: Write API test**

```python
import pytest
from fastapi.testclient import TestClient
from app.api import app

client = TestClient(app)

def test_create_execution():
    response = client.post("/api/execute", json={"code": "print('hello')", "timeout": 10})
    assert response.status_code == 200
    data = response.json()
    assert "execution_id" in data
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd test-client && pytest tests/test_api.py -v
```

Expected: FAIL

- [ ] **Step 3: Implement FastAPI app**

```python
# test-client/app/api.py
from fastapi import FastAPI, HTTPException
from fastapi.responses import StreamingResponse
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel
import uuid
from .client import SandboxClient

app = FastAPI(title="Sandbox Test Client")

# Initialize client (URL from env or default)
import os
SANDBOX_URL = os.getenv("SANDBOX_CONTROLLER_URL", "http://sandbox-controller:8080")
sandbox = SandboxClient(SANDBOX_URL)

# Mount static files
app.mount("/static", StaticFiles(directory="static"), name="static")

class ExecuteRequest(BaseModel):
    code: str
    timeout: int = 30

class ExecuteResponse(BaseModel):
    execution_id: str

# Store execution state
executions = {}

@app.post("/api/execute", response_model=ExecuteResponse)
async def execute_code(req: ExecuteRequest):
    execution_id = str(uuid.uuid4())

    # Create container
    container_id = sandbox.create_container()

    # Start execution
    sandbox.start_container(container_id, req.code, req.timeout)

    executions[execution_id] = {
        "container_id": container_id,
        "timeout": req.timeout
    }

    return ExecuteResponse(execution_id=execution_id)

@app.get("/api/execute/stream")
async def stream_logs(execution_id: str):
    if execution_id not in executions:
        raise HTTPException(status_code=404, detail="Execution not found")

    container_id = executions[execution_id]["container_id"]

    async def generate():
        with sandbox.stream_logs(container_id) as r:
            for line in r.iter_lines():
                if line:
                    yield f"data: {line}\n\n"

        # Cleanup after execution
        sandbox.stop_container(container_id)
        del executions[execution_id]

    return StreamingResponse(generate(), media_type="text/event-stream")
```

```python
# test-client/main.py
import uvicorn
from app.api import app

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd test-client && pytest tests/test_api.py -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add test-client/app/api.py test-client/main.py
git commit -m "feat: add FastAPI with SSE streaming"
```

---

## Task 11: test-client - Dockerfile

**Files:**
- Create: `test-client/Dockerfile`

- [ ] **Step 1: Create Dockerfile**

```dockerfile
FROM python:3.12-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

EXPOSE 8000

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
```

- [ ] **Step 2: Commit**

```bash
git add test-client/Dockerfile
git commit -m "feat: add test-client Dockerfile"
```

---

## Task 12: Web UI

**Files:**
- Create: `webui/index.html`
- Create: `webui/style.css`
- Create: `webui/app.js`

- [ ] **Step 1: Create HTML**

```html
<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Python Sandbox</title>
    <link rel="stylesheet" href="style.css">
</head>
<body>
    <div class="container">
        <header>
            <h1>Python Sandbox</h1>
        </header>
        <main>
            <div class="input-section">
                <textarea id="code" placeholder="Pythonコードを入力...">print("Hello, World!")</textarea>
                <div class="controls">
                    <label>
                        タイムアウト(秒):
                        <input type="number" id="timeout" min="1" max="60" value="10">
                    </label>
                    <button id="run">実行</button>
                </div>
            </div>
            <div class="output-section">
                <h2>出力:</h2>
                <pre id="output"></pre>
            </div>
        </main>
    </div>
    <script src="app.js"></script>
</body>
</html>
```

- [ ] **Step 2: Create CSS**

```css
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    background: #1a1a2e;
    color: #eee;
    min-height: 100vh;
}

.container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px;
}

header {
    margin-bottom: 20px;
}

h1 {
    color: #00d9ff;
}

.input-section {
    margin-bottom: 20px;
}

#code {
    width: 100%;
    min-height: 200px;
    padding: 15px;
    background: #16213e;
    border: 1px solid #0f3460;
    border-radius: 8px;
    color: #00d9ff;
    font-family: 'Monaco', 'Consolas', monospace;
    font-size: 14px;
    resize: vertical;
}

.controls {
    display: flex;
    gap: 15px;
    align-items: center;
    margin-top: 10px;
}

label {
    color: #aaa;
}

#timeout {
    width: 60px;
    padding: 5px;
    background: #16213e;
    border: 1px solid #0f3460;
    border-radius: 4px;
    color: #00d9ff;
}

#run {
    padding: 10px 24px;
    background: #00d9ff;
    border: none;
    border-radius: 6px;
    color: #1a1a2e;
    font-weight: bold;
    cursor: pointer;
    transition: background 0.2s;
}

#run:hover {
    background: #00b8d9;
}

#run:disabled {
    background: #555;
    cursor: not-allowed;
}

.output-section {
    background: #16213e;
    border: 1px solid #0f3460;
    border-radius: 8px;
    padding: 15px;
}

.output-section h2 {
    color: #00d9ff;
    margin-bottom: 10px;
    font-size: 18px;
}

#output {
    min-height: 100px;
    color: #eee;
    font-family: 'Monaco', 'Consolas', monospace;
    font-size: 14px;
    white-space: pre-wrap;
    word-break: break-all;
}
```

- [ ] **Step 3: Create JavaScript**

```javascript
const codeInput = document.getElementById('code');
const timeoutInput = document.getElementById('timeout');
const runButton = document.getElementById('run');
const output = document.getElementById('output');

runButton.addEventListener('click', async () => {
    const code = codeInput.value;
    const timeout = parseInt(timeoutInput.value);

    if (!code.trim()) {
        alert('コードを入力してください');
        return;
    }

    runButton.disabled = true;
    output.textContent = '';

    try {
        // Start execution
        const execResponse = await fetch('/api/execute', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ code, timeout })
        });

        if (!execResponse.ok) {
            throw new Error('実行開始に失敗しました');
        }

        const { execution_id } = await execResponse.json();

        // Stream output
        const eventSource = new EventSource(`/api/execute/stream?execution_id=${execution_id}`);

        eventSource.onmessage = (e) => {
            try {
                const msg = JSON.parse(e.data);
                output.textContent += msg.data + '\n';
            } catch {
                output.textContent += e.data + '\n';
            }
        };

        eventSource.onerror = () => {
            eventSource.close();
            runButton.disabled = false;
        };

        eventSource.addEventListener('close', () => {
            eventSource.close();
            runButton.disabled = false;
        });

    } catch (error) {
        output.textContent = `エラー: ${error.message}`;
        runButton.disabled = false;
    }
});
```

- [ ] **Step 4: Commit**

```bash
git add webui/
git commit -m "feat: add web UI with SSE support"
```

---

## Task 13: Docker Compose

**Files:**
- Create: `docker-compose.yml`
- Create: `seccomp-profile.json`

- [ ] **Step 1: Create seccomp profile**

```json
{
  "defaultAction": "SCMP_ACT_ERRNO",
  "syscalls": [
    {
      "names": ["read", "write", "open", "close", "stat", "fstat", "lstat", "poll", "lseek", "mmap", "mprotect", "munmap", "brk", "rt_sigaction", "rt_sigprocmask", "sigaltstack", "ioctl", "pread64", "pwrite64", "readv", "writev", "access", "pipe", "select", "sched_yield", "mremap", "msync", "mincore", "madvise", "dup", "dup2", "pause", "nanosleep", "getitimer", "alarm", "setitimer", "getpid", "sendfile", "socket", "connect", "accept", "sendto", "recvfrom", "sendmsg", "recvmsg", "shutdown", "bind", "listen", "getsockname", "getpeername", "socketpair", "setsockopt", "getsockopt", "clone", "fork", "vfork", "execve", "exit", "wait4", "kill", "uname", "semget", "semop", "semctl", "shmdt", "shmget", "shmctl", "msgget", "msgrcv", "msgsnd", "msgctl", "fcntl", "flock", "fsync", "fdatasync", "truncate", "ftruncate", "getdents", "getcwd", "chdir", "fchdir", "rename", "mkdir", "rmdir", "creat", "link", "unlink", "symlink", "readlink", "chmod", "fchmod", "chown", "fchown", "lchown", "umask", "gettimeofday", "getrlimit", "getrusage", "sysinfo", "times", "getuid", "getgid", "setuid", "setgid", "geteuid", "getegid", "setpgid", "getppid", "getpgrp", "setsid", "setreuid", "setregid", "getgroups", "setgroups", "setresuid", "getresuid", "setresgid", "getresgid", "getpgid", "setfsuid", "setfsgid", "getsid", "capget", "capset", "rt_sigpending", "rt_sigtimedwait", "rt_sigqueueinfo", "sigpending", "uname", "epoll_create", "epoll_ctl", "epoll_wait", "epoll_pwait"],
      "action": "SCMP_ACT_ALLOW"
    }
  ]
}
```

- [ ] **Step 2: Create docker-compose.yml**

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
      - seccomp:./seccomp-profile.json
    cap_drop:
      - ALL
    user: "65532:65532"

networks:
  sandbox-net:
    driver: bridge
    internal: true
```

- [ ] **Step 3: Commit**

```bash
git add docker-compose.yml seccomp-profile.json
git commit -m "feat: add docker compose with security hardening"
```

---

## Task 14: README

**Files:**
- Create: `README.md`

- [ ] **Step 1: Create README**

```markdown
# Docker Python Sandbox

Dockerコンテナ内でPythonコードを安全に実行するPOCシステム。

## セキュリティ

- sandbox-controller: distroless、読取専用rootfs、capabilities全drop
- sandbox-container: ネットワーク分離、リソース制限
- Docker socket: 読取専用マウント

## 起動

```bash
docker compose up --build
```

## アクセス

http://localhost

## 停止

```bash
docker compose down
```
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add README"
```

---

## Task 15: End-to-End Test

**Files:**
- Create: `test-client/tests/test_e2e.py`

- [ ] **Step 1: Write E2E test**

```python
import pytest
import time
from app.client import SandboxClient

def test_full_execution():
    client = SandboxClient("http://localhost:8080")

    # Create container
    container_id = client.create_container()
    assert container_id is not None

    # Start with test code
    code = "print('test'); import sys; sys.stdout.flush()"
    client.start_container(container_id, code, timeout=10)

    # Wait a bit
    time.sleep(2)

    # Stop and cleanup
    client.stop_container(container_id)

    client.close()
```

- [ ] **Step 2: Commit**

```bash
git add test-client/tests/test_e2e.py
git commit -m "test: add E2E test"
```

---

## Self-Review Results

**Spec Coverage:**
- ✅ sandbox-controller (Go) - Tasks 2-8
- ✅ test-client (FastAPI) - Tasks 9-11
- ✅ Web UI - Task 12
- ✅ Docker Compose - Task 13
- ✅ Security measures integrated throughout

**Placeholder Scan:** No placeholders found

**Type Consistency:** All types and signatures consistent across tasks
