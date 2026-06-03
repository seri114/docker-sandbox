# =============================================================================
# Sandbox Controller Docker Images
# =============================================================================
# This Dockerfile defines multiple images for different purposes:
# - Runtime image: Lightweight for production Python execution
# - Builder image: Full-featured for CI/CD with Go and testing tools
#
# Build targets:
#   - runtime: Production image with Python only (lightweight)
#   - builder: CI/CD image with Python + Go + testing tools (full-featured)
#
# Usage:
#   docker build --target runtime -t sandbox:runtime -f sandbox-controller/docker/Dockerfile .
#   docker build --target builder -t sandbox:latest -f sandbox-controller/docker/Dockerfile .
# =============================================================================

# =============================================================================
# 本番用イメージ（軽量）
# =============================================================================
# Used for production Python code execution in sandboxed containers.
# Contains only Python and minimal dependencies.
# Size: ~50MB
# =============================================================================
FROM python:3.14-alpine@sha256:5a824eb82cc75361f98611f3cfc5091ea33f10a6ccea4d4ebdabbc523b9a1614 AS runtime

LABEL maintainer="docker-sandbox"
LABEL description="Sandbox runtime image with Python 3.14 on Alpine (minimal)"
LABEL version="1.0"

# 必要最小限のAlpineパッケージをインストール
# ca-certificates: HTTPS通信に必要（セキュリティ上必須）
RUN apk add --no-cache \
    ca-certificates \
    && rm -rf /var/cache/apk/*

# ヘルスチェック - Pythonが正常に動作しているか確認
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD python -c "import sys; sys.exit(0)"

# =============================================================================
# CI/CD用イメージ（フル機能）
# =============================================================================
# Used for CI/CD pipelines and local development.
# Contains Python, Go (1.26+), and testing tools.
# Size: ~300MB
#
# Base: golang:1.26-alpine official image includes Go 1.26 on Alpine Linux
# Python is installed from Alpine repository to match runtime image version
# =============================================================================
FROM golang:1.26-alpine@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f AS builder

LABEL maintainer="docker-sandbox"
LABEL description="Sandbox builder image with Python 3.14, Go 1.26, and testing tools"
LABEL version="1.0"

# Python 3.14をAlpineリポジトリからインストール
# runtimeターゲットのPythonバージョンと合わせるため、バージョン指定なし
RUN apk add --no-cache \
    python3 \
    py3-pip \
    ca-certificates \
    git \
    gcc \
    musl-dev \
    golangci-lint \
    && rm -rf /var/cache/apk/*

# Python3をpythonコマンドとして利用可能に
RUN ln -sf /usr/bin/python3 /usr/bin/python

# 作業ディレクトリを作成
WORKDIR /app

# sandbox-controllerのソースをコピー（CI用）
# 注意: このCOPYはdocker buildのコンテキストがルートディレクトリの場合に機能
ARG SOURCE_DIR=.
COPY ${SOURCE_DIR}/sandbox-controller /app/

# sandbox-controllerのGoモジュールをダウンロード
RUN if [ -f go.mod ]; then go mod download; fi

# ヘルスチェック - PythonとGoが正常に動作しているか確認
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD python -c "import sys; sys.exit(0)" && sh -c "go version"
