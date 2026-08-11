## Why

The server currently requires the LLM to guess which documentation file contains the answer to a user's question. The workflow is: discover versions → request tree → guess filename → read file. If the user asks "how do collisions work in CharacterBody2D?" but the LLM doesn't know the exact class name, it has to browse the tree and try files until it finds the right one — wasting round-trips and tokens. Adding a `search_documentation` tool that scans all docs in-memory with token-based scoring lets the LLM find relevant files in a single call, without needing to know the exact path or class name.

## What Changes

- Add `Keys() []string` method to `DocStoreReader` interface and `DocStore` implementation
- Create `internal/search/` package with a zero-dependency substring search engine using token coverage, heading boost, section boost, and proximity scoring
- Add `search_documentation` MCP tool accepting `query`, optional `version`, and optional `limit` parameters
- Register the search tool alongside the existing navigation tools
- Wire the search index build at server startup (after doc store load, before accepting connections)
- Preserve original `.rst` files as fallback when pandoc conversion fails — successful conversions delete the `.rst` via `rm "$1"` inside the converter, so the cleanup step only needs `! -name '*.rst'` to protect the ~6 files pandoc couldn't handle
- **No new external dependencies** — the search engine uses only the Go standard library
- **No spec-level behavior changes to existing tools** — `get_documentation_tree` and `get_documentation_file` remain unchanged

## Capabilities

### New Capabilities

- `documentation-search`: Zero-dependency full-text search over the in-memory documentation store using token-based scoring with heading/section/proximity boosts. Exposed as a `search_documentation` MCP tool returning ranked results with file paths, versions, scores, and text snippets.

### Modified Capabilities

- `version-aware-tools`: Adds a new `search_documentation` tool to the existing MCP tool set. No existing tool behavior changes.

## Impact

- **Dependencies added**: None (standard library only)
- **Files created**: `internal/search/scan.go` (~80 lines), `internal/search/scan_test.go` (~60 lines)
- **Files modified**: `internal/docs/store.go` (+`Keys()` method, ~5 lines), `internal/tools/navigation.go` (+1 tool registration, ~15 lines), `cmd/godot-mcp-server/main.go` (+3 lines for search setup)
- **Files regenerated**: `internal/docs/store_mock.go` (mockery re-run for new `Keys()` method)
- **Binary size**: +0 MB (no new dependencies)
- **Startup time**: +0 ms (scan engine has no index to build — searches scan the existing in-memory map)
- **RAM**: +0 MB (no separate index structure)
- **Search latency**: ~20-50ms for a full scan of ~5000 documents / 29MB of text
