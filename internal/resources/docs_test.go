package resources

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alvarosdev/godot-docs-mcp/internal/docs"
)

// ─── parseVersionPrefix tests (pure logic, no mock) ──────────────────────

func TestParseVersionPrefix_WithVersion(t *testing.T) {
	version, rest := parseVersionPrefix("4.7/classes/class_camera2d.md")
	assert.Equal(t, "4.7", version)
	assert.Equal(t, "classes/class_camera2d.md", rest)
}

func TestParseVersionPrefix_WithoutVersion(t *testing.T) {
	version, rest := parseVersionPrefix("classes/class_camera2d.md")
	assert.Empty(t, version)
	assert.Equal(t, "classes/class_camera2d.md", rest)
}

func TestParseVersionPrefix_DeepPath(t *testing.T) {
	version, rest := parseVersionPrefix("4.7/tutorials/2d/movement.md")
	assert.Equal(t, "4.7", version)
	assert.Equal(t, "tutorials/2d/movement.md", rest)
}

func TestParseVersionPrefix_NotAVersion(t *testing.T) {
	version, rest := parseVersionPrefix("abc/def/file.md")
	assert.Empty(t, version)
	assert.Equal(t, "abc/def/file.md", rest)
}

// ─── URI pattern tests ────────────────────────────────────────────────────

func TestDocResourcePattern_Matches(t *testing.T) {
	for _, uri := range []string{
		"file:///4.7/classes/class_camera2d.md",
		"file:///classes/class_node2d.md",
		"file:///3.6/tutorials/2d/movement.md",
		"file:///getting_started/introduction.md",
	} {
		assert.True(t, docResourcePattern.Regexp().MatchString(uri), "uri: %s", uri)
	}
}

func TestDocResourcePattern_ExtractPath(t *testing.T) {
	match := docResourcePattern.Regexp().FindStringSubmatch("file:///4.7/classes/class_camera2d.md")
	require.Len(t, match, 2)
	assert.Equal(t, "4.7/classes/class_camera2d.md", match[1])
}

func TestDocResourcePattern_NoTraversalMatch(t *testing.T) {
	match := docResourcePattern.Regexp().FindStringSubmatch("file:///../../../etc/passwd")
	if len(match) >= 2 {
		_, err := docs.ValidateDocPath(match[1])
		assert.Error(t, err, "traversal path should be rejected by ValidateDocPath")
	}
}

// ─── Resource handler tests with mock ─────────────────────────────────────

func TestResourceHandler_WithVersionPrefix(t *testing.T) {
	store := docs.NewMockDocStoreReader(t)
	handler := makeDocResourceHandler(store, "/tmp/docs")

	store.On("Get", "4.7/classes/class_node.md").Return("# Node\n\nContent.", true)

	result, err := handler(nil, &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "file:///4.7/classes/class_node.md"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Contents, 1)
	assert.Equal(t, "# Node\n\nContent.", result.Contents[0].Text)
}

func TestResourceHandler_NotFound(t *testing.T) {
	store := docs.NewMockDocStoreReader(t)
	handler := makeDocResourceHandler(store, "/tmp/docs")

	store.On("Get", "4.7/classes/missing.md").Return("", false)

	_, err := handler(nil, &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "file:///4.7/classes/missing.md"},
	})
	assert.Error(t, err)
}

func TestResourceHandler_TraversalPath(t *testing.T) {
	store := docs.NewMockDocStoreReader(t)
	handler := makeDocResourceHandler(store, "/tmp/docs")

	_, err := handler(nil, &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "file:///../../../etc/passwd"},
	})
	assert.Error(t, err)
}
