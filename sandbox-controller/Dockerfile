# Build stage
FROM golang:1.26-alpine@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates gcc musl-dev

# Allow Go toolchain auto-download for compatibility
ENV GOTOOLCHAIN=auto

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /build/controller .

# Final stage - TEMPORARY alpine for testing
FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

# Copy the binary from builder
COPY --from=builder /build/controller /controller

# Set entrypoint
ENTRYPOINT ["/controller"]
