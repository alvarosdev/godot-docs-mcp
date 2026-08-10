// Package resources provides MCP resources for Godot documentation files.
package resources

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yosida95/uritemplate/v3"

	"github.com/alvarosdev/godot-docs-mcp/internal/docs"
)

// docResourcePattern matches the resource template "file:///{+file_path}".
// The + modifier (reserved expansion, RFC 6570) allows file_path to contain
// slashes, enabling version-prefixed paths like "4.7/classes/class_camera2d.md".
var docResourcePattern = uritemplate.MustNew("file:///{+file_path}")

// RegisterResources registers documentation resources on the MCP server.
func RegisterResources(srv *mcp.Server, store docs.DocStoreReader, docsRoot string) {
	handler := makeDocResourceHandler(store, docsRoot)
	srv.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "file:///{+file_path}",
		Name:        "Godot documentation file",
		Description: "Retrieve a Godot documentation file. Optionally prefix the path with a version (e.g., '4.7/classes/class_camera2d.md'). Without a version prefix, the latest version is used.",
		MIMEType:    "text/markdown",
	}, handler)
}

// makeDocResourceHandler creates a ResourceHandler that extracts the
// file_path from the matched URI template and serves documentation content.
func makeDocResourceHandler(store docs.DocStoreReader, docsRoot string) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		uri := req.Params.URI

		// Extract the file_path variable from the URI template match.
		match := docResourcePattern.Regexp().FindStringSubmatch(uri)
		if len(match) < 2 {
			return nil, mcp.ResourceNotFoundError(uri)
		}
		filePath := match[1]

		// Validate the extracted path before any use (defense in depth).
		cleanPath, err := docs.ValidateDocPath(filePath)
		if err != nil {
			return nil, mcp.ResourceNotFoundError(uri)
		}

		// Parse optional version prefix from the path.
		// "4.7/classes/class_camera2d.md" → version="4.7", path="classes/class_camera2d.md"
		// "classes/class_camera2d.md"      → version="" (use latest), path unchanged
		version, docPath := parseVersionPrefix(cleanPath)

		// If no version prefix, resolve from version metadata.
		if version == "" {
			meta, err := docs.LoadVersionMetadata(docsRoot)
			if err != nil {
				return nil, mcp.ResourceNotFoundError(uri)
			}
			version = meta.Latest
			docPath = cleanPath
		}

		// Look up in doc store.
		docKey := version + "/" + docPath
		content, ok := store.Get(docKey)
		if !ok {
			return nil, mcp.ResourceNotFoundError(uri)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					MIMEType: "text/markdown",
					Text:     content,
				},
			},
		}, nil
	}
}

// parseVersionPrefix splits a path on the first "/" and checks if the
// first segment matches a "major.minor" version pattern.
// Returns (version, restOfPath). If no version prefix, returns ("", originalPath).
func parseVersionPrefix(path string) (version string, rest string) {
	idx := strings.Index(path, "/")
	if idx < 0 {
		return "", path
	}
	firstSegment := path[:idx]
	// Check "major.minor" pattern
	if len(firstSegment) >= 3 && strings.Count(firstSegment, ".") == 1 {
		// Simple heuristic: both parts must be digits
		parts := strings.SplitN(firstSegment, ".", 2)
		if isDigits(parts[0]) && isDigits(parts[1]) {
			return firstSegment, path[idx+1:]
		}
	}
	return "", path
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
