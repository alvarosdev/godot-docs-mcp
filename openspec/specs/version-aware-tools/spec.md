## Purpose

Serve Godot Engine documentation to AI assistants through the Model Context Protocol, exposing version-aware tools and resources over stdio and streamable HTTP transports.

## ADDED Requirements

### Requirement: Server identity and instructions

The server SHALL identify itself as `godot-docs-server` to MCP clients and provide usage instructions via the MCP `instructions` field that describe how to use the available tools and resources for accessing Godot documentation.

#### Scenario: Client initialization
- **WHEN** an MCP client sends an `initialize` request
- **THEN** the server responds with `serverInfo.name` equal to `"godot-docs-server"`
- **AND** the response includes `instructions` describing how to use the documentation tools

### Requirement: Multi-version documentation discovery

The server SHALL expose a `get_available_versions` tool that returns a JSON object containing the list of available Godot versions, the latest version identifier, and the build date.

#### Scenario: Versions available
- **WHEN** an MCP client calls `get_available_versions`
- **THEN** the server returns a JSON object with keys `versions` (array of version strings), `latest` (string), and `built` (date string)
- **AND** versions are sorted in ascending order

#### Scenario: No versions.json file
- **WHEN** the `versions.json` metadata file is missing from the docs directory
- **THEN** the server SHALL scan `docs/` for directories matching the `major.minor` version pattern
- **AND** return the discovered versions with the highest version marked as `latest`

### Requirement: Documentation tree navigation

The server SHALL expose a `get_documentation_tree` tool that returns a tree-style directory listing of available documentation files for a specified Godot version.

#### Scenario: Latest version by default
- **WHEN** an MCP client calls `get_documentation_tree` without a `version` argument
- **THEN** the server returns the documentation tree for the latest available version

#### Scenario: Specific version
- **WHEN** an MCP client calls `get_documentation_tree` with `version="4.7"`
- **THEN** the server returns the documentation tree for Godot 4.7

#### Scenario: Unknown version
- **WHEN** an MCP client calls `get_documentation_tree` with a version string that does not exist
- **THEN** the server returns an error message indicating the version is not available

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
- **THEN** the server returns only the Methods section content, not the entire file

#### Scenario: Request non-existent section
- **WHEN** an MCP client requests a section name that does not exist in the file
- **THEN** the server returns an error listing the available section names found in the file

#### Scenario: Section matching is case-insensitive
- **WHEN** an MCP client requests `section="methods"` and the file has a `Methods` heading
- **THEN** the server SHALL match the section regardless of case

#### Scenario: Large file warning
- **WHEN** an MCP client calls `get_documentation_file` for a file exceeding 50,000 characters WITHOUT specifying a `section`
- **THEN** the response SHALL include a note indicating the file size in characters and suggesting the `section` parameter

### Requirement: Path traversal protection

The server SHALL reject documentation file paths that attempt to escape the docs directory. All user-supplied paths SHALL be validated with Go's `filepath.IsLocal()` before resolution, and file reads SHALL use `os.OpenRoot()` (or equivalent) to enforce kernel-level containment within the docs directory.

#### Scenario: Dot-dot traversal attempt
- **WHEN** an MCP client calls `get_documentation_file` with `file_path="../../etc/passwd"`
- **THEN** the server returns an access denied error without reading any file outside the docs directory

#### Scenario: Absolute path attempt
- **WHEN** an MCP client calls `get_documentation_file` with `file_path="/etc/passwd"`
- **THEN** the server returns an access denied error without reading any file outside the docs directory

#### Scenario: URL-encoded traversal attempt
- **WHEN** an MCP client calls `get_documentation_file` with `file_path="%2e%2e/%2e%2e/etc/passwd"`
- **THEN** the server returns an access denied error without reading any file outside the docs directory

#### Scenario: Symlink escape attempt
- **WHEN** a documentation directory contains a symlink pointing outside the docs root
- **THEN** the file open operation SHALL fail with an access denied error

### Requirement: Error messages use relative paths only

The server SHALL NOT expose absolute filesystem paths in any tool or resource error messages returned to MCP clients. All paths in error messages SHALL be relative to the docs directory.

#### Scenario: File not found error
- **WHEN** a documentation file is not found
- **THEN** the error message contains the relative path (e.g. `classes/class_camera2d.md`) and NOT the absolute path (e.g. `/app/docs/4.7/classes/class_camera2d.md`)

### Requirement: Resource URI for documentation files

The server SHALL expose a resource template at `file:///{+file_path}` that returns the content of the matching documentation file, with optional version prefix in the path.

#### Scenario: Resource with version prefix
- **WHEN** an MCP client reads resource `file:///4.7/classes/class_camera2d.md`
- **THEN** the server returns the content of `docs/4.7/classes/class_camera2d.md`

#### Scenario: Resource without version prefix
- **WHEN** an MCP client reads resource `file:///classes/class_camera2d.md`
- **THEN** the server returns the content from the latest version

