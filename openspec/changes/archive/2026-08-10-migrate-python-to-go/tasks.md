## 1. Project scaffolding

- [x] 1.1 Initialize Go module: `go mod init github.com/alvarosdev/godot-mcp-docs`
- [x] 1.2 Add `github.com/modelcontextprotocol/go-sdk` dependency
- [x] 1.3 Create directory structure: `cmd/godot-mcp-server/`, `internal/server/`, `internal/tools/`, `internal/resources/`, `internal/docs/`
- [x] 1.4 Create `scripts/generate-docs.sh` (empty skeleton)

## 2. Docs utilities

- [x] 2.1 Implement `internal/docs/versions.go`: parse `versions.json` into a struct, fallback to scanning `docs/` for `major.minor` directories, resolve latest version. Read from disk on each call (not cached in-memory — version metadata is tiny)
- [x] 2.2 Implement `internal/docs/validate.go`: validate paths with `filepath.IsLocal()` before resolution, reject `..`, absolute paths, URL-encoded traversal. Use `filepath.Join` and `filepath.Clean` for safe path construction
- [x] 2.3 Implement `internal/docs/store.go`: load all docs into a `map[string]string` at startup, expose lock-free `Get(path) (string, bool)`. The map is populated before the server accepts connections and never mutated after
- [x] 2.4 Write unit tests for version resolution (with and without `versions.json`, empty dir, invalid JSON)
- [x] 2.5 Write unit tests for path validation (valid paths, `../` traversal, absolute paths, URL-encoded `%2e%2e`, paths with null bytes, paths over 4096 chars)
- [x] 2.6 Write unit tests for doc store (load, get, get missing key, concurrent reads from 100 goroutines)

## 3. MCP server core

- [x] 3.1 Implement `internal/server/server.go`: create `NewServer()` that initializes `mcp.Server` with `Implementation`, `ServerOptions` (instructions, logger, `KeepAlive` with `KeepAliveFailureThreshold`), registers tools and resources
- [x] 3.2 Implement stdio transport entry point: read `FASTMCP_TRANSPORT=stdio` at startup, call `server.Run(ctx, &mcp.StdioTransport{})`
- [x] 3.3 Implement streamable HTTP entry point: create `mcp.NewStreamableHTTPHandler`, optional auth middleware wrapping, manual `/health` handler
- [x] 3.4 Configure `http.Server` with `ReadTimeout=10s`, `WriteTimeout=30s`, `IdleTimeout=120s` to prevent slow-client resource exhaustion
- [x] 3.5 Configure `ServerOptions.KeepAlive` and `KeepAliveFailureThreshold` to detect and clean up abandoned sessions (mitigates SDK goroutine leak #499)
- [x] 3.6 Implement graceful shutdown via `signal.NotifyContext` (SIGINT, SIGTERM) with `srv.Shutdown(ctx)` for the HTTP server
- [x] 3.7 Read all configuration from environment variables once at startup (no `os.Getenv` in hot paths)

## 4. MCP tools

- [x] 4.1 Implement `get_available_versions` tool: read version metadata from docs utilities, return JSON with `versions`, `latest`, `built`
- [x] 4.2 Implement `get_documentation_tree` tool: accept optional `version` parameter, default to latest, return `docs/{version}/docs_tree.txt` content
- [x] 4.3 Implement `get_documentation_file` tool: accept `file_path` and optional `version` parameters, validate path with `filepath.IsLocal()`, return file content from doc store. Error messages SHALL use relative paths only — never expose absolute filesystem paths
- [x] 4.4 Write unit tests for each tool (mock doc store or use `testdata/` directory)
- [x] 4.5 Verify all error messages use relative paths (audit `Errorf` calls for `%s` with path variables)

## 5. MCP resources

- [x] 5.1 Implement resource handler: extract `file_path` from URI template match, validate with `filepath.IsLocal()` before any filesystem operation, parse optional version prefix (`3.6/classes/foo.md` → version=`3.6`, path=`classes/foo.md`), return content from doc store
- [x] 5.2 Register resource template `file:///{+file_path}` with `AddResourceTemplate` (three slashes, not two: `file:///` is the correct MCP file URI form)
- [x] 5.3 Write unit tests for resource handler (with version prefix, without, not found, traversal attempt via `../`, traversal attempt via `%2e%2e`)

## 6. Authentication

- [x] 6.1 Implement bearer token verifier: capture `MCP_AUTH_TOKEN` from env var once at startup. Use `subtle.ConstantTimeCompare` for token comparison. Empty/unset token means auth disabled
- [x] 6.2 Wire `auth.RequireBearerToken` middleware into streamable HTTP handler only when auth is configured
- [x] 6.3 Write unit test for auth middleware (valid token, invalid token, missing header, auth disabled)

## 7. Docs pipeline (shell script)

- [x] 7.1 Implement `scripts/generate-docs.sh`: `set -euo pipefail`, loop over versions, download with `curl -fsSL` (fail on HTTP errors), extract, convert RST→MD with pandoc (parallel via `xargs -P 8`, capped at 8 workers to prevent OOM on high-core CI runners), cleanup non-.md files, generate tree
- [x] 7.2 Implement `versions.json` generation at end of pipeline
- [x] 7.3 Test script locally with a single Godot version to verify output matches current `docs/` structure
- [x] 7.4 Verify script fails correctly on download error (test with invalid version string), pandoc error (test with corrupt .rst file)

## 8. Docker

- [x] 8.1 Write multi-stage `Dockerfile`: `golang:1.26-alpine` builder → `alpine:3.22` runtime with `ca-certificates`, `CGO_ENABLED=0`, stripped binary (`-ldflags="-s -w"`), non-root user (`USER 1000:1000`), copy `docs/`
- [x] 8.2 Update `docker-compose.yml`: same env vars, same port mapping, updated healthcheck command, remove Python-specific volumes/comments
- [x] 8.3 Update `.dockerignore`: exclude `srcs/`, `docs_converter/`, `__pycache__/`, `.venv/`, `*.py`, `openspec/`, `.claude/`, `.git/`

## 9. CI/CD

- [x] 9.1 Update `.github/workflows/build-publish.yml`: replace `uv sync && uv run python` with `scripts/generate-docs.sh`, replace Docker build with multi-stage Go build
- [x] 9.2 Verify multi-arch build strategy still works (Go cross-compiles natively — Docker `--platform` handles it)
- [x] 9.3 Update `versions.lock` update logic if needed (should be unchanged)

## 10. Documentation

- [x] 10.1 Update `README.md`: replace Python-specific instructions with Go equivalents, update Docker Compose example, update system requirements
- [x] 10.2 Update `.env.example`: keep same keys, add comment about Go server

## 11. Cleanup

- [x] 11.1 Delete Python source files: `main.py`, `srcs/`, `docs_converter/`, `tests/`
- [x] 11.2 Delete Python config files: `pyproject.toml`, `uv.lock`, `.env`
- [x] 11.3 Remove `.venv/` and `__pycache__/` directories
- [x] 11.4 Run `go mod tidy` to finalize dependencies
- [x] 11.5 Verify the project builds cleanly: `go build ./cmd/godot-mcp-server`
- [x] 11.6 Verify tests pass: `go test ./internal/...`
- [x] 11.7 Verify the server starts with stdio transport (smoke test)
