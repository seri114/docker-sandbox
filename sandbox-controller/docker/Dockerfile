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
# 共通ベース - Python 3.14 Alpine
# =============================================================================
FROM python:3.14-alpine@sha256:5a824eb82cc75361f98611f3cfc5091ea33f10a6ccea4d4ebdabbc523b9a1614 AS base

# ラベルを追加
LABEL maintainer="docker-sandbox"
LABEL version="1.0"

# 必要最小限のAlpineパッケージをインストール
# ca-certificates: HTTPS通信に必要（セキュリティ上必須）
RUN apk add --no-cache \
    ca-certificates \
    && rm -rf /var/cache/apk/*

# =============================================================================
# 本番用イメージ（軽量）
# =============================================================================
# Used for production Python code execution in sandboxed containers.
# Contains only Python and minimal dependencies.
# Size: ~50MB
# =============================================================================
FROM base AS runtime

LABEL description="Sandbox runtime image with Python 3.14 on Alpine (minimal)"

# ヘルスチェック - Pythonが正常に動作しているか確認
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD python -c "import sys; sys.exit(0)"

# =============================================================================
# CI/CD用イメージ（フル機能）
# =============================================================================
# Used for CI/CD pipelines and local development.
# Contains Python, Go (1.26+), and testing tools.
# Size: ~300MB
# =============================================================================
FROM base AS builder

LABEL description="Sandbox builder image with Python, Go 1.26+, and testing tools"

# Go 1.26と開発ツールをインストール
# golangci-lint: Goのリンター/フォーマッター
RUN apk add --no-cache \
    git \
    go~=1.26 \
    && rm -rf /var/cache/apk/*

# 作業ディレクトリを作成
WORKDIR /app

# ヘルスチェック - PythonとGoが正常に動作しているか確認
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD python -c "import sys; sys.exit(0)" && sh -c "go version"
