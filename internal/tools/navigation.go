// Package tools provides MCP tools for navigating Godot documentation.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

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
	FilePath string `json:"file_path" jsonschema:"Path to the documentation file relative to the version docs directory (e.g. 'classes/class_camera2d.md')."`
	Version  string `json:"version,omitempty" jsonschema:"Godot version string (e.g. '4.7', '3.6'). If omitted, defaults to the latest available version."`
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

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: content}},
		}, struct{}{}, nil
	}
}

type searchDocArgs struct {
	Query   string `json:"query" jsonschema:"Search query. Use keywords or short phrases. Supports multiple words."`
	Version string `json:"version,omitempty" jsonschema:"Filter to a specific Godot version (e.g. '4.7'). Omit to search all versions."`
	Limit   int    `json:"limit,omitempty" jsonschema:"Max results (default 10, max 50)."`
}

func makeSearchDocumentation(searcher *search.Searcher) mcp.ToolHandlerFor[searchDocArgs, struct{}] {
	return func(ctx context.Context, req *mcp.CallToolRequest, args searchDocArgs) (*mcp.CallToolResult, struct{}, error) {
		if args.Query == "" {
			return errorResult("Query is required. Provide a search term or phrase."), struct{}{}, nil
		}
		if searcher == nil {
			return errorResult("Search is not available. Documentation may not be loaded."), struct{}{}, nil
		}

		results := searcher.Search(args.Query, args.Version, args.Limit)
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

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}
}
