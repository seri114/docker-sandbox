# Build stage
FROM golang:1.27-alpine@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc AS builder

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
