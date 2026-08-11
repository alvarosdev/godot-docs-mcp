package tools

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alvarosdev/godot-docs-mcp/internal/docs"
)

// GFM fixture helpers.

func gfmClassDoc(className, inherits string, sections map[string]string) string {
	var b strings.Builder
	b.WriteString("# " + className + "\n\n")
	if inherits != "" {
		b.WriteString("**Inherits:** " + inherits + "\n\n")
	}
	for heading, body := range sections {
		b.WriteString("## " + heading + "\n\n")
		b.WriteString(body + "\n\n")
	}
	return b.String()
}

func gfmMethodsTable(rows ...string) string {
	var b strings.Builder
	b.WriteString("> | Type | Name |\n> | --- | --- |\n")
	for _, r := range rows {
		b.WriteString(r + "\n")
	}
	return b.String()
}

// ─── findSection edge cases ──────────────────────────────────────────────

func TestFindSection_Methods(t *testing.T) {
	content := "# Control\n\nDesc.\n\n## Properties\n\nprops here\n\n## Methods\n\n- void do_thing()\n\n## Signals\n\nsignals"
	section, ok := findSection(content, "Methods")
	require.True(t, ok)
	assert.Contains(t, section, "void do_thing()")
	assert.NotContains(t, section, "Signals")
}

func TestFindSection_CaseInsensitive(t *testing.T) {
	content := "# Methods\n\nmethod content here.\n\n## Other\n\n"
	_, ok := findSection(content, "methods")
	assert.True(t, ok)
}

func TestFindSection_NotFound(t *testing.T) {
	content := "# Control\n\n## Properties\n\ncontent\n\n## Methods\n\ncontent"
	_, ok := findSection(content, "Signals")
	assert.False(t, ok)
}

func TestFindSection_LastSection(t *testing.T) {
	content := "# Control\n\n## Methods\n\nlast content"
	section, ok := findSection(content, "Methods")
	require.True(t, ok)
	assert.Contains(t, section, "last content")
}

func TestOverviewFindSection_NoHeadings(t *testing.T) {
	content := "Just plain text without any headings."
	_, ok := findSection(content, "Methods")
	assert.False(t, ok)
	assert.Empty(t, listSections(content))
}

func TestOverviewFindSection_WhitespaceHeading(t *testing.T) {
	content := "# Control\n\n   ## Methods   \n\ncontent here\n\n## Signals\n\n"
	section, ok := findSection(content, "Methods")
	require.True(t, ok)
	assert.Contains(t, section, "content here")
}

func TestListSections_GFM(t *testing.T) {
	content := "# Control\n\n## Description\n\ntext\n\n## Properties\n\n## Methods\n\n## Signals\n\n"
	sections := listSections(content)
	assert.Contains(t, sections, "Description")
	assert.Contains(t, sections, "Properties")
	assert.Contains(t, sections, "Methods")
	assert.Contains(t, sections, "Signals")
	assert.NotContains(t, sections, "Control") // H1 is included but we check count
}

// ─── get_class_overview edge cases ────────────────────────────────────────

func TestOverview_EmptyContent(t *testing.T) {
	ov, err := extractOverview("")
	require.NoError(t, err)
	assert.NotNil(t, ov)
	assert.Empty(t, ov.Class)
	assert.Equal(t, 0, ov.Counts.Methods)
}

func TestOverview_NoSignals(t *testing.T) {
	content := gfmClassDoc("Control", "Node", map[string]string{
		"Description": "Base class.",
		"Properties":  gfmMethodsTable("> | float | anchor_left |"),
		"Methods":     gfmMethodsTable("> | void | do_thing() |"),
	})
	ov, err := extractOverview(content)
	require.NoError(t, err)
	assert.Equal(t, 0, ov.Counts.Signals)
	assert.Empty(t, ov.Signals)
}

func TestOverview_MalformedRow(t *testing.T) {
	content := "# Control\n\n## Methods\n\n> | void | good_method() |\n> malformed row without pipes\n> | bool | another_method() |\n"
	ov, err := extractOverview(content)
	require.NoError(t, err)
	// At least the two good rows should be extracted; malformed row skipped.
	assert.GreaterOrEqual(t, len(ov.Methods), 2)
}

func TestOverview_CompleteFidelity(t *testing.T) {
	content := "# Control\n\n**Inherits:** CanvasItem\n\n## Description\n\nBase class for all UI.\n\n## Properties\n\n> | float | anchor_left |\n> | float | anchor_top |\n\n## Methods\n\n> | void | do_thing() |\n\n## Signals\n\n> | void | pressed() |\n"
	ov, err := extractOverview(content)
	require.NoError(t, err)
	assert.Equal(t, "Control", ov.Class)
	assert.Equal(t, 2, ov.Counts.Properties)
	assert.Equal(t, 1, ov.Counts.Methods)
	assert.Equal(t, 1, ov.Counts.Signals)
}

// ─── DocStore GFM + RST fallback ──────────────────────────────────────────

func TestDocStore_GFMAndRSTFallback(t *testing.T) {
	// Use real testdata: it has .md and .rst
	s := docs.NewDocStore()
	err := s.Load("../docs/testdata/store")
	require.NoError(t, err)
	// .rst file should be loaded
	_, ok := s.Get("4.7/classes/class_test.rst")
	assert.True(t, ok, ".rst fallback should be loaded")
	// .md file should be loaded
	_, ok = s.Get("4.7/classes/class_node.md")
	assert.True(t, ok, ".md should be loaded")
}

// ─── get_class_overview tool ──────────────────────────────────────────────

func TestGetClassOverview_Success(t *testing.T) {
	content := gfmClassDoc("Control", "CanvasItem", map[string]string{
		"Description": "Base class for all UI.",
		"Methods":     gfmMethodsTable("> | void | do_thing() virtual |"),
		"Properties":  gfmMethodsTable("> | float | anchor_left |"),
	})
	// Build a mock store with the class file
	store := mockStore(map[string]string{
		"4.7/classes/class_control.md": content,
	})
	meta := &docs.VersionMetadata{Versions: []string{"4.7"}, Latest: "4.7"}
	handler := makeGetClassOverview(store, meta)
	result, _, err := handler(nil, nil, getClassOverviewArgs{Class: "Control"})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].(*mcp.TextContent).Text, "Control")
}

func TestGetClassOverview_NotFound(t *testing.T) {
	store := mockStore(map[string]string{})
	meta := &docs.VersionMetadata{Versions: []string{"4.7"}, Latest: "4.7"}
	handler := makeGetClassOverview(store, meta)
	result, _, err := handler(nil, nil, getClassOverviewArgs{Class: "NonExistent"})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// helpers
func mockStore(files map[string]string) docs.DocStoreReader {
	// Use a real DocStore with injected files via temp dir
	return &memStore{files: files}
}

type memStore struct {
	files map[string]string
}

func (m *memStore) Get(key string) (string, bool) { v, ok := m.files[key]; return v, ok }
func (m *memStore) Size() int                     { return len(m.files) }
func (m *memStore) Keys() []string {
	keys := make([]string, 0, len(m.files))
	for k := range m.files {
		keys = append(keys, k)
	}
	return keys
}
