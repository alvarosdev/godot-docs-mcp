// Package tools provides MCP tools for navigating Godot documentation.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/alvarosdev/godot-docs-mcp/internal/docs"
	"github.com/alvarosdev/godot-docs-mcp/internal/search"
)

// RegisterTools registers all navigation tools on the MCP server.
func RegisterTools(srv *mcp.Server, store docs.DocStoreReader, meta *docs.VersionMetadata, docsRoot string, searcher *search.Searcher) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_available_versions",
		Description: "Get the list of available Godot documentation versions and metadata. Returns a JSON string with versions, latest version, and build date. Use this tool first to discover which Godot versions are available.",
	}, makeGetAvailableVersions(meta))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_documentation_tree",
		Description: "Get a tree-style overview of the documentation folder for a specific version.",
	}, makeGetDocumentationTree(meta, store))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_documentation_file",
		Description: "Get the content of a specific documentation file for a given version.",
	}, makeGetDocumentationFile(meta, store, docsRoot))

	if searcher != nil {
		mcp.AddTool(srv, &mcp.Tool{
			Name:        "search_documentation",
			Description: "Search Godot documentation by keyword or phrase. Returns ranked results with file paths, scores, and text snippets showing matching content. Use this when you don't know the exact file path.",
		}, makeSearchDocumentation(searcher))
	}

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_class_overview",
		Description: "Get structured metadata for a Godot class: inheritance, summary, member counts, and member lists (methods, properties, signals). Returns JSON.",
	}, makeGetClassOverview(store, meta))
}

// makeGetAvailableVersions returns the handler for get_available_versions.
func makeGetAvailableVersions(meta *docs.VersionMetadata) mcp.ToolHandlerFor[struct{}, struct{}] {
	return func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, struct{}, error) {
		if meta == nil {
			return errorResult("No version metadata available. Documentation may not be loaded."), struct{}{}, nil
		}
		data, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			return errorResult(fmt.Sprintf("Error encoding version metadata: %v", err)), struct{}{}, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, struct{}{}, nil
	}
}

type getDocTreeArgs struct {
	Version string `json:"version,omitempty" jsonschema:"Godot version string (e.g. '4.7', '3.6'). If omitted, defaults to the latest available version."`
}

// makeGetDocumentationTree returns the handler for get_documentation_tree.
func makeGetDocumentationTree(meta *docs.VersionMetadata, store docs.DocStoreReader) mcp.ToolHandlerFor[getDocTreeArgs, struct{}] {
	return func(ctx context.Context, req *mcp.CallToolRequest, args getDocTreeArgs) (*mcp.CallToolResult, struct{}, error) {
		version := args.Version
		if version == "" {
			if meta == nil {
				return errorResult("No version metadata available and no version specified."), struct{}{}, nil
			}
			version = meta.Latest
		}

		treePath := version + "/docs_tree.txt"
		content, ok := store.Get(treePath)
		if !ok {
			return errorResult(fmt.Sprintf(
				"Documentation tree not found for version %s. Use get_available_versions() to see available versions.", version)), struct{}{}, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: content}},
		}, struct{}{}, nil
	}
}

type getDocFileArgs struct {
	FilePath string `json:"file_path" jsonschema:"Path to the documentation file relative to the version docs directory (e.g. 'classes/class_camera2d.md'). Fallback to .rst is available for files that failed GFM conversion."`
	Version  string `json:"version,omitempty" jsonschema:"Godot version string (e.g. '4.7', '3.6'). If omitted, defaults to the latest available version."`
	Section  string `json:"section,omitempty" jsonschema:"GFM section name to extract (e.g. 'Methods', 'Properties', 'Signals'). Case-insensitive. Omit to return the full file."`
}

// makeGetDocumentationFile returns the handler for get_documentation_file.
func makeGetDocumentationFile(meta *docs.VersionMetadata, store docs.DocStoreReader, docsRoot string) mcp.ToolHandlerFor[getDocFileArgs, struct{}] {
	return func(ctx context.Context, req *mcp.CallToolRequest, args getDocFileArgs) (*mcp.CallToolResult, struct{}, error) {
		version := args.Version
		if version == "" {
			if meta == nil {
				return errorResult("No version metadata available and no version specified."), struct{}{}, nil
			}
			version = meta.Latest
		}

		// Validate the path (prevents traversal attacks).
		cleanPath, err := docs.ValidateDocPath(args.FilePath)
		if err != nil {
			return errorResult(fmt.Sprintf("Invalid path: %v", err)), struct{}{}, nil
		}

		// Look up in the doc store using versioned relative path.
		// Error messages use relative paths only — never absolute filesystem paths.
		docKey := version + "/" + cleanPath
		content, ok := store.Get(docKey)
		if !ok {
			available := ""
			if meta != nil {
				available = fmt.Sprintf(" Available versions: %v.", meta.Versions)
			}
			return errorResult(fmt.Sprintf(
				"Documentation file not found: %s (version: %s).%s Use get_documentation_tree(version) to see available paths.",
				args.FilePath, version, available)), struct{}{}, nil
		}

		// Section extraction.
		if args.Section != "" {
			sectionContent, found := findSection(content, args.Section)
			if !found {
				sections := listSections(content)
				return errorResult(fmt.Sprintf(
					"Section %q not found. Available sections: %v.",
					args.Section, sections)), struct{}{}, nil
			}
			content = sectionContent
		}

		// Size warning for large files.
		const sizeWarnThreshold = 50_000
		resultContent := content
		if args.Section == "" && len(content) > sizeWarnThreshold {
			resultContent += fmt.Sprintf(
				"\n\n[Note: This file has %d characters. Use the 'section' parameter to request specific parts (e.g. section='Methods', section='Properties').]",
				len(content))
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: resultContent}},
		}, struct{}{}, nil
	}
}

