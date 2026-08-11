## MODIFIED Requirements

### Requirement: Documentation file retrieval by path

The server SHALL expose a `get_documentation_file` tool that returns the full content of a documentation file for a given Godot version. Documentation files SHALL be served as GitHub-Flavored Markdown (`.md`). An optional `section` parameter SHALL allow requesting only a specific GFM section by name. When the file exceeds 50,000 characters and no `section` is specified, the response SHALL include a warning suggesting use of the `section` parameter.

#### Scenario: Retrieve class documentation
- **WHEN** an MCP client calls `get_documentation_file` with `file_path="classes/class_camera2d.md"` and `version="4.7"`
- **THEN** the server returns the full content of the file at `docs/4.7/classes/class_camera2d.md`

#### Scenario: Default to latest version
- **WHEN** an MCP client calls `get_documentation_file` with `file_path="classes/class_node2d.md"` and no `version` argument
- **THEN** the server returns the content from the latest available version

#### Scenario: File not found
- **WHEN** an MCP client calls `get_documentation_file` with a `file_path` that does not exist in the specified version
- **THEN** the server returns an error message indicating the file was not found and listing available versions

#### Scenario: Request specific section
- **WHEN** an MCP client calls `get_documentation_file` with `file_path="classes/class_control.md"` and `section="Methods"`
- **THEN** the server returns only the Methods section content, not the entire file

#### Scenario: Request non-existent section
- **WHEN** an MCP client requests a section name that does not exist in the file
- **THEN** the server returns an error listing the available section names found in the file

#### Scenario: Section matching is case-insensitive
- **WHEN** an MCP client requests `section="methods"` and the file has a `## Methods` heading
- **THEN** the server SHALL match the section regardless of case

#### Scenario: Large file warning
- **WHEN** an MCP client calls `get_documentation_file` for a file exceeding 50,000 characters WITHOUT specifying a `section`
- **THEN** the response SHALL include a note indicating the file size in characters and suggesting the `section` parameter

#### Scenario: RST fallback is served when MD is absent
- **WHEN** a documentation file has no `.md` conversion (conversion failed) but a `.rst` source exists
- **THEN** the server SHALL return the `.rst` content as a fallback, allowing the LLM to read it natively

### Requirement: Class overview retrieval

The server SHALL expose a `get_class_overview` tool that returns structured JSON metadata about a class: class name, inheritance chain, brief summary, member counts, and member lists (methods, properties, signals).

#### Scenario: Retrieve class overview
- **WHEN** an MCP client calls `get_class_overview` with `class="Control"` and `version="4.7"`
- **THEN** the server returns JSON containing `class`, `inherits`, `summary`, `counts`, and `members`
- **AND** the JSON is structured (machine-readable), not raw text

#### Scenario: Class not found
- **WHEN** an MCP client calls `get_class_overview` for a class that does not exist
- **THEN** the server returns an error message indicating the class was not found

#### Scenario: Version omitted
- **WHEN** an MCP client calls `get_class_overview` without a `version` argument
- **THEN** the server resolves the latest version and returns the overview from it

#### Scenario: Member counts are accurate
- **WHEN** an MCP client calls `get_class_overview` for a class with 85 methods, 58 properties, and 8 signals
- **THEN** the returned `counts` object reports `methods: 85`, `properties: 58`, `signals: 8`
