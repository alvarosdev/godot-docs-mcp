## Purpose

Provides structured class metadata to MCP clients by extracting class name, inheritance, summary, member counts, and member lists from GFM documentation content.

## ADDED Requirements

### Requirement: Structured class metadata extraction

The server SHALL extract structured metadata from a class's GFM documentation content: the class name, the inheritance chain, a brief summary, counts of each member type, and the list of members (methods, properties, signals). Extraction SHALL be deterministic and based on the GFM structure (headings, bold inheritance lines, and markdown tables).

#### Scenario: Inheritance chain extracted
- **WHEN** a class's GFM content has `**Inherits:** CanvasItem < Node < Object`
- **THEN** the extracted `inherits` field is `CanvasItem < Node < Object`

#### Scenario: Summary extracted
- **WHEN** a class's GFM content has a description paragraph after the `## Description` heading
- **THEN** the extracted `summary` field contains that paragraph

#### Scenario: Member lists from markdown tables
- **WHEN** a class's GFM content has `## Methods` followed by a markdown table with rows like `| void | get_contact_body() const |`
- **THEN** each table row is extracted as a member with its type and name

#### Scenario: Member counts match member lists
- **WHEN** a class's GFM content has 8 method rows in the `## Methods` table
- **THEN** the `counts.methods` value is 8
- **AND** the `methods` list has exactly 8 entries

### Requirement: Overview resilience to format variation

The overview extraction SHALL tolerate imperfect GFM structure: if a member table row has an unexpected shape, that row SHALL be skipped (not crash the extraction). If a section (e.g., `## Signals`) is absent, its count SHALL be zero and its list empty rather than producing an error.

#### Scenario: Malformed table row skipped
- **WHEN** a member table contains a row that does not match the expected `| type | name |` shape
- **THEN** the row is skipped and does not appear in the member list
- **AND** extraction continues with subsequent rows

#### Scenario: Absent section produces zero count
- **WHEN** a class has no signals section in its GFM content
- **THEN** `counts.signals` is 0 and `signals` is an empty list, with no error raised

### Requirement: Overview is bounded and fast

The overview extraction SHALL operate on the in-memory GFM content of a single class file without external services or filesystem access, completing in bounded time proportional to the file size.

#### Scenario: Extraction over large class
- **WHEN** `get_class_overview` is called for a large class file (e.g., 150KB of GFM)
- **THEN** extraction completes within milliseconds
- **AND** no network or filesystem I/O occurs beyond the already-loaded doc store
