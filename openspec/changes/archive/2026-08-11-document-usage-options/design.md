## Context

The README currently documents one usage path (Docker HTTP hosting). The server supports multiple modes — stdio agent-launch, HTTP self-host, and native Go — but none are documented beyond the single Docker example. See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**
- Document all ways to run the MCP server, each with a copy-paste example
- Recommend Podman over Docker (daemon-less, rootless) without being prescriptive
- Explain the image update cadence and how users stay current
- Explain every env var in context, so users know what each does
- Keep it friendly for Godot devs — practical, not academic

**Non-Goals:**
- Modify any code, Dockerfile, docker-compose.yml, or Makefile
- Update the GitHub release body (managed separately)
- Explain MCP protocol internals
- Cover every imaginable MCP client — document the standard config shape, clients adapt

## Decisions

### 1. Structure: a "Usage" section with sub-sections per mode

The README's `Deployment` section becomes a `Usage` section with clear sub-sections, each self-contained (no "see above" cross-dependencies):

1. **Container, agent-launched (stdio)** — for users who want the agent to run the container itself
2. **Container, self-hosted (HTTP)** — for users running their own server (Docker/Podman + compose)
3. **Native (no container)** — for development and single-binary deployment

**Why**: Each mode has different env vars and different MCP config. Mixing them in one example confuses users. Self-contained sections let a user jump straight to their case.

### 2. Podman recommended, Docker shown

Both commands are shown. A short note recommends Podman for local use (no daemon, rootless, works with the same commands via aliasing). The `docker` commands remain valid for users on Docker.

### 3. Stdio config is the headline addition

The key new content is the stdio mode — a ready-to-paste MCP client config where the agent launches the container:

```json
{
  "mcpServers": {
    "godot-docs": {
      "command": "podman",
      "args": ["run", "--rm", "-i", "-e", "FASTMCP_TRANSPORT=stdio",
               "ghcr.io/alvarosdev/godot-docs-mcp:latest"]
    }
  }
}
```

**Why**: This is the "it just works" experience — no port mapping, no daemon, the agent manages the container lifecycle. The three flags matter: `-i` keeps stdin open, no `-t` (a TTY would corrupt the protocol), `--rm` cleans up on exit. Documented in plain terms.

### 4. Env vars explained once, in a reference table

A single reference table lists each var, its default, and what it does. Each usage sub-section references only the vars relevant to that mode. This avoids repeating the same definitions in every example.

| Var | Default | Purpose |
|-----|---------|---------|
| `FASTMCP_TRANSPORT` | `http` | `http` or `stdio` — transport mode |
| `FASTMCP_HOST` | `127.0.0.1` | Bind address (HTTP only) |
| `FASTMCP_PORT` | `8000` | Port (HTTP only) |
| `FASTMCP_PATH` | `/mcp` | HTTP path (HTTP only) |
| `MCP_AUTH_TOKEN` | empty | Bearer token for HTTP auth; ignored in stdio |
| `DOCS_DIR` | `docs` | Where the docs live in the container |

### 5. Update cadence: weekly check, conditional build

Document the CI behavior plainly:
- A scheduled workflow runs **weekly** (Sundays) and checks whether upstream Godot docs changed
- The image is **only rebuilt when docs actually changed** (compared via `versions.lock`) — so there's no fixed image schedule; it's "on doc change, checked weekly"
- `docker pull ghcr.io/alvarosdev/godot-docs-mcp:latest` always gets the freshest build
- Manual trigger exists for forcing a rebuild

### 6. README accuracy: all 5 tools + new flow

The README's `Available Tools` lists 3 tools but the server exposes 5. List all five, with the search/overview descriptions. The `Recommended System Prompt` is updated from the old tree→file flow to the new search→overview→file flow:

```
Old flow:  get_available_versions → get_documentation_tree → guess file → get_documentation_file
New flow:  get_available_versions → search_documentation → get_class_overview → get_documentation_file(section)
```

### 7. README structure: TOC + Quick Start up top

Add a table of contents (the README is ~200 lines) and a `Quick Start` section immediately after the intro — "run it in 30 seconds" with a 5-line MCP config. A Godot dev landing on the repo should try it before scrolling.

### 8. Context sections: "Why GFM?" + repo layout + FAQ

- **"Why GFM?"** — short, non-technical rationale for converting RST→GFM (RST files like `class_control.rst` are 272KB with hundreds of `:ref:` cross-refs; GFM is compact and LLM-readable). RST fallback preserved.
- **Repository layout** — a small table (server Go code, CI pipeline, docs) for contributors.
- **FAQ** — 3-5 entries (how to update docs, use your own MCP client, etc.).

## Risks / Trade-offs

- **[Risk] Example drift**: documented configs could drift from reality. → **Mitigation**: each example is tested against the real binary before merging; the stdio config was verified manually.
- **[Risk] Podman vs Docker confusion**: `command: "podman"` may not exist on a user's machine. → **Mitigation**: show both `docker` and `podman` variants; recommend Podman but don't require it.
- **[Trade-off] Not documenting every MCP client**: different agents (Claude Desktop, Cursor, etc.) have slightly different config shapes. → Document the standard `mcpServers` shape; users adapt the wrapper.
- **[Risk] Tool list drifts again**: future tools added without README update. → **Mitigation**: the tools section is a single table that mirrors `RegisterTools()`; flag it in the code review checklist.

## Migration Plan

Docs-only change. Rewrite the README (structure + accuracy + usage modes), verify each example against the real binary, commit. Rollback is trivial (git revert).

## Open Questions

None.
