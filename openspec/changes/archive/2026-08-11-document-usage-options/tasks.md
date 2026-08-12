## 1. Container stdio (agent-launched)

- [x] 1.1 Add a sub-section documenting container stdio mode with a copy-paste MCP client config using `command: "podman"` (and a `docker` variant), `args: ["run", "--rm", "-i", "-e", "FASTMCP_TRANSPORT=stdio", "ghcr.io/alvarosdev/godot-docs-mcp:latest"]`
- [x] 1.2 Explain the three flags in plain terms: `-i` keeps stdin open, no `-t` (a TTY would corrupt the protocol), `--rm` cleans up on exit
- [x] 1.3 Note that auth (`MCP_AUTH_TOKEN`) does not apply in stdio mode — it is local

## 2. Container self-hosted (HTTP)

- [x] 2.1 Add a sub-section for self-hosting via `docker run` / `podman run` with port mapping, showing `FASTMCP_TRANSPORT=http` (the default) and the `MCP_AUTH_TOKEN` example
- [x] 2.2 Document `docker-compose.yml` as the recommended self-host path, referencing the existing file (no compose changes) and listing what each env var does

## 3. Native (no container)

- [x] 3.1 Add a sub-section for running without a container: `make run` (HTTP) and `make build` (static binary to `dist/`), plus `go run ./cmd/godot-mcp-server` for development
- [x] 3.2 Note the binary is static and needs no Go toolchain at runtime; docs must be available in `docs/` (or set `DOCS_DIR`)

## 4. Env var reference

- [x] 4.1 Add a single reference table documenting each env var: `FASTMCP_TRANSPORT`, `FASTMCP_HOST`, `FASTMCP_PORT`, `FASTMCP_PATH`, `MCP_AUTH_TOKEN`, `DOCS_DIR` — with defaults and purpose
- [x] 4.2 Cross-reference each usage sub-section to the table for its relevant vars only

## 5. Update cadence

- [x] 5.1 Add a section explaining the image update cadence: weekly scheduled check, conditional build only when upstream Godot docs change, `docker pull` for the latest image, manual trigger for forced rebuilds

## 6. Podman recommendation

- [x] 6.1 Add a short note recommending Podman for local use (daemon-less, rootless, Docker-compatible) without requiring it; keep Docker commands valid

## 7. README structure (TOC + Quick Start)

- [x] 7.1 Add a table of contents (TOC) near the top of the README
- [x] 7.2 Add a `Quick Start` section after the intro: "run it in 30 seconds" — `docker run -p 8000:8000 ghcr.io/alvarosdev/godot-docs-mcp:latest` + a 5-line MCP client config

## 8. Tool list + system prompt accuracy

- [x] 8.1 Update `Available Tools` to list all 5 tools: `get_available_versions`, `get_documentation_tree`, `get_documentation_file`, `search_documentation`, `get_class_overview` — with descriptions matching the actual tool schemas
- [x] 8.2 Update the `Recommended System Prompt` to the new flow: `search_documentation` → `get_class_overview` → `get_documentation_file` (with section)

## 9. Context sections

- [x] 9.1 Add a short `Why GFM?` section: RST→GFM rationale (RST files are 272KB with hundreds of cross-refs; GFM is LLM-friendly; RST fallback preserved)
- [x] 9.2 Add a `Repository layout` table (Go server code, CI pipeline, docs, scripts) for contributors
- [x] 9.3 Add a brief `FAQ` (3-5 entries: how to update docs, use your own MCP client, etc.)

## 10. Verification

- [x] 10.1 Verify every documented command against the real binary (stdio config, HTTP run, native run) before finalizing
- [x] 10.2 Confirm no code, Dockerfile, compose, or Makefile changes were made (docs only)
- [x] 10.3 Review the README reads naturally for a Godot dev — no MCP internals, no academic detail
- [x] 10.4 Confirm the tool list in the README matches `RegisterTools()` exactly (5 tools)
