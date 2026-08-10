## Context

The current server is ~872 lines of Python using FastMCP. It exposes 3 MCP tools and 1 MCP resource over stdio/HTTP/SSE transports, serving pre-built Godot documentation (243MB of Markdown files under `docs/{version}/`). Auth is a custom `StaticTokenVerifier` class (39 lines). The docs pipeline is Python calling pandoc as a subprocess; it only runs in CI, never at runtime.

Go 1.26 is available. The official `modelcontextprotocol/go-sdk` v1.2.0 provides all needed MCP primitives with a built-in `auth` package. See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**
- Identical MCP API surface: same tools, same resource URIs, same response formats
- Single statically-compiled Go binary (~10-15MB, CGO_ENABLED=0)
- Multi-stage Docker image ≤ 25MB (runtime only, no pandoc, no build tools)
- Shell-based docs pipeline for CI (no Python in any path)
- Auth via SDK's `auth.RequireBearerToken` (less code, standard `http.Handler` middleware)
- Same env var configuration surface (`MCP_AUTH_TOKEN`, `FASTMCP_TRANSPORT`, `FASTMCP_HOST`, `FASTMCP_PORT`, `FASTMCP_PATH`)
- Native multi-arch support via `GOOS`/`GOARCH` (no QEMU needed for builds)

**Non-Goals:**
- Change documentation format or structure
- Add new MCP tools/resources beyond the existing 3 tools + 1 resource
- Change the CI conditional-build logic (SHA comparison, artifact passing)
- Add a web UI, metrics endpoint, or observability beyond `slog`
- Support the legacy SSE transport
- Graceful shutdown beyond standard `signal.NotifyContext` (not needed today)
- File caching beyond an in-memory map (no LRU, no TTL — 243MB fits easily)

## Decisions

### 1. Package structure: flat `internal/` with `cmd/` entry points

```
cmd/godot-mcp-server/main.go       ← single runtime binary
scripts/generate-docs.sh            ← CI docs pipeline (shell)
internal/
├── server/server.go                ← mcp.NewServer + config
├── tools/navigation.go             ← 3 MCP tools
├── resources/docs.go               ← 1 resource template + handler
├── docs/versions.go                ← versions.json parsing + fallback
└── docs/validate.go                ← path validation + traversal protection
```

**Why**: Standard Go layout. `internal/` prevents external import. One package per concern, no deep nesting. The converter is shell because it only runs in CI — a Go binary would add compilation overhead for no runtime benefit. The CI pipeline already has `pandoc`, `curl`, `jq`, and `tree` installed.

**Alternatives considered**:
- `pkg/` instead of `internal/` — rejected; no external consumers
- Single `main.go` with everything — rejected; harder to test
- Go binary for docs converter — rejected; shell is simpler and CI already has the tools

### 2. Resource template: `file:///{+file_path}` with reserved expansion

The SDK uses RFC 6570 URI templates via `github.com/yosida95/uritemplate/v3`. The `+` modifier (reserved expansion) allows `/` in variable values, enabling multi-segment paths like `3.6/classes/class_camera2d.md`.

**Critical**: MCP `file:///` URIs require **three slashes** (scheme + empty authority + absolute path). Two slashes would parse as `host=3.6`, breaking resource matching.

```go
server.AddResourceTemplate(&mcp.ResourceTemplate{
    URITemplate: "file:///{+file_path}",   // ← three slashes, not two
    Name:        "Godot documentation file",
    Description: "Retrieve a Godot documentation file. Optionally prefix with version.",
    MIMEType:    "text/markdown",
}, docResourceHandler)
```

**Confirmed by SDK test** (`resource_test.go:116-132`): `file:///{+path}` matches `file:///path/to/file` — 3 segments.

**Alternatives considered**:
- Register each file as a static resource — rejected for 5000+ files
- Use `file://{version}/{path}` with two variables — worse UX, two segments to match
- Split on first `/` manually — fragile, bypasses URI template matching

### 3. File loading: in-memory map at startup (immutable after load)

