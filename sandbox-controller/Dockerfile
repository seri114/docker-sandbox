# Build stage
FROM golang:1.27-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS builder

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
FROM alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf

# Copy the binary from builder
COPY --from=builder /build/controller /controller

# Set entrypoint
ENTRYPOINT ["/controller"]
