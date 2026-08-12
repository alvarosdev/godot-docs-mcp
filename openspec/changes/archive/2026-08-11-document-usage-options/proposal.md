## Why

The README has two gaps. First, it documents only one way to use the MCP server — Docker HTTP hosting — while the server supports several: local agent integration via stdio, Podman (recommended for security + no daemon), native Go without containers, and the existing HTTP hosting. Second, the README's tool list and usage flows are stale: it lists 3 MCP tools but the server now exposes 5 (`search_documentation`, `get_class_overview` are missing), and the recommended system prompt uses the old tree→file flow instead of the new search→overview flow. This change makes the README accurate and covers every way to run and consume the server, so the documentation itself facilitates Godot docs access at the MCP level.

## What Changes

### Usage modes (Deployment section)

- Rewrite the README `Deployment` section to document every usage mode:
  - **Container stdio (agent-launched)**: `docker run` / `podman run` with `FASTMCP_TRANSPORT=stdio` — the user's MCP client launches the container directly. Includes ready-to-paste MCP client config.
  - **Container HTTP (self-hosted)**: `docker run` / `podman run` with port mapping, plus the existing `docker-compose.yml` referenced as the recommended way to self-host (no compose file changes — document what exists and what each env var does).
  - **Native (no container)**: `make run` / `go run` for development, `make build` for a static binary.
- Add a note recommending **Podman** over Docker (daemon-less, rootless, Docker-compatible).
- Document the **image update cadence**: images rebuild weekly on schedule (cron), but the workflow only pushes when the upstream Godot docs actually change (conditional build against `versions.lock`). Explain `docker pull` for updates.
- Explain each environment variable (`FASTMCP_TRANSPORT`, `FASTMCP_HOST`, `FASTMCP_PORT`, `FASTMCP_PATH`, `MCP_AUTH_TOKEN`, `DOCS_DIR`) in the context of each mode.

### Content accuracy (tools, structure, quick start)

- Add a table of contents (TOC) at the top for a ~200-line README
- Add a **Quick Start** at the top — "run it in 30 seconds" with a 5-line MCP config, so a Godot dev doesn't scroll to try it
- Update **Available Tools** to list all 5 tools (`get_available_versions`, `get_documentation_tree`, `get_documentation_file`, `search_documentation`, `get_class_overview`) with the new search/overview descriptions
- Update the **Recommended System Prompt** to the new flow: search → overview → file
- Add a short **"Why GFM?"** section explaining why docs are RST→GFM converted (RST files are 272KB with hundreds of cross-refs; GFM is LLM-friendly) — non-technical, just the rationale
- Add a brief **Repository layout** table (server Go, CI pipeline, docs) for contributors
- Add a short **FAQ** (3-5 entries: how to update docs, use your own MCP client, etc.)
- Keep the README non-technical about MCP internals — focus on "how to run it", not protocol details

### Constraints

- **No code changes** — this is documentation only. `skip_specs: true` (no behavioral change).
- **Release body is out of scope** — it is managed separately, not updated here.

## Capabilities

None — documentation-only change. `skip_specs: true` set in `.openspec.yaml`.

## Impact

- **Files modified**: `README.md` (Deployment, Tools, Recommended System Prompt, + new sections: TOC, Quick Start, Why GFM, Repository layout, FAQ)
- **Files created**: none
- **Code**: zero changes — Go server, Dockerfile, docker-compose.yml, Makefile all untouched
- **Behavior**: none — this documents existing functionality
- **Out of scope**: the GitHub release body is managed separately and is NOT updated in this change
