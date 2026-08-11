## Why

Large Godot class files like `Control` (272KB) and `Node` (214KB) are truncated by the MCP protocol, making methods and properties inaccessible. Searching for common class names like "control" returns noise from 237+ files where the word appears incidentally — `class_control.rst` doesn't even make the top results. The LLM has to fall back to GitHub or manual grep to read documentation, defeating the purpose of the MCP server.

## What Changes

- Add optional `section` parameter to `get_documentation_file` — parses RST section headers on-the-fly and returns only the requested section (Methods, Properties, Signals, etc.)
- Add filename match boost to `search_documentation` — query tokens matching a file's class name get a 10× score multiplier, ensuring `class_control.rst` ranks #1 for "control"
- Add optional `category` parameter to `search_documentation` — filter results by path prefix (`classes/`, `tutorials/`, or omit for all)
- Add file size warning to `get_documentation_file` — when the file exceeds 50KB, the response includes a hint suggesting the `section` parameter
- **No new dependencies** — RST section parsing uses only the standard library

## Capabilities

### Modified Capabilities

- `version-aware-tools`: `get_documentation_file` gains optional `section` parameter and size warning. `search_documentation` gains optional `category` parameter.
- `documentation-search`: Search scoring includes filename match boost (10×). Search accepts optional category filter by path prefix.

## Impact

- **Files modified**: `internal/tools/navigation.go` (+section parsing, +category filter, +size warning, ~60 lines), `internal/search/scan.go` (+filename boost, +category filter, ~15 lines)
- **Files created**: `internal/tools/navigation_test.go` (+section tests, +category filter tests, ~30 lines)
- **Binary size**: +0 MB (no new dependencies)
- **Runtime behavior**: `get_documentation_file` returns full file when `section` is omitted (backward compatible). Search results re-ranked for class-name queries.
