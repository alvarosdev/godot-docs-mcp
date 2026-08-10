## Purpose

Exposes Godot documentation through MCP tools that are aware of multiple documentation versions, allowing clients to discover available versions and query docs for a specific version.

## ADDED Requirements

### Requirement: get_available_versions returns version metadata
The server SHALL expose a `get_available_versions` tool that returns the list of available Godot documentation versions, which version is marked as latest, and the build date.

#### Scenario: Multiple versions available
- **WHEN** the server has docs for versions 3.6 and 4.7
- **THEN** `get_available_versions()` returns `{"versions": ["3.6", "4.7"], "latest": "4.7", "built": "2026.08.03"}`

#### Scenario: No docs directory present
- **WHEN** the docs directory is missing or empty
- **THEN** `get_available_versions()` returns an error message indicating no documentation is available

### Requirement: get_documentation_tree accepts optional version parameter
The `get_documentation_tree` tool SHALL accept an optional `version` parameter. When a version is provided, it returns the documentation tree for that version. When omitted, it defaults to the latest version.

#### Scenario: Version provided
- **WHEN** `get_documentation_tree(version="3.6")` is called
- **THEN** the content of `docs/3.6/docs_tree.txt` is returned

#### Scenario: Version omitted (default to latest)
- **WHEN** `get_documentation_tree()` is called without a version argument and the latest version is 4.7
- **THEN** the content of `docs/4.7/docs_tree.txt` is returned

#### Scenario: Requested version not found
- **WHEN** `get_documentation_tree(version="2.0")` is called and version 2.0 is not available
- **THEN** an error message is returned listing the available versions

### Requirement: get_documentation_file accepts version parameter
The `get_documentation_file` tool SHALL accept a `version` parameter in addition to the existing `file_path` parameter. When `version` is omitted, it defaults to the latest version.

#### Scenario: File retrieved for specific version
- **WHEN** `get_documentation_file(file_path="classes/class_camera2d.md", version="3.6")` is called
- **THEN** the content of `docs/3.6/classes/class_camera2d.md` is returned

#### Scenario: Version omitted (default to latest)
- **WHEN** `get_documentation_file(file_path="classes/class_camera2d.md")` is called without a version argument
- **THEN** the file is read from the latest version's directory

#### Scenario: File not found in specified version
- **WHEN** `get_documentation_file(file_path="classes/class_nonexistent.md", version="4.7")` is called
- **THEN** an error message is returned indicating the file was not found, suggesting use of `get_documentation_tree` to browse available paths

#### Scenario: Path traversal attempt is blocked
- **WHEN** `get_documentation_file(file_path="../../../etc/passwd", version="4.7")` is called
- **THEN** an access-denied error is returned without reading the file

### Requirement: Resources include version as path prefix
The `file://` resource SHALL accept paths with a version prefix (`{version}/{doc_path}`) so that resource URIs are version-aware without changing the resource URI scheme.

#### Scenario: Versioned resource path
- **WHEN** resource `file://3.6/classes/class_camera2d.md` is requested
- **THEN** the content of `docs/3.6/classes/class_camera2d.md` is returned

#### Scenario: Backward compatibility without version prefix
- **WHEN** resource `file://classes/class_camera2d.md` is requested without a version prefix
- **THEN** the file is served from the latest version's directory
