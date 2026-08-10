# Godot MCP Documentation Server

[![Docker](https://img.shields.io/badge/Docker-Ready-blue?logo=docker)](https://www.docker.com/)
[![Godot](https://img.shields.io/badge/Godot-3.x%20%7C%204.x-478cbf?logo=godot-engine)](https://godotengine.org/)
[![MCP](https://img.shields.io/badge/MCP-Compatible-green)](https://modelcontextprotocol.io/)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Model Context Protocol (MCP) server that provides AI assistants with access to the complete Godot Engine documentation across multiple versions, helping developers with Godot development by serving documentation directly to LLMs.

> **Built with Go** — This is a complete rewrite in Go 1.26 using the official [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk). The original idea and Python implementation were created by [Nihilantropy](https://github.com/Nihilantropy).

## Purpose

This server bridges the gap between AI assistants and Godot documentation, allowing developers to get instant, accurate answers about Godot classes, tutorials, and features — for both Godot 3.x and 4.x — without leaving their AI chat interface.

## Multi-Version Support

The server ships with the **latest stable minor of each major Godot version** in a single image. As of writing, that means:

| Version | Branch |
|---------|--------|
| 3.6 | Latest Godot 3.x docs |
| 4.7 | Latest Godot 4.x docs |

A `versions.json` manifest at the docs root provides discovery:

```json
{
  "versions": ["3.6", "4.7"],
  "latest": "4.7",
  "built": "2026.08.03"
}
```

## Deployment

### Pull the pre-built image

```bash
docker pull ghcr.io/alvarosdev/godot-docs-mcp:latest
```

Pin to a specific daily snapshot:

```bash
docker pull ghcr.io/alvarosdev/godot-docs-mcp:2026.08.09
```

### Build locally

1. **Clone the repository:**
   ```bash
   git clone https://github.com/alvarosdev/godot-docs-mcp.git
   cd godot-docs-mcp
   ```

2. **Build the Docker image** (multi-stage Go build, auto-detects latest stable versions):
   ```bash
   docker build -t godot-mcp-docs:local .
   ```

3. **Run with Docker Compose:**
   ```bash
   docker compose up -d
   ```

4. **Configure your MCP client** (Claude Desktop example):
   ```json
   {
     "mcpServers": {
       "godot-docs": {
         "type": "http",
         "url": "http://localhost:8000/mcp",
         "headers": {
           "Authorization": "Bearer your-token-here"
         }
       }
     }
   }
   ```

   If you didn't set `MCP_AUTH_TOKEN`, omit the `headers` block.

## Documentation Structure

```
docs/
├── 3.6/
│   ├── classes/
│   ├── tutorials/
│   └── docs_tree.txt
├── 4.7/
│   ├── classes/
│   ├── tutorials/
│   └── docs_tree.txt
└── versions.json
```

## Available Tools

- **`get_available_versions()`** — Returns available Godot versions and metadata (JSON). Call this first to discover what's available.
- **`get_documentation_tree(version?)`** — Get a tree-style overview of the documentation for a specific version. Defaults to latest when `version` is omitted.
- **`get_documentation_file(file_path, version?)`** — Retrieve the content of a specific documentation file for a given version. Defaults to latest.

### Resource URIs

Documentation files are also exposed as resources with optional version prefix:

```
file://4.7/classes/class_camera2d.md   → Godot 4.7 Camera2D docs
file://3.6/classes/class_camera2d.md   → Godot 3.6 Camera2D docs
file://classes/class_camera2d.md        → latest version
```

## Sample Usage

**Discover available versions:**
```
What Godot documentation versions are available?
```
→ The LLM calls `get_available_versions()` first

**Explore version-specific docs:**
```
Show me the 4.7 documentation tree
```
→ The LLM calls `get_documentation_tree(version="4.7")`

**Get class documentation for a specific version:**
```
How does CharacterBody2D work in Godot 4.7?
```
→ The LLM calls `get_documentation_file("classes/class_characterbody2d.md", version="4.7")`

**Compare across versions:**
```
What's the difference between Node2D in Godot 3.6 vs 4.7?
```
→ The LLM fetches both versions and compares

## Image Tags

Images are tagged by **build date** rather than Godot version:

| Tag | Meaning |
|-----|---------|
| `2026.08.09` | Docs snapshot built on August 9, 2026 |
| `latest` | Most recent build (always points to the newest `YYYY.MM.DD` tag) |

Use `latest` to always get fresh docs. Pin to a `YYYY.MM.DD` tag for reproducibility.

**Supported architectures:** `linux/amd64`, `linux/arm64`

## CI/CD — Conditional Builds

The image is rebuilt **only when docs actually change**, not on a fixed schedule:

1. A scheduled workflow runs weekly and fetches the HEAD SHA of each tracked Godot docs branch via the GitHub API.
2. SHAs are compared against [`versions.lock`](versions.lock) in this repo.
3. If **any** SHA differs → docs are pre-generated, a native multi-arch image is built and pushed with `YYYY.MM.DD` + `latest` tags, and `versions.lock` is updated.
4. If **no** SHAs differ → the build is skipped entirely.

### Manual trigger

To force a rebuild, use the **workflow_dispatch** trigger in the Actions tab. You can optionally specify versions:

```
godot_versions: "3.6 4.7"
```

Leave blank to auto-detect the latest of each major.

## Updating Documentation

**Option 1: Pull the latest image**
```bash
docker pull ghcr.io/alvarosdev/godot-docs-mcp:latest
```

**Option 2: Rebuild locally with updated docs**
```bash
docker build -t godot-mcp-docs:local .
```

**Option 3: Regenerate docs locally**
```bash
# Requires: pandoc, tree, curl, unzip, jq
bash scripts/generate-docs.sh "3.6,4.7"
```

## Recommended System Prompt

For optimal results when working with Godot, use this system prompt:

> "When working with Godot game development questions, always search for the latest available documentation using the godot-mcp-docs tools. Start with `get_available_versions()` to see which Godot versions are available, then use `get_documentation_tree(version)` for an overview and `get_documentation_file(path, version)` to retrieve specific information. When the user asks about a specific Godot version, always query that version's docs. Prioritize official Godot documentation over general knowledge when providing Godot-related assistance."

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

The Godot documentation content follows the original Godot documentation licensing:
- Documentation content (excluding `classes/` folder): [CC BY 3.0](https://creativecommons.org/licenses/by/3.0/)
- Class reference files (`classes/` folder): MIT License
- Attribution: "Juan Linietsky, Ariel Manzur and the Godot community"
