## ADDED Requirements

### Requirement: Documentation search tool

The server SHALL expose a `search_documentation` tool that performs token-based full-text search across the loaded documentation and returns ranked results with file paths, versions, scores, and text snippets.

#### Scenario: Search discovers docs without knowing file path
- **WHEN** an MCP client calls `search_documentation` with `query="move_and_slide"`
- **THEN** the server returns results including `classes/class_characterbody2d.md` (if it contains that function) without the client needing to know the file path

#### Scenario: Search complements existing navigation tools
- **WHEN** a user asks a question about a Godot concept without specifying a class name
- **THEN** the LLM SHALL be able to use `search_documentation` to discover relevant files, then use `get_documentation_file` to read the full content of the most promising result
