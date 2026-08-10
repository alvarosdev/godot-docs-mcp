## Context

The current system downloads a single godot-docs branch, converts all `.rst` files to `.md`, and serves them through MCP tools that read from a flat `docs/` directory. The Dockerfile bakes one version at build time via `ARG GODOT_VERSION`.

See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**
- Support multiple Godot doc versions in one image without duplicating the MCP server
- Keep tools backward-compatible where possible (default to latest when version omitted)
- Rebuild the image only when docs actually change
- Make image tags human-readable and meaningful (date-based)

**Non-Goals:**
- Patch-level version granularity (Godot docs only use `major.minor` branches)
- Supporting `master`/unstable branch alongside stable versions
- On-the-fly doc updates without rebuilding the image
- Per-version image tags (an image always bundles all versions)

## Decisions

### 1. Directory structure: `docs/{version}/`

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

**Alternatives considered:**
- Flat with filename prefix (`docs/3.6_classes_camera2d.md`): breaks path conventions, harder to navigate
- Separate images per version: simpler per-image but forces users to run multiple containers

**Rationale:** Version-prefixed directories are the natural layout. `versions.json` at the root provides discovery without file-system scanning.

### 2. `versions.json` as the single source of truth for version metadata

```json
{
  "versions": ["3.6", "4.7"],
  "latest": "4.7",
  "built": "2026.08"
}
```

Generated at build time by the docs converter after all versions are processed. The MCP server reads it once at startup. `get_available_versions()` returns this data directly.

**Rationale:** Avoids filesystem scanning (slow, fragile) and provides a single place for metadata.

### 3. Version resolution: default to latest when omitted

Both `get_documentation_tree(version?)` and `get_documentation_file(path, version?)` default to the version marked as `latest` in `versions.json` when no `version` argument is passed.

**Rationale:** Backward-compatible for simple queries. Power users (and LLMs) can pin a specific version. This keeps the common path short.

### 4. Converter: sequential download, parallel conversion per version

```
for version in [3.6, 4.7]:
    download_repo(version)          → downloads godot-docs-{version}.zip
    convert_files_parallel()        → converts .rst → .md (already parallel internally)
    move docs/ → docs/{version}/
generate versions.json
```

**Alternatives considered:**
- Parallel downloads: the ZIP download is I/O-bound and GitHub rate-limits; sequential is safer and fast enough (~30s per version)
- Incremental conversion (only changed files): not worth the complexity; full conversion takes <2min per version

### 5. Conditional build: GitHub API SHA comparison

The workflow fetches the HEAD SHA of each tracked branch via `GET /repos/godotengine/godot-docs/git/ref/heads/{version}` and compares against `versions.lock` in the repo.

```
versions.lock:
{
  "3.6": "abc123def456789...",
  "4.7": "789ghi012jkl345..."
}
```

**Alternatives considered:**
- Docker image labels: requires pulling the previous image to read labels, slower and adds registry dependency
- Content diff (download zip, hash files): wastes bandwidth on every check

**Rationale:** API calls are free (no download), fast (~200ms each), and the lockfile doubles as an audit log of when each version was last updated.

### 6. Version detection: group branches by major, pick max minor in each

The action already lists all `X.Y` branches. The new logic groups by major (`3`, `4`) and selects the highest minor within each group. This automatically picks up new majors (Godot 5.x) and new minors (4.8, 4.9) without manual intervention.

**Edge case:** When a new major version appears (e.g., Godot 5.0), it is automatically included in the next build. If this is undesirable, the `versions.lock` can be manually pinned to exclude it.

### 7. Image tags: `YYYY.MM.DD` + `latest`

| Tag | Meaning |
|-----|---------|
| `2026.08.03` | Docs snapshot from August 3, 2026 |
| `latest` | Always points to the most recent build |

The date is derived from the build date, not the docs dates. Each build gets a unique tag. With conditional rebuilds (only when docs change), most weeks produce no tag at all, so the registry stays clean.

**Alternatives considered:**
- `YYYY.MM`: human-readable but two builds in the same month would collide (first tag gets overwritten)
- Content hash (`sha256-abc12`): cryptographically precise but not human-readable

## Risks / Trade-offs

- **[Risk] Image size grows linearly with versions.** Each version is ~50MB (converted .md files). Two versions = ~100MB, three = ~150MB. → **Mitigation:** We only include the latest of each major (2 versions today, maybe 3 in the future). Alpine base keeps the image lean.
- **[Risk] Automatic major version inclusion may be surprising.** When Godot 5.0 ships, it gets auto-included. → **Mitigation:** This is documented behavior. The lockfile can be manually edited to exclude versions before the next scheduled build.
- **[Risk] Race condition: docs update between SHA check and download.** The branch HEAD could advance between the API check and the ZIP download. → **Mitigation:** Acceptable risk. The image will be at most one commit behind; the next scheduled build catches it. The window is seconds.
- **[Trade-off] Sequential version downloads increase build time.** ~1min per version for download + conversion. Two versions = ~2min extra. Acceptable for a weekly build.

## Migration Plan

1. Deploy the new GitHub Action workflow (it replaces the old one)
2. Delete old image tags (`v4.7`, `v3.6`) — the new `latest` and `YYYY.MM.DD` tags replace them
3. Existing `docker-compose.yml` users update `GODOT_VERSION` to pull the new `latest` image
4. MCP clients (Claude Desktop, etc.) update their tool schemas — the server advertises the new signatures automatically via FastMCP

## Open Questions

*(none — all design decisions are resolved)*
