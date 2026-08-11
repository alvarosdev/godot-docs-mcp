package search

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alvarosdev/godot-docs-mcp/internal/docs"
)

func TestSearch_SingleToken(t *testing.T) {
	store := injectDocs(t, map[string]string{
		"4.7/classes/class_characterbody2d.md": "# CharacterBody2D\n\nInherits: PhysicsBody2D\n\nA 2D physics body for characters.\nIt uses move_and_slide for collision detection.",
		"4.7/classes/class_node2d.md":          "# Node2D\n\nA 2D game object.",
	})

	s := New(store, []string{"4.7"})
	results := s.Search("CharacterBody2D", "", 10)
	require.Len(t, results, 1)
	assert.Equal(t, "classes/class_characterbody2d.md", results[0].Path)
	assert.Greater(t, results[0].Score, 0.0)
}

func TestSearch_MultiTokenScore(t *testing.T) {
	store := injectDocs(t, map[string]string{
		"4.7/classes/class_a.md": "# Node\n\ncollision detection for 2D physics.",
		"4.7/classes/class_b.md": "# Node\n\ncollision handling for physics bodies.",
	})

	s := New(store, []string{"4.7"})
	results := s.Search("collision detection", "", 10)
	require.GreaterOrEqual(t, len(results), 2)
	// The doc with both tokens should rank higher.
	assert.Equal(t, "classes/class_a.md", results[0].Path)
	assert.Greater(t, results[0].Score, results[1].Score)
}

func TestSearch_HeadingBoost(t *testing.T) {
	store := injectDocs(t, map[string]string{
		"4.7/classes/class_a.md": "# CharacterBody2D\n\nThis is the body text mentioning CharacterBody2D somewhere here.",
		"4.7/classes/class_b.md": "Body text mentioning CharacterBody2D. No heading match.",
	})

	s := New(store, []string{"4.7"})
	results := s.Search("CharacterBody2D", "", 10)
	require.Len(t, results, 2)
	// Heading match should score higher.
	assert.Equal(t, "classes/class_a.md", results[0].Path)
	assert.Greater(t, results[0].Score, results[1].Score)
}

func TestSearch_SectionBoost(t *testing.T) {
	store := injectDocs(t, map[string]string{
		"4.7/classes/class_node.md": "# Node\n\nA game object Node.",
		"4.7/about/introduction.md": "# Introduction\n\nA game object Node is the base of everything.",
	})

	s := New(store, []string{"4.7"})
	results := s.Search("Node game object", "", 10)
	require.GreaterOrEqual(t, len(results), 2)
	// classes/ should rank above about/ for the same content tokens.
	assert.Equal(t, "classes/class_node.md", results[0].Path)
	assert.Greater(t, results[0].Score, results[1].Score)
}

func TestSearch_ProximityBonus(t *testing.T) {
	store := injectDocs(t, map[string]string{
		"4.7/classes/class_a.md": "collision detection is used in physics. " +
			"These two words are right next to each other in this sentence.",
		"4.7/classes/class_b.md": "collision is important. " +
			generatePadding(500) +
			"detection is also important but far away from the first word.",
	})

	s := New(store, []string{"4.7"})
	results := s.Search("collision detection", "", 10)
	require.GreaterOrEqual(t, len(results), 2)
	assert.Equal(t, "classes/class_a.md", results[0].Path)
	assert.Greater(t, results[0].Score, results[1].Score)
}

func TestSearch_VersionFilter(t *testing.T) {
	store := injectDocs(t, map[string]string{
		"4.7/classes/class_node.md": "# Node\n\nNode documentation.",
		"3.6/classes/class_node.md": "# Node\n\nOlder Node documentation.",
	})

	s := New(store, []string{"3.6", "4.7"})
	results := s.Search("Node", "3.6", 10)
	require.Len(t, results, 1)
	assert.Equal(t, "3.6", results[0].Version)
}

func TestSearch_NoResults(t *testing.T) {
	store := injectDocs(t, map[string]string{
		"4.7/classes/class_node.md": "# Node\n\nDocumentation.",
	})

	s := New(store, []string{"4.7"})
	results := s.Search("nonexistentxyz", "", 10)
	assert.Empty(t, results)
}

func TestSearch_SnippetContainsContext(t *testing.T) {
	store := injectDocs(t, map[string]string{
		"4.7/classes/class_node.md": "# Node\n\nInherits: Object\n\n" +
			"This is the node class. It provides base functionality for all scene objects. " +
			"Nodes can be arranged in a tree structure.",
	})

	s := New(store, []string{"4.7"})
	results := s.Search("node", "", 10)
	require.NotEmpty(t, results)
	assert.NotEmpty(t, results[0].Snippet)
	// Snippet should not contain raw markup
	assert.NotContains(t, results[0].Snippet, "```")
	assert.NotContains(t, results[0].Snippet, "**")
}

// ─── helpers ─────────────────────────────────────────────────────────────

func injectDocs(t *testing.T, files map[string]string) *docs.DocStore {
	t.Helper()
	dir := t.TempDir()
	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("writefile: %v", err)
		}
	}
	s := docs.NewDocStore()
	if err := s.Load(dir); err != nil {
		t.Fatalf("load: %v", err)
	}
	return s
}

// ─── integration tests ───────────────────────────────────────────────────

func TestIntegration_SearchRealTestdata(t *testing.T) {
	s := docs.NewDocStore()
	err := s.Load("../docs/testdata/store")
	require.NoError(t, err)

	searcher := New(s, []string{"4.7"})
	results := searcher.Search("node", "", 10)
	require.NotEmpty(t, results)
	// Should find class_node.md
	found := false
	for _, r := range results {
		if r.Path == "classes/class_node.md" {
			found = true
			assert.Equal(t, "4.7", r.Version)
			assert.Greater(t, r.Score, 0.0)
			assert.NotEmpty(t, r.Snippet)
		}
	}
	assert.True(t, found, "expected to find classes/class_node.md in results")
}

func TestIntegration_SearchWithVersionFilter(t *testing.T) {
	s := docs.NewDocStore()
	// Inject multi-version testdata
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(dir+"/4.7/classes", 0755))
	require.NoError(t, os.MkdirAll(dir+"/3.6/classes", 0755))
	require.NoError(t, os.WriteFile(dir+"/4.7/classes/class_node.md", []byte("# Node\n\nGodot 4.7 node docs."), 0644))
	require.NoError(t, os.WriteFile(dir+"/3.6/classes/class_node.md", []byte("# Node\n\nGodot 3.6 node docs."), 0644))
	require.NoError(t, s.Load(dir))

	searcher := New(s, []string{"3.6", "4.7"})

	// Unfiltered: both versions
	all := searcher.Search("node", "", 10)
	assert.Len(t, all, 2)

	// Filtered: only 3.6
	v36 := searcher.Search("node", "3.6", 10)
	assert.Len(t, v36, 1)
	assert.Equal(t, "3.6", v36[0].Version)

	// Filtered: only 4.7
	v47 := searcher.Search("node", "4.7", 10)
	assert.Len(t, v47, 1)
	assert.Equal(t, "4.7", v47[0].Version)
}

func generatePadding(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'x'
	}
	return string(b)
}
