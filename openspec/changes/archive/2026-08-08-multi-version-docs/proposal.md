## Why

The Godot MCP server currently packages a single version of Godot docs per Docker image. Users working with Godot 3.x projects have no way to query 3.x docs; those on 4.x are locked to whichever version was baked into the image. Additionally, the GitHub Action rebuilds the image every Sunday without checking whether the docs actually changed — wasting CI minutes and registry storage.

## What Changes

- **Multi-version docs in a single image**: the latest stable of each major version (currently 3.6 and 4.7), downloaded and converted during build, stored under `docs/{version}/`
- **Version-aware MCP tools**:
  - `get_available_versions()` — new tool returning available versions and metadata
  - `get_documentation_tree(version?)` — accepts optional `version`; defaults to latest when omitted
  - `get_documentation_file(path, version?)` — accepts `version` parameter; defaults to latest
  - **BREAKING**: resource URIs now include version as a path prefix (`file://3.6/classes/...`)
- **Conditional rebuilds**: `versions.lock` in the repo stores the HEAD SHA of each version branch. The scheduled workflow queries the GitHub API to compare; only builds if a SHA changed
- **Date-based image tags**: `YYYY.MM.DD` instead of `v{version}` semver. E.g., `2026.08.03`, `latest`
- **Metadata**: `versions.json` inside the image with available versions, which one is latest, and build date
- **Documentation**: README updated with multi-version usage, new tool signatures, date-based tags, and conditional build behavior

## Capabilities

### New Capabilities
- `multi-version-converter`: Download and convert multiple Godot docs versions into a versioned directory structure `docs/{version}/`
- `version-aware-tools`: MCP tools (`get_available_versions`, `get_documentation_tree`, `get_documentation_file`) that accept a `version` parameter and support version discovery
- `conditional-build`: GitHub Action that compares branch HEAD SHAs against `versions.lock` and only builds the image when docs have actually changed

### Modified Capabilities
*(none — no existing specs to modify)*

## Impact

| Area | Affected files |
|------|---------------|
| Docs converter | `docs_converter/godot_docs_converter.py`, `docs_converter/download_docs.py` |
| MCP Tools | `srcs/tools/navigation_tools.py` |
| MCP Resources | `srcs/resources/doc_resources.py` |
| Utils | `srcs/utils/docs_utils.py` |
| Server bootstrap | `main.py` |
| Docker | `Dockerfile`, `docker-compose.yml` |
| CI/CD | `.github/workflows/build-publish.yml` |
| New file | `versions.lock` (branch SHA metadata) |
| Documentation | `README.md` |
