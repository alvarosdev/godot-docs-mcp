## Context

See proposal.md for motivation. Three small, targeted fixes to the existing tools and search engine — no architectural changes.

## Goals / Non-Goals

**Goals:**
- Enable reading specific RST sections from large files via `section` parameter
- Make class-name searches return the class file as the #1 result
- Allow filtering search results by top-level directory (classes, tutorials)
- Warn the LLM when a file is too large and suggest using sections

**Non-Goals:**
- Index sections at load time (parse on-demand is fast enough for 272KB files)
- Add dedicated per-section tools (section parameter solves this generically)
- Full RST parser (we only need to find section boundaries)

## Decisions

### 1. Section parsing: on-demand string scanning

**Approach**: When `get_documentation_file` is called with a `section` parameter, scan the file content for RST section boundaries and extract only the requested section.

RST sections in Godot class files follow a consistent pattern:
```
Section Name       ← heading text (first line)
-----------        ← underline (second line, all `-` or `=` characters)
```

The parser:
1. Finds the section heading by matching `{SectionName}\n{underline}` (case-insensitive)
2. Extracts content from after the underline until the next section boundary or EOF
3. Returns only that content

**Why on-demand**: Pre-indexing all 5000+ files at startup adds complexity and memory. A 272KB file is scanned in microseconds. The section parameter is used for known sections in known files — the LLM already knows which section it wants.

**Alternatives considered**:
- Pre-parse at startup → rejected; adds startup time and complexity for infrequent use
- Store section offsets in DocStore → rejected; requires changing the immutable store contract

### 2. Filename match boost: 10× for class name tokens

**Approach**: Extract the "stem" from the filename (before the extension and after known prefixes like `class_`). If any query token matches this stem, multiply the final score by 10.

For file `classes/class_characterbody2d.rst`:
- Stem: `characterbody2d`
- Query "CharacterBody2D" → token "characterbody2d" matches → 10× boost

**Why 10×**: The heading boost is 3× and section boost is 2×. Combined they're 6×. A 10× filename boost outranks any non-filename match, ensuring `class_control.rst` beats any incidental mention of "control".

### 3. Category filter: path prefix match

**Approach**: Add `category` parameter to `search_documentation`. When set to `"classes"`, only search keys starting with `"classes/"`. When `"tutorials"`, only `"tutorials/"`. When omitted, search all.

Implementation: add a single `strings.HasPrefix(path, category+"/")` check inside the search loop.

**Why path prefix**: The doc store keys are `"{version}/{path}"`. The `splitVersion` helper already separates version from path. Filtering by path prefix is zero-cost.

### 4. Size warning: simple length check

When `get_documentation_file` returns a file >50,000 characters and no `section` was specified, append a warning to the response: `"Note: This file is {size} characters. Use the 'section' parameter (e.g. section='methods') to read specific parts."`

50K is ~12,000 tokens — a reasonable threshold where truncation becomes likely.

## Risks / Trade-offs

- **[Risk] Section name mismatch**: The LLM might request `section="method"` (singular) when the header is "Methods" (plural). → **Mitigation**: Case-insensitive matching. For plural mismatches, the section won't be found and the tool returns available sections.
- **[Risk] Filename boost false positives**: Files like `class_control.rst` get boosted for query "control" — correct. But files with "control" in unrelated parts of the filename could also get boosted. → **Mitigation**: The boost only applies to tokens extracted from the filename stem, which follows a strict pattern (`class_<name>.rst`).

## Open Questions

None.
