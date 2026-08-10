## Why

The project has partial test coverage using only the `testing` standard library — no assertions, no mocks, no test doubles. This forces tests that need the `DocStore` to use real files on disk (in `testdata/`), coupling tool and resource tests to filesystem state. Adding testify for assertions and mockery for generated mocks via a single `DocStoreReader` interface enables fast, deterministic unit tests that never touch disk or external services.

Additionally, the migration from Python to Go left some artifacts behind: an old Python `deploy/Dockerfile`, references to the original fork's author in the license, and outdated repo/image URLs pointing to the old `godot-mcp-docs-http` name. This change also completes that cleanup — rebranding the project as a standalone Go project rather than a Python fork.

## What Changes

### Test infrastructure

- Add `github.com/stretchr/testify` dependency for `assert`/`require`/`mock` packages
- Add `github.com/vektra/mockery/v3` as a `tool` in `go.mod` (not a runtime dependency)
- Extract a `DocStoreReader` interface in `internal/docs/store.go` with 2 methods: `Get`, `Size`
- Change `tools/navigation.go` handlers and `resources/docs.go` handler to accept `DocStoreReader` instead of `*DocStore`
- Create `.mockery.yml` configuration at project root with testify template, inpackage mode
- Generate `internal/docs/store_mock.go` via `go tool mockery`
- Rewrite existing tests in all packages using testify assertions + `MockDocStoreReader`
- Add tests for `internal/server/server.go` (`LoadConfig`, `New`) using testify + `t.Setenv`
- Remove all uses of manual `if got != want { t.Errorf(...) }` in favor of `assert.Equal` / `require.NoError`
- Delete `testdata/` directories that are no longer needed for tools/resources tests (docs store tests keep theirs)

### Project cleanup

- Delete `deploy/Dockerfile` — obsolete Python-based Dockerfile, replaced by multi-stage Go Dockerfile at repo root
- Update `LICENSE` — replace `Copyright (c) 2025 Nihilantropy` with `Copyright (c) 2026 Álvaro G.`
- Update all references in `README.md`: `godot-mcp-docs-http` → `godot-docs-mcp` (repo name, image URL, clone URL)
- Add attribution line in README acknowledging Nihilantropy's original Python MCP server as the idea base
- Update git remote: `git@github.com:alvarosdev/godot-mcp-docs-http.git` → `git@github.com:alvarosdev/godot-docs-mcp.git`
- Remove fork relationship — this is 100% new Go code, not a fork of the original Python project
- **No spec-level behavior changes** — the MCP server serves identical tools and resources

## Capabilities

None — pure tooling, quality, and cleanup change. `skip_specs: true` set in `.openspec.yaml`.

## Impact

- **Dependencies added**: `github.com/stretchr/testify` (assert + mock), `github.com/vektra/mockery/v3` (tool)
- **Files modified**: `internal/docs/store.go` (+6 lines for interface), `internal/tools/navigation.go` (signature change), `internal/resources/docs.go` (signature change), `LICENSE`, `README.md`, `cmd/godot-mcp-server/main.go` (no change needed — `*DocStore` satisfies interface)
- **Files created**: `.mockery.yml`, `internal/docs/store_mock.go` (generated), `*_test.go` rewrites, `internal/server/server_test.go`
- **Files deleted**: `deploy/Dockerfile`, `internal/tools/testdata/`, `internal/resources/testdata/`
- **Git remote**: changed from `godot-mcp-docs-http` to `godot-docs-mcp`
- **Runtime behavior**: zero changes — `DocStore` implements `DocStoreReader` implicitly
- **CI**: `go tool mockery && git diff --exit-code` to enforce mock freshness
