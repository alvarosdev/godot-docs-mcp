## Purpose

Enables full-text search over the in-memory Godot documentation store using zero-dependency token-based scoring with heading, section, proximity, and filename boosts, exposed as an MCP tool with optional category filtering.

## Requirements

### Requirement: Token-based full-text search

The server SHALL expose a `search_documentation` tool that accepts a query string and returns a ranked list of matching documentation files with scores and text snippets. The ranking SHALL be determined by token coverage weighted by heading matches (3x), section type boost (2x/1.5x/1x), match proximity (0.5-1.5x), and filename match boost (10x). An optional `category` parameter SHALL filter results by path prefix.

#### Scenario: Single keyword search
- **WHEN** an MCP client calls `search_documentation` with `query="CharacterBody2D"`
- **THEN** the server returns results ranked by score, where `classes/class_characterbody2d.md` is the highest-ranked result
- **AND** each result includes `path`, `version`, `score`, and `snippet`

#### Scenario: Multi-word query
- **WHEN** an MCP client calls `search_documentation` with `query="collision detection"`
- **THEN** documents containing both "collision" and "detection" score higher than those containing only one token
- **AND** documents where both tokens appear close together receive a proximity bonus

#### Scenario: Heading match boost
- **WHEN** a query token matches text inside a heading line
- **THEN** that match SHALL count 3x compared to a match in body text

#### Scenario: Filename match boost for class names
- **WHEN** a query contains a token matching a file's class name (e.g., "CharacterBody2D" matches `class_characterbody2d.md`)
- **THEN** that file SHALL receive a 10x score multiplier, ensuring it ranks above files that only mention the term incidentally

#### Scenario: No matches found
- **WHEN** an MCP client calls `search_documentation` with a query that matches no documents
- **THEN** the server returns an empty result list with a message indicating no matches were found

#### Scenario: Category filter prefixes the path
- **WHEN** `category="classes"` is specified
- **THEN** only files whose doc path starts with `classes/` are included in results

### Requirement: Version-filtered search

The server SHALL accept an optional `version` parameter on the `search_documentation` tool. When provided, search results SHALL be limited to documentation for the specified Godot version.

#### Scenario: Version filter applied
- **WHEN** an MCP client calls `search_documentation` with `query="Node2D"` and `version="3.6"`
- **THEN** only documents from version 3.6 are returned in the results

#### Scenario: No version filter
- **WHEN** an MCP client calls `search_documentation` with `query="Node2D"` and no `version` parameter
- **THEN** documents from all available versions are included, with the version field identifying which version each result belongs to

### Requirement: Configurable result limit

The server SHALL accept an optional `limit` parameter on the `search_documentation` tool, defaulting to 10 and capped at 50.

#### Scenario: Default limit
- **WHEN** an MCP client calls `search_documentation` with `query="physics"` and no `limit`
- **THEN** at most 10 results are returned

#### Scenario: Custom limit
- **WHEN** an MCP client calls `search_documentation` with `query="physics"` and `limit=25`
- **THEN** at most 25 results are returned

#### Scenario: Limit capped
- **WHEN** an MCP client calls `search_documentation` with `limit=100`
- **THEN** at most 50 results are returned (enforced maximum)

### Requirement: Snippet extraction

Each search result SHALL include a `snippet` field containing the surrounding context (+-100 characters) of the first or highest-density match of the query tokens. Markdown and RST formatting markers (headings, code fences, inline markup) SHALL be stripped from the snippet for readability.

#### Scenario: Snippet with surrounding context
- **WHEN** a search matches a document containing the query tokens
- **THEN** the `snippet` field shows the matched region with surrounding context
- **AND** the snippet does not contain raw markup markers

### Requirement: Zero external dependencies

The search engine SHALL use only Go standard library packages. No full-text index library (bleve, FTS5), no external service, and no disk-based index SHALL be required.

#### Scenario: Search runs with no external services
- **WHEN** the server starts and loads documentation into memory
- **THEN** the `search_documentation` tool is available without any network access, disk I/O beyond the initial doc load, or third-party library initialization
