# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Allow Go toolchain auto-download for compatibility
ENV GOTOOLCHAIN=auto

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary (use goversion to auto-detect needed version)
ENV GOTOOLCHAIN=go1.25.10
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /build/controller .

# Final stage - TEMPORARY alpine for testing
FROM alpine:latest

# Copy the binary from builder
COPY --from=builder /build/controller /controller

# Set entrypoint
ENTRYPOINT ["/controller"]
