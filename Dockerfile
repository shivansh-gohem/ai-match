# ═══════════════════════════════════════════════
# DevConnect — Multi-Stage Docker Build
# Optimized for small image size (~20MB)
# ═══════════════════════════════════════════════

# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency files first (for Docker layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary (static, no CGO)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# Stage 2: Minimal runtime image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /server .

# Copy static frontend files
COPY --from=builder /app/web ./web

# Copy .env (optional, can be overridden by K8s configmaps/secrets)
COPY --from=builder /app/.env ./.env

EXPOSE 8080

# Run the server
CMD ["./server"]