type searchDocArgs struct {
	Query    string `json:"query" jsonschema:"Search query. Use keywords or short phrases. Supports multiple words."`
	Version  string `json:"version,omitempty" jsonschema:"Filter to a specific Godot version (e.g. '4.7'). Omit to search all versions."`
	Category string `json:"category,omitempty" jsonschema:"Filter by top-level directory: 'classes', 'tutorials', etc. Omit to search all."`
	Limit    int    `json:"limit,omitempty" jsonschema:"Max results (default 10, max 50)."`
}

func makeSearchDocumentation(searcher *search.Searcher) mcp.ToolHandlerFor[searchDocArgs, struct{}] {
	return func(ctx context.Context, req *mcp.CallToolRequest, args searchDocArgs) (*mcp.CallToolResult, struct{}, error) {
		if args.Query == "" {
			return errorResult("Query is required. Provide a search term or phrase."), struct{}{}, nil
		}
		if searcher == nil {
			return errorResult("Search is not available. Documentation may not be loaded."), struct{}{}, nil
		}

		results := searcher.Search(args.Query, args.Version, args.Category, args.Limit)
		if len(results) == 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{
					Text: "No documentation found matching your query. Try different keywords or use get_documentation_tree to browse available files.",
				}},
			}, struct{}{}, nil
		}

		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return errorResult("Error formatting search results."), struct{}{}, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, struct{}{}, nil
	}
}

// ─── GFM section parsing ──────────────────────────────────────────────────

var gfmHeadingRe = regexp.MustCompile(`^#{1,6}\s+(.+)$`)

// findSection locates a GFM section by heading name and returns its content.
// GFM headings are ATX style: "## Methods", "### Method Descriptions".
// Matching is case-insensitive. Returns the content from after the heading
// line until the next heading of equal-or-higher level (fewer #'s) or EOF.
func findSection(content, name string) (string, bool) {
	lowerName := strings.ToLower(strings.TrimSpace(name))
	if lowerName == "" {
		return "", false
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		m := gfmHeadingRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		text := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(m[1]), "#"))
		if strings.ToLower(text) != lowerName {
			continue
		}
		level := headingLevel(line)
		start := i + 1
		end := len(lines)
		for j := start; j < len(lines); j++ {
			trimmed := strings.TrimSpace(lines[j])
			if mm := gfmHeadingRe.FindStringSubmatch(trimmed); mm != nil {
				if headingLevel(trimmed) <= level {
					end = j
					break
				}
			}
		}
		return strings.Join(lines[start:end], "\n"), true
	}
	return "", false
}

// listSections returns all GFM section headings (level >=2) found in the content.
// H1 is the class title and is excluded from the section list.
func listSections(content string) []string {
	var sections []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		m := gfmHeadingRe.FindStringSubmatch(trimmed)
		if m == nil {
			continue
		}
		if headingLevel(trimmed) < 2 {
			continue
		}
		text := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(m[1]), "#"))
		if text != "" {
			sections = append(sections, text)
		}
	}
	return sections
}

func headingLevel(line string) int {
	count := 0
	for _, c := range strings.TrimSpace(line) {
		if c == '#' {
			count++
		} else {
			break
		}
	}
	return count
}

// ─── get_class_overview ─────────────────────────────────────────────────

type getClassOverviewArgs struct {
	Class   string `json:"class" jsonschema:"Class name e.g. Control, Node2D"`
	Version string `json:"version,omitempty" jsonschema:"Godot version string (e.g. '4.7'). If omitted, defaults to the latest available version."`
}

func makeGetClassOverview(store docs.DocStoreReader, meta *docs.VersionMetadata) mcp.ToolHandlerFor[getClassOverviewArgs, struct{}] {
	return func(ctx context.Context, req *mcp.CallToolRequest, args getClassOverviewArgs) (*mcp.CallToolResult, struct{}, error) {
		if args.Class == "" {
			return errorResult("Class name is required."), struct{}{}, nil
		}
		version := args.Version
		if version == "" {
			if meta == nil {
				return errorResult("No version metadata available and no version specified."), struct{}{}, nil
			}
			version = meta.Latest
		}
		// Try .md first, then .rst fallback.
		for _, ext := range []string{".md", ".rst"} {
			key := fmt.Sprintf("%s/classes/class_%s%s", version, strings.ToLower(args.Class), ext)
			content, ok := store.Get(key)
			if !ok {
				continue
			}
			ov, err := extractOverview(content)
			if err != nil {
				return errorResult(fmt.Sprintf("Error extracting overview: %v", err)), struct{}{}, nil
			}
			ov.Class = args.Class
			data, err := json.MarshalIndent(ov, "", "  ")
			if err != nil {
				return errorResult(fmt.Sprintf("Error formatting overview: %v", err)), struct{}{}, nil
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
			}, struct{}{}, nil
		}
		return errorResult(fmt.Sprintf("Class %q not found (version: %s). Use search_documentation or get_documentation_tree to discover available classes.", args.Class, version)), struct{}{}, nil
	}
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}
}