A `map[string]string` keyed by versioned relative path (`"4.7/classes/class_camera2d.md"`), populated by walking `docs/` at startup. This is ~243MB of RSS — acceptable for a server with 512MB+ RAM.

**Concurrency safety**: The map is write-once-read-many. Go guarantees safe concurrent reads from a map that has no concurrent writes. The `Load()` call happens synchronously before the server starts accepting connections (`sync.WaitGroup` or sequential startup in `main`). No mutex needed. This contract is enforced: there is no hot-reload, no background refresh, no mutation path after `Load()` returns.

```go
type DocStore struct {
    docs map[string]string // "4.7/classes/foo.md" → content, immutable after Load()
}
func (s *DocStore) Load(root string) error { /* filepath.Walk, single-threaded */ }
func (s *DocStore) Get(path string) (string, bool) { /* lock-free map lookup */ }
```

**Path validation in Get()**: Before lookup, paths pass through `filepath.IsLocal()` rejection. Paths with `..`, absolute prefixes (`/`), or Windows volume letters are rejected before touching the map or filesystem. This is defense in depth — the map is pre-populated so an invalid key simply won't match, but explicit rejection produces clearer errors and prevents logic bugs from turning into security bugs later.

**Why**: Instant reads, no I/O on every MCP request, trivial code, lock-free concurrency. The current Python uses `@lru_cache(maxsize=128)` which evicts under load — the Go approach is simpler and faster.

**Alternatives considered**:
- Read from disk on every request — rejected; slower, more I/O
- LRU cache with TTL — rejected; more complex, no benefit when all docs fit in memory
- `sync.RWMutex` wrapping all access — rejected; unnecessary because the map is immutable post-load
- Embed docs via `embed.FS` at compile time — rejected; docs are pre-built in CI and change independently of the binary

### 4. Docker: multi-stage, scratch-like runtime

```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/server ./cmd/godot-mcp-server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/server /usr/local/bin/
COPY docs/ /docs/
USER 1000:1000
EXPOSE 8000
ENTRYPOINT ["server"]
```

**Why**: `CGO_ENABLED=0` produces a static binary. Alpine is 3MB, the stripped binary is ~10-15MB. Total image ~15-25MB. No pandoc, no interpreter, no package manager at runtime. `ca-certificates` is included for TLS (GitHub API calls in CI, not needed at runtime but harmless).

**Why**: `CGO_ENABLED=0` produces a static binary. Alpine is 3MB, the stripped binary is ~10-15MB. Total image ~15-25MB. No pandoc, no interpreter, no package manager at runtime.

**Alternatives considered**:
- `scratch` base — rejected; no CA certificates, no `/etc/passwd` for USER
- `distroless` — rejected; overkill for a single binary
- Keep `golang:alpine` as runtime — rejected; adds ~100MB unnecessarily

### 5. Auth: SDK's `auth.RequireBearerToken` with startup-captured token

The token is read from the environment **once at startup**, not on every request. This prevents:
- Unnecessary `os.Getenv` syscalls in the hot path
- Time-of-check-time-of-use (TOCTOU) issues if the environment variable changes at runtime
- Predictable constant-time comparison behavior (the token value is fixed for the process lifetime)

```go
// Read once at startup in main.go
token := os.Getenv("MCP_AUTH_TOKEN")

if token != "" {
    verifier := func(ctx context.Context, tokenStr string, req *http.Request) (*auth.TokenInfo, error) {
        if subtle.ConstantTimeCompare([]byte(tokenStr), []byte(token)) != 1 {
            return nil, auth.ErrInvalidToken
        }
        return &auth.TokenInfo{}, nil
    }
    authMw := auth.RequireBearerToken(verifier, &auth.RequireBearerTokenOptions{
        Scopes: []string{},
    })
    // wrap handler ...
}
```

**Why**: The SDK's auth package handles header parsing, WWW-Authenticate challenges, 401/403 responses, and context propagation. Our current `StaticTokenVerifier` is 39 lines of Python; this is a 6-line closure. Middleware integrates with `StreamableHTTPHandler` as a standard `func(http.Handler) http.Handler`.

