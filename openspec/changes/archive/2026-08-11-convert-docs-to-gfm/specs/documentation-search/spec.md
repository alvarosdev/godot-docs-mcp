## MODIFIED Requirements

### Requirement: Token-based full-text search

The server SHALL expose a `search_documentation` tool that accepts a query string and returns a ranked list of matching documentation files with scores and text snippets. Search operates over the GFM (`.md`) documentation content, with `.rst` fallback files also searchable. The ranking SHALL be determined by token coverage weighted by heading matches (3x), section type boost (2x/1.5x/1x), match proximity (0.5-1.5x), and filename match boost (10x).

#### Scenario: Single keyword search
- **WHEN** an MCP client calls `search_documentation` with `query="CharacterBody2D"`
- **THEN** the server returns results ranked by score, where `classes/class_characterbody2d.md` is the highest-ranked result
- **AND** each result includes `path`, `version`, `score`, and `snippet`

#### Scenario: Search returns MD paths
- **WHEN** an MCP client searches and a GFM-converted file matches
- **THEN** the result `path` uses the `.md` extension
- **AND** when only a `.rst` fallback exists, the result `path` uses the `.rst` extension

#### Scenario: Filename match boost for class names
- **WHEN** a query contains a token matching a file's class name (e.g., "Control" matches `class_control.md`)
- **THEN** that file SHALL receive a 10x score multiplier, ensuring it ranks above files that only mention the term incidentally

#### Scenario: No matches found
- **WHEN** an MCP client calls `search_documentation` with a query that matches no documents
- **THEN** the server returns an empty result list with a message indicating no matches were found
