package docs

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Version metadata tests ───────────────────────────────────────────────

func TestLoadVersionMetadata_GoodFile(t *testing.T) {
	root := filepath.Join("testdata", "versions", "good")
	meta, err := LoadVersionMetadata(root)
	require.NoError(t, err)
	assert.Len(t, meta.Versions, 2)
	assert.Equal(t, []string{"3.6", "4.7"}, meta.Versions)
	assert.Equal(t, "4.7", meta.Latest)
	assert.Equal(t, "2026.08.10", meta.Built)
}

func TestLoadVersionMetadata_MalformedJSON(t *testing.T) {
	root := filepath.Join("testdata", "versions", "malformed")
	_, err := LoadVersionMetadata(root)
	assert.Error(t, err)
}

func TestLoadVersionMetadata_MissingFile_ScanFallback(t *testing.T) {
	root := filepath.Join("testdata", "versions", "scanned")
	meta, err := LoadVersionMetadata(root)
	require.NoError(t, err)
	assert.Equal(t, "4.7", meta.Latest)
	assert.Len(t, meta.Versions, 2)
}

func TestLoadVersionMetadata_EmptyDir(t *testing.T) {
	root := filepath.Join("testdata", "versions", "empty")
	_, err := LoadVersionMetadata(root)
	assert.Error(t, err)
}

// ─── Path validation tests ────────────────────────────────────────────────

func TestValidateDocPath_Valid(t *testing.T) {
	for _, p := range []string{
		"classes/class_camera2d.md",
		"tutorials/2d/movement.md",
		"getting_started/step_by_step/scripting.md",
		"docs_tree.txt",
	} {
		_, err := ValidateDocPath(p)
		assert.NoError(t, err, "path %q", p)
	}
}

func TestValidateDocPath_DotDot(t *testing.T) {
	for _, p := range []string{"../../etc/passwd", "classes/../../../etc/shadow", "..", "../"} {
		_, err := ValidateDocPath(p)
		assert.Error(t, err, "path %q should be rejected", p)
	}
}

func TestValidateDocPath_Absolute(t *testing.T) {
	for _, p := range []string{"/etc/passwd", "/home/user/file.md"} {
		_, err := ValidateDocPath(p)
		assert.Error(t, err, "absolute path %q should be rejected", p)
	}
}

func TestValidateDocPath_URLEncodedTraversal(t *testing.T) {
	for _, p := range []string{
		"%2e%2e/%2e%2e/etc/passwd",
		"%2E%2E/%2E%2E/etc/passwd",
		"classes/%2e%2e%2fsecret",
		"foo%5c..%5cbar",
	} {
		_, err := ValidateDocPath(p)
		assert.Error(t, err, "URL-encoded traversal %q should be rejected", p)
	}
}

func TestValidateDocPath_NullByte(t *testing.T) {
	_, err := ValidateDocPath("classes/foo.md\x00.jpg")
	assert.Error(t, err)
}

func TestValidateDocPath_PathTooLong(t *testing.T) {
	long := make([]byte, 4097)
	for i := range long {
		long[i] = 'a'
	}
	_, err := ValidateDocPath(string(long))
	assert.Error(t, err)
}

func TestResolveDocPath_Valid(t *testing.T) {
	abs, err := ResolveDocPath("/tmp/docs", "4.7", "classes/class_node.md")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/docs/4.7/classes/class_node.md", abs)
}

func TestResolveDocPath_OutsideRoot(t *testing.T) {
	_, err := ResolveDocPath("/tmp/docs", "4.7", "../../etc/passwd")
	assert.Error(t, err)
}

// ─── DocStore tests ───────────────────────────────────────────────────────

func TestNewDocStore_Empty(t *testing.T) {
	s := NewDocStore()
	assert.Equal(t, 0, s.Size())
}

func TestDocStore_LoadAndGet(t *testing.T) {
	root := filepath.Join("testdata", "store")
	s := NewDocStore()
	require.NoError(t, s.Load(root))

	content, ok := s.Get("4.7/classes/class_node.md")
	require.True(t, ok)
	assert.NotEmpty(t, content)

	content, ok = s.Get("4.7/docs_tree.txt")
	require.True(t, ok)
	assert.NotEmpty(t, content)
}

func TestDocStore_GetMissing(t *testing.T) {
	root := filepath.Join("testdata", "store")
	s := NewDocStore()
	require.NoError(t, s.Load(root))

	_, ok := s.Get("nonexistent/file.md")
	assert.False(t, ok)
}

func TestDocStore_ConcurrentReads(t *testing.T) {
	root := filepath.Join("testdata", "store")
	s := NewDocStore()
	require.NoError(t, s.Load(root))

	var wg sync.WaitGroup
	const goroutines = 100
	errCh := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, ok := s.Get("4.7/classes/class_node.md")
				if !ok {
					errCh <- assert.AnError
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		assert.NoError(t, err)
	}
}

func TestDocStore_Size(t *testing.T) {
	root := filepath.Join("testdata", "store")
	s := NewDocStore()
	require.NoError(t, s.Load(root))
	assert.Greater(t, s.Size(), 0)
}

func TestDocStore_Keys(t *testing.T) {
	root := filepath.Join("testdata", "store")
	s := NewDocStore()
	require.NoError(t, s.Load(root))

	keys := s.Keys()
	assert.Len(t, keys, s.Size(), "Keys() length should match Size()")

	// Verify keys is a copy — mutating it doesn't affect the store
	originalKey := keys[0]
	originalSize := s.Size()
	keys[0] = "modified_key"
	_, ok := s.Get("modified_key")
	assert.False(t, ok, "mutated key should not exist in store")
	// Original key should still work
	_, ok = s.Get(originalKey)
	assert.True(t, ok, "original key should still exist after slice mutation")
	assert.Equal(t, originalSize, s.Size(), "mutating returned slice should not affect store")

	// Verify keys are valid (can be used with Get)
	freshKeys := s.Keys()
	for _, k := range freshKeys {
		_, ok := s.Get(k)
		assert.True(t, ok, "key %q should exist in store", k)
	}
}

func TestDocStore_LoadsRstFiles(t *testing.T) {
	root := filepath.Join("testdata", "store")
	s := NewDocStore()
	require.NoError(t, s.Load(root))

	// .rst file should be loaded
	content, ok := s.Get("4.7/classes/class_test.rst")
	assert.True(t, ok, ".rst file should be loaded by DocStore")
	assert.NotEmpty(t, content)
}
