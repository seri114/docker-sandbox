# =============================================================================
# Sandbox Controller Custom Docker Image
# =============================================================================
# This Dockerfile defines a custom image for sandbox execution.
#
# Purpose:
#   - Production: Execute Python code in isolated containers
#   - Testing: Run integration and E2E tests with Alpine commands
#
# Base: Python 3.14 on Alpine Linux (includes Python + Alpine tools)
#
# Renovate: This format is automatically updated by Renovate bot
#
# Usage:
#   docker build -t sandbox:latest -f sandbox-controller/docker/Dockerfile .
# =============================================================================

# =============================================================================
# サンドボックスイメージ
# =============================================================================
# Used for both Python code execution and Alpine command testing.
# Base image (python:3.14-alpine) already includes both Python and Alpine tools.
#
# Note: Renovate automatically updates the tag and digest
# =============================================================================

FROM python:3.14-alpine@sha256:5a824eb82cc75361f98611f3cfc5091ea33f10a6ccea4d4ebdabbc523b9a1614

# ラベルを追加
LABEL maintainer="docker-sandbox"
LABEL description="Sandbox execution environment with Python 3.14 on Alpine"
LABEL version="1.0"

# 必要最小限のAlpineパッケージをインストール
# ca-certificates: HTTPS通信に必要（セキュリティ上必須）
RUN apk add --no-cache \
    ca-certificates \
    git \
    && rm -rf /var/cache/apk/*

# 作業ディレクトリを作成
WORKDIR /app

# ヘルスチェック - PythonとAlpineが正常に動作しているか確認
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD python -c "import sys; sys.exit(0)" && sh -c "exit 0"