#### Scenario: Resource not found
- **WHEN** an MCP client reads a resource URI that does not match any existing file
- **THEN** the server returns an MCP resource-not-found error

#### Scenario: Resource handler path validation
- **WHEN** an MCP client reads a resource URI where the captured `file_path` variable attempts directory traversal (e.g. `file:///../etc/passwd`)
- **THEN** the handler SHALL validate the extracted path with `filepath.IsLocal()` before resolving it
- **AND** return a resource-not-found error if validation fails

### Requirement: Streamable HTTP transport

The server SHALL support the MCP streamable HTTP transport via `mcp.NewStreamableHTTPHandler`, accepting POST and GET requests at a configurable path and host. The underlying `http.Server` SHALL set `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` to prevent slow-client resource exhaustion.

#### Scenario: HTTP server starts
- **WHEN** the server is started with transport mode set to streamable HTTP
- **THEN** it listens on the configured host and port with a non-zero `ReadTimeout` (10s), `WriteTimeout` (30s), and `IdleTimeout` (120s)
- **AND** accepts MCP protocol messages at the configured path

#### Scenario: Health check
- **WHEN** an HTTP GET request is made to the `/health` endpoint
- **THEN** the server responds with HTTP 200 and a JSON body containing `{"status": "healthy"}`

### Requirement: Session keepalive to prevent goroutine leaks

The server SHALL configure `KeepAlive` and `KeepAliveFailureThreshold` in `ServerOptions` to detect and clean up abandoned MCP sessions.

#### Scenario: Abandoned session cleanup
- **WHEN** an MCP client connects via streamable HTTP and disappears without closing the session
- **THEN** the server SHALL detect the dead session via keepalive ping failures within the configured threshold
- **AND** release associated goroutines and resources

### Requirement: Stdio transport

The server SHALL support the MCP stdio transport for local MCP clients.

#### Scenario: Stdio server starts
- **WHEN** the server is started with transport mode set to stdio
- **THEN** it reads MCP protocol messages from stdin and writes responses to stdout

### Requirement: Bearer token authentication

The server SHALL support optional bearer token authentication for HTTP transports. The authentication token SHALL be read from the `MCP_AUTH_TOKEN` environment variable once at startup and captured in a closure — never re-read from the environment on each request. When `MCP_AUTH_TOKEN` is set, the server SHALL require clients to include the token in the `Authorization: Bearer <token>` header.

#### Scenario: Valid token
- **WHEN** a client sends a request with `Authorization: Bearer <correct-token>`
- **THEN** the server processes the request normally

#### Scenario: Missing or invalid token
- **WHEN** a client sends a request without a valid bearer token and `MCP_AUTH_TOKEN` is configured
- **THEN** the server responds with HTTP 401 Unauthorized

#### Scenario: Auth not configured
- **WHEN** `MCP_AUTH_TOKEN` is not set or is empty
- **THEN** the server accepts all requests without authentication

### Requirement: Configuration via environment variables

The server SHALL read its runtime configuration from environment variables at startup (not on every request) with the same names as the current Python implementation.

#### Scenario: Default configuration
- **WHEN** no environment variables are set
- **THEN** the server uses HTTP transport on `127.0.0.1:8000` at path `/mcp` with no authentication

#### Scenario: Custom configuration
- **WHEN** `FASTMCP_HOST=0.0.0.0`, `FASTMCP_PORT=3000`, `MCP_AUTH_TOKEN=secret` are set
- **THEN** the server listens on `0.0.0.0:3000` with `ReadTimeout=10s`, `WriteTimeout=30s`, `IdleTimeout=120s`, and requires bearer token `secret`

### Requirement: Immutable documentation store

The documentation store (files loaded from `docs/` into memory) SHALL be populated once at startup and remain immutable for the lifetime of the process. No concurrent writes to the backing data structure SHALL occur after the initial load completes.

#### Scenario: Startup load and immutability
- **WHEN** the server starts and loads documentation into memory
- **THEN** all subsequent lookups SHALL read from the pre-loaded store without locks
- **AND** no reload, hot-swap, or mutation of the store occurs during normal operation

### Requirement: Documentation search tool

The server SHALL expose a `search_documentation` tool that performs token-based full-text search across the loaded documentation and returns ranked results with file paths, versions, scores, and text snippets. An optional `category` parameter SHALL allow filtering results to a specific top-level directory.

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

### Requirement: Class overview retrieval

The server SHALL expose a `get_class_overview` tool that returns structured JSON metadata about a class: class name, inheritance chain, brief summary, member counts, and member lists (methods, properties, signals).

#### Scenario: Retrieve class overview
- **WHEN** an MCP client calls `get_class_overview` with `class="Control"` and `version="4.7"`
- **THEN** the server returns JSON containing `class`, `inherits`, `summary`, `counts`, and `members`

#### Scenario: Class not found
- **WHEN** an MCP client calls `get_class_overview` for a class that does not exist
- **THEN** the server returns an error message indicating the class was not found
