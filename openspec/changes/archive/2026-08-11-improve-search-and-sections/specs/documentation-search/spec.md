## MODIFIED Requirements

### Requirement: Token-based full-text search

The server SHALL expose a `search_documentation` tool that accepts a query string and returns a ranked list of matching documentation files with scores and text snippets. The ranking SHALL be determined by token coverage weighted by heading matches (3×), section type boost (2×/1.5×/1×), match proximity (0.5–1.5×), and filename match boost (10×). An optional `category` parameter SHALL filter results by path prefix.

#### Scenario: Single keyword search
- **WHEN** an MCP client calls `search_documentation` with `query="CharacterBody2D"`
- **THEN** the server returns results ranked by score, where `classes/class_characterbody2d.rst` is the highest-ranked result
- **AND** each result includes `path`, `version`, `score`, and `snippet`

#### Scenario: Multi-word query
- **WHEN** an MCP client calls `search_documentation` with `query="collision detection"`
- **THEN** documents containing both "collision" and "detection" score higher than those containing only one token
- **AND** documents where both tokens appear close together receive a proximity bonus

#### Scenario: Heading match boost
- **WHEN** a query token matches text inside a heading line
- **THEN** that match SHALL count 3× compared to a match in body text

#### Scenario: Filename match boost for class names
- **WHEN** a query contains a token matching a file's class name (e.g., "CharacterBody2D" matches `class_characterbody2d.rst`)
- **THEN** that file SHALL receive a 10× score multiplier, ensuring it ranks above files that only mention the term incidentally

#### Scenario: No matches found
- **WHEN** an MCP client calls `search_documentation` with a query that matches no documents
- **THEN** the server returns an empty result list with a message indicating no matches were found

#### Scenario: Category filter prefixes the path
- **WHEN** `category="classes"` is specified
- **THEN** only files whose doc path starts with `classes/` are included in results
