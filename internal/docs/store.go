package docs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DocStoreReader provides read-only access to documentation files.
// Implemented by [DocStore]. Mock generated via mockery (store_mock.go).
type DocStoreReader interface {
	Get(key string) (string, bool)
	Size() int
	Keys() []string
}

// DocStore holds all documentation files loaded into memory.
// It is populated once at startup and never mutated after.
// Concurrent reads are safe without locks because Go maps
// support safe concurrent reads when no writes occur.
type DocStore struct {
	docs map[string]string // "4.7/classes/class_camera2d.md" → content
}

// NewDocStore creates an empty DocStore.
func NewDocStore() *DocStore {
	return &DocStore{
		docs: make(map[string]string),
	}
}

// Load walks the docs root directory, reads all .md, .rst, and .txt files,
// and stores them keyed by their path relative to the docs root.
// This must be called before the server starts accepting connections.
// After Load returns, the store is immutable.
func (s *DocStore) Load(docsRoot string) error {
	count := 0
	err := filepath.WalkDir(docsRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".rst" && ext != ".txt" {
			return nil
		}

		relPath, err := filepath.Rel(docsRoot, path)
		if err != nil {
			return fmt.Errorf("cannot compute relative path for %s: %w", path, err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", relPath, err)
		}

		s.docs[relPath] = string(content)
		count++
		return nil
	})

	if err != nil {
		return fmt.Errorf("loading docs: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("no documentation files found in %s", docsRoot)
	}

	return nil
}

// Get returns the content for a documentation file. The key should
// be relative to the docs root (e.g., "4.7/classes/class_camera2d.md").
// Returns false if the key is not found.
func (s *DocStore) Get(key string) (string, bool) {
	content, ok := s.docs[key]
	return content, ok
}

// Size returns the number of files in the store.
func (s *DocStore) Size() int {
	return len(s.docs)
}

// Keys returns all document keys in the store. The returned slice
// is a copy — mutating it does not affect the store.
func (s *DocStore) Keys() []string {
	keys := make([]string, 0, len(s.docs))
	for k := range s.docs {
		keys = append(keys, k)
	}
	return keys
}
