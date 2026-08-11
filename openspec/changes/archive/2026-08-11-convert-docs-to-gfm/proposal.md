## Why

Raw RST class files are too heavy for LLM tool-call contexts. A single `class_control.rst` is 272KB with 867 cross-references (`:ref:`), 270 badges (`|virtual|`), and RST grid tables that render poorly in plain text — even models with 1M-token contexts waste most of it on markup. We proved `pandoc` to GitHub-Flavored Markdown (GFM) produces clean, legible output (16.5KB vs 24.5KB for a class file) once the Sphinx `.. table::` directive is stripped. This change converts the docs pipeline to produce GFM, adapts the Go tooling to the new format, adds a `get_class_overview` tool, and cleans up dead code and fragile regex as part of the migration.

## What Changes

- **BREAKING**: `generate-docs.sh` converts RST → GFM Markdown via pandoc (with `.. table::` pre-strip fix) instead of serving raw RST
- **BREAKING**: `.rst` files that pandoc fails to convert are preserved as fallback; successful conversions delete the source `.rst`
- `get_documentation_file` paths and error messages change from `.rst` to `.md`
- `findSection`/`listSections` adapt from RST underline headings (`Methods\n-------`) to GFM headings (`## Methods`)
- Add `get_class_overview` tool returning structured JSON (class, inherits, summary, counts, members) via regex extraction over the GFM content
- Remove dead code: the RST-only fallback paths in `generate-docs.sh`, any unused constants/functions in `internal/search`
- Optimize hot paths: tokenizer, scoring, section extraction (avoid re-splitting/recompiling in loops, hoist compiled regex)
- **No new runtime dependencies** — pandoc runs in CI only; the Go binary stays static
- Add comprehensive tests including edge cases

## Capabilities

### Modified Capabilities

- `multi-version-converter`: Docs pipeline converts RST → GFM with pandoc instead of serving raw RST; failed conversions fall back to `.rst`
- `version-aware-tools`: `get_documentation_file` returns GFM content and `section` parsing targets GFM headings. Adds `get_class_overview` tool. Path/error references use `.md`
- `documentation-search`: Search operates over GFM content (same interface, updated indexing of `.md` files)

### New Capabilities

- `class-overview`: Structured class metadata extraction (`get_class_overview`) via regex over GFM, returning JSON with class name, inheritance, summary, member counts, and member lists

## Impact

- **Files modified**: `scripts/generate-docs.sh` (pandoc step + cleanup), `internal/tools/navigation.go` (section parsing, paths, new tool), `internal/search/scan.go` (regex optimization, dead code removal)
- **Files created**: `internal/tools/overview.go` + `internal/tools/overview_test.go`, updated `internal/tools/navigation_test.go`, `internal/search/scan_test.go` edge cases
- **Docs output**: `docs/**/*.rst` → `docs/**/*.md` (GFM) + `.rst` fallback only for failures
- **Runtime**: +0 dependencies, binary size unchanged, startup unchanged
- **CI**: pandoc added to `apt-get install` (was removed earlier, now restored with the table fix)
