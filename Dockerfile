# ============================================================
# Godot MCP Server — Multi-stage Go build
#
# Docs are pre-built in CI and copied in via build context.
# No pandoc, no Python, no build tools at runtime.
# ============================================================

# Stage 1: Build Go binary
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/godot-mcp-server ./cmd/godot-mcp-server

# Stage 2: Runtime
FROM alpine:3.22

RUN apk add --no-cache ca-certificates curl

COPY --from=builder /out/godot-mcp-server /usr/local/bin/

# Pre-built docs (from CI artifact)
COPY docs/ /docs/

# Security: non-root user
RUN adduser -D appuser
USER appuser

ENV GOMEMLIMIT=256MiB

# Runtime defaults (can be overridden)
ENV FASTMCP_HOST=0.0.0.0
ENV FASTMCP_TRANSPORT=http

EXPOSE 8000

ENTRYPOINT ["godot-mcp-server"]