### 6. CI: shell pipeline replaces Python converter

The `prepare-docs` job currently runs `uv sync && uv run python docs_converter/godot_docs_converter.py`. This becomes:

```bash
set -euo pipefail

# Cap parallelism at 8 workers (same as the Python converter's min(cpu_count(), 8))
MAX_WORKERS=8

for VERSION in $VERSIONS; do
  # 1. Download ZIP from GitHub
  #    -f: fail on HTTP errors (4xx/5xx)
  #    -s: silent progress, -S: show errors
  curl -fsSL "https://github.com/godotengine/godot-docs/archive/refs/heads/$VERSION.zip" \
      -o /tmp/godot-docs.zip
  unzip -q /tmp/godot-docs.zip -d /tmp/godot-docs-temp
  mv "/tmp/godot-docs-temp/godot-docs-$VERSION" "docs/$VERSION"

  # 2. Convert RST → Markdown (parallel, capped at MAX_WORKERS)
  find "docs/$VERSION" -name '*.rst' -print0 \
      | xargs -0 -P "$MAX_WORKERS" -I {} sh -c \
          'pandoc "$1" -o "${1%.rst}.md" && rm "$1"' _ {}

  # 3. Cleanup non-.md files and empty dirs
  find "docs/$VERSION" -type f ! -name '*.md' ! -name 'docs_tree.txt' -delete
  find "docs/$VERSION" -type d -empty -delete

  # 4. Generate docs tree
  tree "docs/$VERSION" > "docs/$VERSION/docs_tree.txt"
done

# 5. Write versions.json
# ... jq logic
```

**Why**: `set -euo pipefail` ensures the script fails on any error (unset variable, failed command, failed pipe). The Python code was 148 lines of orchestrating CLI tools (`pandoc`, `tree`, `curl`). Shell is the native orchestrator for CLI pipelines. CI already has all dependencies installed. Workers are capped at 8 (matching the Python converter's `min(cpu_count(), 8)`) to prevent OOM on high-core CI runners.

## Risks / Trade-offs

- **[Risk] Go SDK API instability**: The SDK is pre-2.0 and may have breaking changes. → **Mitigation**: Pin to a specific version in `go.mod`. The API surface we use (tools, resources, streamable HTTP, auth) is mature and covered by conformance tests.
- **[Risk] In-memory file store OOM**: 243MB of docs plus Go runtime overhead. → **Mitigation**: Acceptable for modern containers (512MB+). Can fall back to disk reads if needed by adding a `sync.Map` cache layer — but the current Python LRU cache already implies this isn't a concern.
- **[Risk] SDK goroutine leak on abandoned sessions**: The SDK documents known goroutine leaks (issue #499) when streamable HTTP clients disappear without cleanup. → **Mitigation**: Configure `KeepAlive` and `KeepAliveFailureThreshold` in `ServerOptions`. Dead sessions are detected via ping failures and closed, releasing goroutines.
- **[Risk] Slow-client resource exhaustion**: Without HTTP timeouts, a client that opens a connection and sends no data holds a goroutine indefinitely. → **Mitigation**: Set `ReadTimeout` (10s), `WriteTimeout` (30s), and `IdleTimeout` (120s) on the `http.Server`.
- **[Risk] Timing side-channel in token comparison**: Using `!=` for string comparison of auth tokens leaks timing information about the token prefix. → **Mitigation**: Use `subtle.ConstantTimeCompare` for token comparison.
- **[Risk] Shell pipeline portability**: `xargs -P`, `pandoc`, `tree` behavior may differ across distros. → **Mitigation**: All run in CI on `ubuntu-latest` (same as current). Worker count is capped at 8 (matching Python's limit) to prevent OOM on high-core runners. `set -euo pipefail` ensures errors stop the pipeline.
- **[Trade-off] No Python → no pypandoc**: The converter used `pypandoc` which bundles pandoc bindings. The shell script calls pandoc directly. → Same pandoc binary, same output, slightly different error handling. Acceptable since this only runs in CI.
