## 1. Docs Converter — Multi-Version Support

- [x] 1.1 Add `resolve_versions()` function that fetches all `X.Y` branches from the godot-docs repo, groups by major, and picks the highest minor per major
- [x] 1.2 Modify `download_repo()` to accept a `target_subdir` parameter so each version downloads to a version-specific directory
- [x] 1.3 Refactor `godot_docs_converter.py` to accept multiple versions (comma-separated or `--all` flag) and loop through them
- [x] 1.4 Generate `docs/versions.json` after all versions are processed, writing `{"versions": [...], "latest": "...", "built": "YYYY.MM.DD"}`
- [x] 1.5 Handle per-version conversion failures gracefully: log the error, skip the failed version, continue with remaining versions, exit non-zero at the end

## 2. MCP Utilities — Version-Aware Path Resolution

- [x] 2.1 Refactor `srcs/utils/docs_utils.py`: replace the flat `DOCS_DIR` constant with version-aware helpers (`get_version_docs_dir(version)`, `get_latest_version()`, `get_available_versions_metadata()`)
- [x] 2.2 Add `load_versions_metadata()` that reads `docs/versions.json` and caches the result
- [x] 2.3 Update `validate_doc_path()` to accept an optional version parameter and resolve paths under `docs/{version}/`

## 3. MCP Tools — Version Parameters

- [x] 3.1 Add `get_available_versions()` tool that returns the contents of `versions.json` (or an error if no docs are available)
- [x] 3.2 Update `get_documentation_tree(version?)` to accept an optional `version` parameter, defaulting to latest when omitted
- [x] 3.3 Update `get_documentation_file(file_path, version?)` to accept an optional `version` parameter, defaulting to latest
- [x] 3.4 Update `get_documentation_resource()` to parse a version prefix from the resource path (`{version}/{doc_path}`) and fall back to latest when no prefix is present

## 4. Docker — Multi-Version Build

- [x] 4.1 Update `Dockerfile` to accept a space-separated `GODOT_VERSIONS` build arg and loop through versions in the converter step
- [x] 4.2 Generate per-version `docs_tree.txt` files inside each version directory during build
- [x] 4.3 Remove the single-version `GODOT_VERSION` arg; replace with `GODOT_VERSIONS` defaulting to the two latest stable versions
- [x] 4.4 Update `docker-compose.yml` to use the new `GODOT_VERSIONS` build arg

## 5. GitHub Action — Conditional Build

- [x] 5.1 Add a detection job that fetches HEAD SHAs for each tracked version branch via GitHub API and compares against `versions.lock`
- [x] 5.2 Implement the version-selection logic: fetch all `X.Y` branches, group by major, pick the highest minor per major
- [x] 5.3 Change image tags from `v{version}` to `YYYY.MM.DD` + `latest` using the build date
- [x] 5.4 Add a step that updates `versions.lock` with new SHAs and commits + pushes it back to the default branch on successful build
- [x] 5.5 Skip the build job entirely when no SHAs changed (exit early in the detection job)
- [x] 5.6 Retain `workflow_dispatch` support with an optional `godot_version` input that overrides automatic version selection

## 6. Documentation

- [x] 6.1 Update `README.md` with multi-version usage: new tool signatures (`get_available_versions`, versioned `get_documentation_tree`, versioned `get_documentation_file`), version-prefixed resource paths, and `versions.json` metadata
- [x] 6.2 Document the new Docker image tagging scheme (`YYYY.MM.DD` + `latest`) and how to pin or float with `latest`
- [x] 6.3 Document the conditional build behavior: how `versions.lock` works, how to trigger a manual build via `workflow_dispatch`, and how to force a rebuild

## 7. Validation & Cleanup

- [x] 7.1 Test the converter locally with `python docs_converter/godot_docs_converter.py --all` and verify `docs/3.6/`, `docs/4.7/`, and `versions.json` are created
- [x] 7.2 Test the MCP server locally with `python main.py` (stdio mode) and verify `get_available_versions()`, versioned `get_documentation_tree()`, and versioned `get_documentation_file()` work correctly
- [x] 7.3 Build the Docker image locally and verify the multi-version directory structure inside the container
- [x] 7.4 Verify the GitHub Action detection logic manually by running the SHA comparison script against the live godot-docs API
