## Context

The docs pipeline currently serves raw RST (`generate-docs.sh` downloads and cleans). The Go tools (`findSection`, `listSections`) parse RST underline headings, and `get_documentation_file` returns full RST files. We proved that pandoc → GFM produces clean output (legible tables, headers) once the Sphinx `.. table::` directive is stripped before conversion. See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**
- Convert RST → GFM in CI (`generate-docs.sh`), generating docs once
- Preserve `.rst` fallback for files pandoc cannot convert
- Adapt section parsing to GFM headings (`## Methods` not `Methods\n-------`)
- Add `get_class_overview` returning structured JSON via regex over GFM
- Remove dead code and optimize hot paths (regex hoisting, no recompilation in loops)
- Comprehensive tests including edge cases

**Non-Goals:**
- Full RST/GFM parser library (regex extraction is sufficient — the LLM consumes the result, not machine logic)
- bleve or full-text index (see earlier exploration — grep suffices for 33MB, LLM does the ranking)
- JSON AST generation in CI (pandoc `-t json`) — deferred; regex over GFM is enough initially
- Changing `DocStore` — it already loads both `.md` and `.rst`

## Decisions

### 1. Pandoc conversion in `generate-docs.sh` with `.. table::` pre-strip

**Approach**: Before each pandoc call, strip the Sphinx `.. table::` and `:widths:` directives that break grid-table rendering. Then convert RST → GFM with `--wrap=none`, and post-process with sed to remove `classref-*` lines and `<div class="rst-class">` blocks.

```bash
sed '/^\.\. table::$/d; /^   :widths:/d' "$1" \
  | pandoc -f rst -t gfm --wrap=none 2>/dev/null \
  | sed '/^<div class="rst-class">$/,/^<\/div>$/d; /^classref-/d; ...' \
  > "${1%.rst}.md"
```

**Why GFM over plain markdown**: GFM preserves headers and grid tables as markdown tables (`| a | b |`), which are legible to LLMs and match what we tested. Plain text loses structure.

**Alternatives considered**: pandoc `-t markdown` (loses tables), serving raw RST (the problem we're solving), docutils in CI (robust but adds Python + second generator — rejected).

### 2. RST fallback: delete source only on successful conversion

**Approach**: The pandoc pipeline deletes the `.rst` only when conversion succeeds (`pandoc && rm`). Failed conversions leave `.rst` in place, and the generic cleanup preserves `.rst` files. `DocStore.Load` already loads both `.md` and `.rst`.

**Why**: "A lost `.rst` is worse than a non-optimal `.rst`" — the LLM can read RST natively as a last resort.

### 3. Section parsing: GFM headings

**Approach**: `findSection`/`listSections` detect GFM ATX headings (`## Methods`, `### Method Descriptions`) instead of RST underline headings. Parse line-by-line: a line matching `^#{1,6}\s+(.+)$` is a heading; section content extends to the next heading of equal-or-higher level.

**Why**: GFM uses ATX headings for all sections (`## Properties`, `## Methods`, `## Property Descriptions`, `## Method Descriptions`). Underline detection is dead after the switch.

### 4. `get_class_overview`: regex over GFM content

**Approach**: A new `internal/tools/overview.go` extracts structured metadata from the GFM content with hoisted, precompiled regexes (single-pass):

```go
var (
    inheritsRe = regexp.MustCompile(`\*\*Inherits:\*\* (.*)`)
    memberRe   = regexp.MustCompile(`>\s*\| ([^|]+) \| ([^|]+) \|`)
    headingRe  = regexp.MustCompile(`^#{1,6}\s+(.+)$`)
)
```

- **Class name**: first H1 (`# Control`)
- **Inherits**: `**Inherits:** ...` line
- **Summary**: first paragraph after `## Description`
- **Counts**: count member-table rows per section
- **Members**: extract `| type | name() badges |` rows from each member table

**Why regex over a parser**: 30 lines, stdlib-only, fast over 150KB in-memory. The LLM consumes the result, so imperfect structure is acceptable. A full parser (80-100 lines) is over-engineering for this volume.

### 5. Dead code removal and optimization

**Removals**:
- RST-only sed branches in `generate-docs.sh` (the old `.rst`-preservation generic pass that kept everything — now scoped)
- Any `findSection` RST-underline logic that GFM headings replace
- Unused constants/functions in `internal/search` discovered during the switch (e.g., badge substitution that no longer applies since pandoc+sed cleans badges)

**Optimizations**:
- Hoist `regexp.MustCompile` to package-level (compile once, not per-call) in `overview.go`
- In `internal/search/scan.go`, hoist the tokenizer's stop-word lookups and avoid re-`strings.ToLower` per iteration where the lowercased form can be computed once
- `extractSnippet`/`findPositions`: avoid re-slicing content repeatedly — single pass with `strings.Index` continuation

### 6. Backward compatibility of search

**Approach**: `search_documentation` and `get_documentation_file` keep the same signatures. Only the extension in paths/error messages changes (`.rst` → `.md`). `category` filter and `filename boost` operate on paths, extension-agnostic — no change needed.

## Risks / Trade-offs

- **[Risk] Pandoc output changes between versions**: GFM table syntax could change on a pandoc upgrade, breaking regexes. → **Mitigation**: Pin pandoc in CI (`apt install pandoc=…` is brittle; instead add tests that freeze the expected GFM structure with a fixture `.md`).
- **[Risk] `.. table::` variants**: Some RST files may use different directive syntax. → **Mitigation**: The sed strips the two known forms; any residue is covered by the `.rst` fallback (file kept, not silently corrupted).
- **[Risk] Regex fragility on member rows**: `> | type | name() |` format could vary. → **Mitigation**: Tests cover multiple real class files; if a row shape differs, the row is skipped (never panics) and the `.rst` fallback remains.
- **[Trade-off] Regex vs parser**: Less robust than a parser but 30 lines vs 80-100 and sufficient for LLM consumption.

## Migration Plan

1. Update `generate-docs.sh` with the pandoc step (pre-strip, convert, post-clean, delete `.rst` on success)
2. Add pandoc back to CI `apt-get install`
3. Update `findSection`/`listSections` to GFM headings
4. Add `get_class_overview` tool + regex extraction
5. Update paths/errors from `.rst` to `.md`
6. Remove dead code, optimize hot paths
7. Add tests + edge cases
8. Regenerate docs locally (`bash scripts/generate-docs.sh "4.7"`) and verify MCP tools against the GFM output

**Rollback**: Keep the `.rst` fallback in the store. If GFM output regresses, revert `generate-docs.sh` and the section parser — the tool signatures are unchanged so clients are unaffected.

## Open Questions

None.
