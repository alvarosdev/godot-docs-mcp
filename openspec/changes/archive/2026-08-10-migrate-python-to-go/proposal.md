## Why

Eliminate the Python runtime entirely — interpreter, venv, uv, pypandoc, and FastMCP — in favor of a single statically-compiled Go binary. This slashes the Docker image from ~150MB to ~15-25MB, cuts server startup from seconds to milliseconds, reduces idle memory from ~80-150MB to ~15-30MB, and dramatically simplifies the CI pipeline. The official `modelcontextprotocol/go-sdk` (v1.2.0) provides all the MCP primitives we currently use (tools, resources, streamable HTTP, stdio, bearer-token auth), and Go 1.26 is available today.

## What Changes

- **BREAKING**: Reimplement the MCP server in Go using `github.com/modelcontextprotocol/go-sdk`
- **BREAKING**: Replace the Python `docs_converter` pipeline with a shell script (CI only)
- **BREAKING**: Replace `Dockerfile` (Python 3.13 alpine + uv) with a multi-stage Go build producing a ~15-25MB image
- **BREAKING**: Remove `pyproject.toml`, `uv.lock`, `.venv`, `__pycache__`, `srcs/`, `docs_converter/`, `main.py`, `.env`
- Add `go.mod`, `cmd/godot-mcp-server/`, `internal/` with clean package structure
- Add `scripts/generate-docs.sh` for CI docs pipeline (download → pandoc → cleanup → tree → versions.json)
- Update `.github/workflows/build-publish.yml` to build Go instead of installing uv/pip
- Auth becomes a one-liner using the SDK's built-in `auth.RequireBearerToken` instead of custom `StaticTokenVerifier`
- Documentation files (`docs/`) remain identical — no format change, same structure

## Capabilities

### New Capabilities

- `mcp-server`: MCP server exposing Godot documentation via tools (`get_available_versions`, `get_documentation_tree`, `get_documentation_file`) and a resource (`file:///{+file_path}`) over stdio and streamable HTTP transports, with optional bearer-token authentication, HTTP timeouts, and session keepalive to prevent resource exhaustion
- `docs-pipeline`: CI pipeline that downloads Godot docs from GitHub, converts RST to Markdown via pandoc, generates metadata (`versions.json`, `docs_tree.txt`), and produces a clean docs artifact

### Modified Capabilities

None — existing specs are empty/non-existent. This is a rewrite of the entire runtime with identical MCP behavior.

## Impact

- **Code removed**: ~872 lines of Python (`main.py`, `srcs/`, `docs_converter/`)
- **Code added**: ~400-500 lines of Go + ~50 lines of shell
- **Docker image**: python:3.13-alpine (150MB) → golang:1.26-alpine → alpine:3.22 (15-25MB final)
- **Dependencies**: pypandoc, FastMCP, uv → only MCP Go SDK (`github.com/modelcontextprotocol/go-sdk`)
- **CI**: uv sync + pandoc install → go build + pandoc (already in CI)
- **Runtime**: No Python, no interpreter, no venv — single Go binary
- **Users**: No migration needed. Same MCP tools, same URIs, same env vars, same Docker Compose interface
