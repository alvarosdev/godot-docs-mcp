## MODIFIED Requirements

### Requirement: Documentation file retrieval by path

The server SHALL expose a `get_documentation_file` tool that returns the full content of a documentation file for a given Godot version. An optional `section` parameter SHALL allow requesting only a specific RST section by name. When the file exceeds 50,000 characters and no `section` is specified, the response SHALL include a warning suggesting use of the `section` parameter.

#### Scenario: Retrieve class documentation
- **WHEN** an MCP client calls `get_documentation_file` with `file_path="classes/class_camera2d.rst"` and `version="4.7"`
- **THEN** the server returns the full content of the file at `docs/4.7/classes/class_camera2d.rst`

#### Scenario: Default to latest version
- **WHEN** an MCP client calls `get_documentation_file` with `file_path="classes/class_node2d.rst"` and no `version` argument
- **THEN** the server returns the content from the latest available version

#### Scenario: File not found
- **WHEN** an MCP client calls `get_documentation_file` with a `file_path` that does not exist in the specified version
- **THEN** the server returns an error message indicating the file was not found and listing available versions

#### Scenario: Request specific section
- **WHEN** an MCP client calls `get_documentation_file` with `file_path="classes/class_control.rst"` and `section="Methods"`
- **THEN** the server returns only the Methods section content (heading + table + description), not the entire file

#### Scenario: Request non-existent section
- **WHEN** an MCP client requests a section name that does not exist in the file
- **THEN** the server returns an error listing the available section names found in the file

#### Scenario: Section matching is case-insensitive
- **WHEN** an MCP client requests `section="methods"` and the file has a `Methods` heading
- **THEN** the server SHALL match the section regardless of case

#### Scenario: Large file warning
- **WHEN** an MCP client calls `get_documentation_file` for a file exceeding 50,000 characters WITHOUT specifying a `section`
- **THEN** the response SHALL include a note indicating the file size in characters and suggesting the `section` parameter

### Requirement: Documentation search tool

The server SHALL expose a `search_documentation` tool that performs token-based full-text search across the loaded documentation and returns ranked results with file paths, versions, scores, and text snippets. An optional `category` parameter SHALL allow filtering results to a specific top-level directory (`classes`, `tutorials`, etc.).

#### Scenario: Search discovers docs without knowing file path
- **WHEN** an MCP client calls `search_documentation` with `query="move_and_slide"`
- **THEN** the server returns results including classes documentation containing that function without the client needing to know the exact file path

#### Scenario: Search complements existing navigation tools
- **WHEN** a user asks a question about a Godot concept without specifying a class name
- **THEN** the LLM SHALL be able to use `search_documentation` to discover relevant files, then use `get_documentation_file` to read the full content of the most promising result

#### Scenario: Category filter restricts results
- **WHEN** an MCP client calls `search_documentation` with `query="collision"` and `category="classes"`
- **THEN** only files whose path starts with `classes/` are returned, excluding tutorials and other sections

#### Scenario: Category filter omitted
- **WHEN** an MCP client calls `search_documentation` with no `category` parameter
- **THEN** all files are searched regardless of path prefix
