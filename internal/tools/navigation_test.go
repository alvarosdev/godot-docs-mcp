package tools

import (
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alvarosdev/godot-docs-mcp/internal/docs"
	"github.com/alvarosdev/godot-docs-mcp/internal/search"
)

func TestGetAvailableVersions_Success(t *testing.T) {
	meta := &docs.VersionMetadata{
		Versions: []string{"3.6", "4.7"},
		Latest:   "4.7",
		Built:    "2026.08.10",
	}
	handler := makeGetAvailableVersions(meta)
	result, _, err := handler(nil, nil, struct{}{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	content := result.Content[0].(*mcp.TextContent).Text
	var parsed docs.VersionMetadata
	require.NoError(t, json.Unmarshal([]byte(content), &parsed))
	assert.Equal(t, "4.7", parsed.Latest)
	assert.Equal(t, []string{"3.6", "4.7"}, parsed.Versions)
}

func TestGetAvailableVersions_NilMeta(t *testing.T) {
	handler := makeGetAvailableVersions(nil)
	result, _, err := handler(nil, nil, struct{}{})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestGetDocumentationTree_Latest(t *testing.T) {
	store := docs.NewMockDocStoreReader(t)
	meta := &docs.VersionMetadata{Latest: "4.7", Versions: []string{"3.6", "4.7"}}

	store.On("Get", "4.7/docs_tree.txt").Return("├── classes\n│   └── class_node.md\n", true)

	handler := makeGetDocumentationTree(meta, store)
	result, _, err := handler(nil, nil, getDocTreeArgs{})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].(*mcp.TextContent).Text, "classes")
}

func TestGetDocumentationTree_SpecificVersion(t *testing.T) {
	store := docs.NewMockDocStoreReader(t)
	meta := &docs.VersionMetadata{Latest: "4.7", Versions: []string{"3.6", "4.7"}}

	store.On("Get", "3.6/docs_tree.txt").Return("├── classes\n│   └── class_node.md\n", true)

	handler := makeGetDocumentationTree(meta, store)
	result, _, err := handler(nil, nil, getDocTreeArgs{Version: "3.6"})
	require.NoError(t, err)
	assert.False(t, result.IsError)
}

func TestGetDocumentationTree_VersionNotFound(t *testing.T) {
	store := docs.NewMockDocStoreReader(t)
	meta := &docs.VersionMetadata{Latest: "4.7", Versions: []string{"3.6", "4.7"}}

	store.On("Get", "4.7/docs_tree.txt").Return("", false)

	handler := makeGetDocumentationTree(meta, store)
	result, _, err := handler(nil, nil, getDocTreeArgs{})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestGetDocumentationFile_Found(t *testing.T) {
	store := docs.NewMockDocStoreReader(t)
	meta := &docs.VersionMetadata{Latest: "4.7", Versions: []string{"4.7"}}

	store.On("Get", "4.7/classes/class_node.md").Return("# Node\n\nBase class.", true)

	handler := makeGetDocumentationFile(meta, store, "/tmp/docs")
	result, _, err := handler(nil, nil, getDocFileArgs{
		FilePath: "classes/class_node.md",
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].(*mcp.TextContent).Text, "# Node")
}

func TestGetDocumentationFile_InvalidPath(t *testing.T) {
	store := docs.NewMockDocStoreReader(t)
	meta := &docs.VersionMetadata{Latest: "4.7", Versions: []string{"4.7"}}

	handler := makeGetDocumentationFile(meta, store, "/tmp/docs")
	result, _, err := handler(nil, nil, getDocFileArgs{
		FilePath: "../../etc/passwd",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(*mcp.TextContent).Text, "Invalid path")
}

func TestGetDocumentationFile_NotFound(t *testing.T) {
	store := docs.NewMockDocStoreReader(t)
	meta := &docs.VersionMetadata{Latest: "4.7", Versions: []string{"3.6", "4.7"}}

	store.On("Get", "4.7/classes/class_missing.md").Return("", false)

	handler := makeGetDocumentationFile(meta, store, "/tmp/docs")
	result, _, err := handler(nil, nil, getDocFileArgs{
		FilePath: "classes/class_missing.md",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(*mcp.TextContent).Text, "not found")
}

// ─── search_documentation tests ──────────────────────────────────────────

func TestSearchDocumentation_Success(t *testing.T) {
	store := docs.NewDocStore()
	searcher := search.New(store, []string{"4.7"})

	// Searcher needs real docs — skip if store is empty.
	handler := makeSearchDocumentation(searcher)
	result, _, err := handler(nil, nil, searchDocArgs{
		Query: "nonexistent_xyz",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
	// Should return a valid response (empty results message, not error)
}

func TestSearchDocumentation_EmptyQuery(t *testing.T) {
	handler := makeSearchDocumentation(nil)
	result, _, err := handler(nil, nil, searchDocArgs{
		Query: "",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestSearchDocumentation_NilSearcher(t *testing.T) {
	handler := makeSearchDocumentation(nil)
	result, _, err := handler(nil, nil, searchDocArgs{
		Query: "test",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(*mcp.TextContent).Text, "not available")
}

// ─── section parsing edge cases ──────────────────────────────────────────

func TestFindSection_NoHeadings(t *testing.T) {
	content := "Just plain text without any headings. No markdown headings here."
	_, ok := findSection(content, "Methods")
	assert.False(t, ok, "should not find section when no headings exist")
	assert.Empty(t, listSections(content), "listSections should return empty for content with no headings")
}

func TestFindSection_WhitespaceHeading(t *testing.T) {
	content := "# Control\n\n   ## Methods   \n\ncontent here\n\n## Signals\n\nsignals here"
	section, ok := findSection(content, "Methods")
	require.True(t, ok, "heading with leading/trailing whitespace should be matched")
	assert.Contains(t, section, "content here")
	assert.NotContains(t, section, "signals here")
}

func TestFindSection_WhitespaceQuery(t *testing.T) {
	content := "# Control\n\n## Methods\n\nmethod body\n\n## Signals\n\n"
	section, ok := findSection(content, "  Methods  ")
	require.True(t, ok, "query with whitespace should be trimmed and matched")
	assert.Contains(t, section, "method body")
}
