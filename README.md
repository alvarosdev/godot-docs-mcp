# Godot MCP Documentation Server

[![Docker](https://img.shields.io/badge/Docker-Ready-blue?logo=docker)](https://www.docker.com/)
[![Podman](https://img.shields.io/badge/Podman-Ready-purple?logo=podman)](https://podman.io/)
[![Godot](https://img.shields.io/badge/Godot-3.x%20%7C%204.x-478cbf?logo=godot-engine)](https://godotengine.org/)
[![MCP](https://img.shields.io/badge/MCP-Compatible-green)](https://modelcontextprotocol.io/)
[![Go](https://img.shields.io/badge/Go-blue?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Model Context Protocol (MCP) server that gives AI assistants direct access to the complete Godot Engine documentation, across multiple versions, in a format designed for how LLMs consume text.

> **Built with Go** using the official [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk). The original idea came from [Nihilantropy](https://github.com/Nihilantropy)'s Python MCP server.

---

## Table of Contents

- [Quick Start](#quick-start)
- [Why GFM?](#why-gfm)
- [Usage](#usage)
  - [Container, agent-launched (stdio)](#container-agent-launched-stdio)
  - [Container, self-hosted (HTTP)](#container-self-hosted-http)
  - [Native (no container)](#native-no-container)
- [Environment Variables](#environment-variables)
- [Multi-Version Support](#multi-version-support)
- [Available Tools](#available-tools)
- [Sample Usage](#sample-usage)
- [Image Tags & Updates](#image-tags--updates)
- [Repository Layout](#repository-layout)
- [FAQ](#faq)
- [License](#license)

---

## Quick Start

Try it in 30 seconds:

```bash
docker run -d -p 8000:8000 \
  -e MCP_AUTH_TOKEN=your-token \
  ghcr.io/alvarosdev/godot-docs-mcp:latest
```

Then point your MCP client at `http://localhost:8000/mcp`:

```json
{
  "mcpServers": {
    "godot-docs": {
      "type": "http",
      "url": "http://localhost:8000/mcp",
      "headers": { "Authorization": "Bearer your-token" }
    }
  }
}
```

If you didn't set `MCP_AUTH_TOKEN`, omit the `headers` block.

---

## Why GFM?

Godot's documentation is authored in RST, which is great for the engine's docs build but heavy for an LLM to consume directly — a single `class_control.rst` is 272KB with hundreds of cross-references. This server converts the docs to clean GitHub-Flavored Markdown once, at build time, so the documentation your AI reads is compact and readable. Files that don't convert cleanly keep their original RST as a fallback, so nothing is ever lost.

---

## Usage

### Container, agent-launched (stdio)

The simplest experience: your MCP client launches the container itself and talks to it over stdin/stdout. No port mapping, no daemon to manage — the agent handles the container lifecycle.

Add this to your MCP client config (Podman shown; swap `podman` → `docker` if you use Docker):

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

The three flags matter:

- `-i` keeps stdin open so the agent can send messages (required)
- **no** `-t` — a terminal (TTY) would corrupt the protocol (do not add it)
- `--rm` removes the container when the agent closes it

Auth (`MCP_AUTH_TOKEN`) is not used in stdio mode — the container runs locally as a child of your agent, so there's no network boundary to protect.

### Container, self-hosted (HTTP)

Run your own always-on server and connect multiple clients to it. Recommended when you want it hosted separately from a single agent.

**With Docker or Podman:**

```bash
docker run -d -p 8000:8000 \
  -e MCP_AUTH_TOKEN=your-token \
  ghcr.io/alvarosdev/godot-docs-mcp:latest
```

**Recommended: Docker Compose.** The repo ships a [`docker-compose.yml`](docker-compose.yml) that wires up the env vars and a healthcheck:

```bash
docker compose up -d
```

The compose file exposes the env vars listed in [Environment Variables](#environment-variables) — edit the `environment:` block or pass them via a `.env` file.

### Native (no container)

For development or when you want a single static binary with no container runtime.

```bash
# Build a static binary (outputs to dist/)
make build

# Or run directly for development
make run          # HTTP mode
go run ./cmd/godot-mcp-server

# Cross-compile for linux/arm64
make build-arm64
```

The binary is fully static — it needs no Go toolchain at runtime. Documentation must be available under `docs/` (or point `DOCS_DIR` at your docs directory).

---

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `FASTMCP_TRANSPORT` | `http` | Transport mode: `http` (self-hosted) or `stdio` (agent-launched) |
| `FASTMCP_HOST` | `127.0.0.1` | Bind address (HTTP only) |
| `FASTMCP_PORT` | `8000` | Port to listen on (HTTP only) |
| `FASTMCP_PATH` | `/mcp` | HTTP endpoint path (HTTP only) |
| `MCP_AUTH_TOKEN` | *(empty)* | Bearer token required for HTTP auth; ignored in stdio |
| `DOCS_DIR` | `docs` | Directory containing the versioned documentation |

---

## Multi-Version Support

The server ships with the **latest stable minor of each major Godot version** in a single image:

| Version | Branch |
|---------|--------|
| 3.6 | Latest Godot 3.x docs |
| 4.7 | Latest Godot 4.x docs |

A `versions.json` manifest at the docs root provides discovery:

```json
{
  "versions": ["3.6", "4.7"],
  "latest": "4.7",
  "built": "2026.08.11"
}
```

---

## Available Tools

The server exposes five MCP tools:

| Tool | Purpose |
|------|---------|
| **`get_available_versions()`** | List available Godot versions and metadata. Call this first. |
| **`get_documentation_tree(version?)`** | Tree-style overview of the docs for a version. Defaults to latest. |
| **`get_documentation_file(file_path, version?, section?)`** | Content of a specific doc file, optionally a single GFM section (e.g. `Methods`). Defaults to latest version. |
| **`search_documentation(query, version?, category?, limit?)`** | Full-text search across the docs, ranked by relevance. `category` filters to `classes`/`tutorials`. |
| **`get_class_overview(class, version?)`** | Structured class metadata as JSON — inheritance, summary, member counts, and member lists. |

### Resource URIs

Documentation files are also exposed as resources with an optional version prefix:

```
file://4.7/classes/class_camera2d.md   → Godot 4.7 Camera2D docs
file://3.6/classes/class_camera2d.md   → Godot 3.6 Camera2D docs
file://classes/class_camera2d.md        → latest version
```

---

## Sample Usage

**Discover available versions:**
```
What Godot documentation versions are available?
```
→ The LLM calls `get_available_versions()` first

**Find docs without knowing the exact class name:**
```
How do I handle collisions in Godot?
```
→ The LLM calls `search_documentation("collision")`, then reads the top result

**Get a structured class overview:**
```
What methods does Control have in Godot 4.7?
```
→ The LLM calls `get_class_overview(class="Control", version="4.7")`

**Get a specific section of a large file:**
```
Show me the signals of Node2D
```
→ The LLM calls `get_documentation_file("classes/class_node2d.md", version="4.7", section="Signals")`

**Compare across versions:**
```
What's the difference between Node2D in Godot 3.6 vs 4.7?
```
→ The LLM fetches both versions and compares

---

## Image Tags & Updates

Images are tagged by **build date** rather than Godot version:

| Tag | Meaning |
|-----|---------|
| `latest` | Most recent build |
| `2026.08.11` | Docs snapshot built on August 11, 2026 |

Use `latest` to always get fresh docs. Pin to a `YYYY.MM.DD` tag for reproducibility.

**Supported architectures:** `linux/amd64`, `linux/arm64`

### How often is the image updated?

The image is rebuilt **only when the Godot documentation actually changes**, not on a fixed schedule:

1. A scheduled workflow runs **weekly** (Sundays) and checks the HEAD SHA of each tracked Godot docs branch via the GitHub API.
2. SHAs are compared against [`versions.lock`](versions.lock) in this repo.
3. If **any** SHA differs → docs are regenerated, a new multi-arch image is built and pushed with `YYYY.MM.DD` + `latest` tags, and `versions.lock` is updated.
4. If **no** SHAs differ → the build is skipped.

So there's no fixed image cadence — it's "on docs change, checked weekly". To get the latest, run:

```bash
docker pull ghcr.io/alvarosdev/godot-docs-mcp:latest
```

To force a rebuild manually, use the **workflow_dispatch** trigger in the Actions tab (optionally specifying `godot_versions: "3.6 4.7"`).

---

## Repository Layout

| Path | What it is |
|------|-----------|
| `cmd/godot-mcp-server/` | The MCP server binary entry point |
| `internal/` | Go packages: `server`, `tools`, `resources`, `docs`, `search` |
| `scripts/generate-docs.sh` | CI pipeline: download Godot docs, convert RST→GFM, build the docs artifact |
| `docs/` | Generated documentation (built in CI, not committed) |
| `.github/workflows/` | CI/CD: doc detection, build, and publish |
| `Makefile` | Build helpers: `make build`, `make run`, `make test`, etc. |

---

## FAQ

**How do I update the documentation?**
```bash
docker pull ghcr.io/alvarosdev/godot-docs-mcp:latest
```
Or regenerate locally: `bash scripts/generate-docs.sh "3.6,4.7"` (requires pandoc, tree, curl, unzip, jq).

**Can I use my own MCP client?**
Yes. The server speaks standard MCP over stdio or HTTP — Claude Desktop, Cursor, and other MCP-compatible agents work.

**Why does the image have both `3.6` and `4.7` docs?**
The server ships the latest stable minor of each Godot major so you can compare APIs across engine versions.

**How is auth handled?**
HTTP mode uses an optional bearer token (`MCP_AUTH_TOKEN`). Stdio mode needs no auth — the container runs locally as a child of your agent.

---

## Recommended System Prompt

For optimal results, pair the server with this system prompt:

> "When working with Godot game development questions, use the godot-docs MCP tools. Start with `get_available_versions()` to see which Godot versions are available. Use `search_documentation(query)` to find relevant documentation, `get_class_overview(class, version)` for structured class metadata, and `get_documentation_file(path, version, section)` to read specific content. When the user asks about a specific Godot version, always query that version's docs. Prioritize official Godot documentation over general knowledge when providing Godot-related assistance."

---

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

The Godot documentation content follows the original Godot documentation licensing:
- Documentation content (excluding `classes/` folder): [CC BY 3.0](https://creativecommons.org/licenses/by/3.0/)
- Class reference files (`classes/` folder): MIT License
- Attribution: "Juan Linietsky, Ariel Manzur and the Godot community"
