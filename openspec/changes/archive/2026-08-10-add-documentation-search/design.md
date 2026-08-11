## Context

The current MCP server serves documentation as a file store: the LLM calls `get_documentation_tree` to see available files, then `get_documentation_file` to read specific ones. This requires the LLM to know the exact file path. For queries like "how does collision detection work?", the LLM must guess which file contains the answer. A search tool provides direct discovery without file-path guessing.

The doc store holds ~5000 files / ~29MB of Markdown text loaded into an immutable `map[string]string` at startup. Searches scan this map in-memory — no disk I/O, no external service, no index build step.

See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**
- Single `search_documentation` tool returning ranked results with file path, version, score, and text snippet
- Zero external dependencies — standard library only
- Scored ranking using token coverage, heading matches, section type, and match proximity
- Optional `version` filter to scope search to a specific Godot version
- Configurable result limit (default 10, max 50)

**Non-Goals:**
- Fuzzy matching or typo tolerance (the LLM produces clean queries)
- Query operators (`AND`, `OR`, `-exclude`) — not needed for LLM-generated queries
- Real-time index updates (docs are immutable after startup)
- Semantic/embedding search (requires external model)

## Decisions

### 1. In-memory map scan instead of bleve or other full-text index

**Why**: Benchmarked bleve: +53 dependencies, +5.8MB binary, +20-40MB RAM, +2-5s startup. For 29MB of text, a linear scan of the existing `map[string]string` takes 20-50ms — imperceptible for interactive use. No dependencies, no binary bloat, no startup cost.

**Why the LLM use case makes this viable**: An LLM needs RECALL (the right file appears somewhere in the top 10-15), not PRECISION (the right file is #1). The LLM reads snippets and selects the most relevant file. A scan with token-based scoring achieves sufficient recall for this pattern.

**Alternatives considered**:
- bleve — rejected for weight (+53 deps, +5.8MB) vs benefit for 29MB corpus
- FTS5 via `mattn/go-sqlite3` — rejected; adds CGO dependency, breaks static compilation
- `regexp` scan — rejected; too slow for 5000 files, no ranking

### 2. Four-factor scoring algorithm

```
score = token_coverage × heading_boost × section_boost × proximity_bonus
```

**token_coverage** (0.0–1.0): Fraction of query tokens found in the document. A query with 3 tokens where all 3 match gets 1.0; 1 of 3 gets 0.33. Tokens are lowercased and split on whitespace. Stop words (a, the, is, of, etc.) are excluded.

**heading_boost** (1.0–3.0): Tokens matching in a heading line (`# ...` or `===`) count 3× compared to tokens in body text. Heading lines are detected by `#` prefix (Markdown) or `===`/`---` underline (RST-style).

**section_boost** (1.0–2.0): Documents in `classes/` get 2.0× (API reference is the primary use case). Documents in `tutorials/` get 1.5×. Everything else gets 1.0×.

**proximity_bonus** (0.5–1.5): When multiple tokens appear within a sliding window of 500 characters, the document gets a bonus proportional to the match density. Tokens scattered across 50KB of text get 0.5×. Tokens clustered in the same paragraph get 1.5×.

### 3. Snippet extraction

The snippet shows the first occurrence of any query token with ±100 characters of surrounding context. If multiple tokens match in different locations, the snippet is extracted from the highest-density region (most matches in the smallest window). Markdown formatting (headings, code fences) is stripped for readability.

### 4. Search tool API design

```json
{
  "name": "search_documentation",
  "description": "Search Godot documentation by keyword or phrase. Returns ranked results with file paths and text snippets.",
  "parameters": {
    "query":    { "type": "string", "required": true },
    "version":  { "type": "string", "required": false },
    "limit":    { "type": "integer", "required": false, "default": 10 }
  }
}
```

**Why version as optional filter**: Most queries are version-specific ("how does X work in 4.7?"). An optional filter avoids cross-version noise. When omitted, searches all versions and includes version in results.

**Why limit with default**: The LLM rarely needs more than 10 results. A cap prevents token waste. Max 50 for edge cases.

### 5. DocStore iteration support

`DocStoreReader` gains a `Keys() []string` method that returns all stored paths. This enables scanning without exposing the internal map. The returned slice is freshly allocated — the underlying map is never exposed.

```go
type DocStoreReader interface {
    Get(key string) (string, bool)
    Size() int
    Keys() []string   // NEW
}
```

## Risks / Trade-offs

- **[Risk] Scan latency at 100K+ files**: Currently 5000 files. If docs grow 10× (50K+ files, 300MB+), scan latency may exceed 500ms. → **Mitigation**: The search engine can be swapped for bleve later without changing the tool API. The `DocStoreReader` interface isolates the dependency.
- **[Risk] Token-only scoring misses synonyms**: "velocity" won't match "speed". → **Mitigation**: The LLM generates queries with multiple phrasings. If the user says "speed", the LLM typically also tries "velocity", "movement", etc. in the query. This is acceptable for the use case.
- **[Trade-off] Section boost is hardcoded**: `classes/` > `tutorials/` > other. → If Godot reorganizes docs (e.g., `api/` instead of `classes/`), the boost won't apply. Acceptable — it's a heuristic, not a correctness requirement.

### 6. RST fallback when pandoc conversion fails

The cleanup step in `generate-docs.sh` currently deletes all non-`.md` files. The conversion step already deletes `.rst` files on success (`pandoc ... && rm "$1"`), so only failed conversions leave `.rst` files behind. The cleanup step excludes `.rst` from deletion (`! -name '*.rst'`) to preserve those ~6 fallback files. The DocStore also gains `.rst` loading so the server can serve them.

This also future-proofs for a potential switch to serving RST natively (eliminating pandoc entirely).

## Open Questions

None.
