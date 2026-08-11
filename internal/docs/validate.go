package docs

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateDocPath checks that a user-supplied path is safe and within
// the allowed docs directory. It returns the cleaned relative path
// without any directory traversal or absolute path components.
//
// Rejects:
//   - Paths with ".." (directory traversal)
//   - Absolute paths (starting with / or \)
//   - Windows volume paths (C:\)
//   - Paths containing null bytes
//   - URL-encoded traversal sequences (%2e%2e)
//   - Excessively long paths (>4096 chars)
//
// The returned path is relative to the version docs directory and
// ready to be joined with the docs root.
func ValidateDocPath(rawPath string) (string, error) {
	// Reject null bytes
	if strings.ContainsRune(rawPath, 0) {
		return "", fmt.Errorf("path contains null byte")
	}

	// Reject excessively long paths
	if len(rawPath) > 4096 {
		return "", fmt.Errorf("path too long (%d characters)", len(rawPath))
	}

	// Detect URL-encoded traversal attempts (%2e = ".", %2f = "/", %5c = "\")
	// as defense-in-depth before IsLocal checks.
	if strings.Contains(rawPath, "%") {
		lower := strings.ToLower(rawPath)
		if strings.Contains(lower, "%2e%2e") || strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c") {
			return "", fmt.Errorf("path contains URL-encoded traversal sequence")
		}
	}

	// Clean the path to normalize slashes and resolve ".." segments
	cleaned := filepath.Clean(rawPath)

	// filepath.IsLocal rejects:
	// - Absolute paths (/foo, C:\foo)
	// - Paths containing ".."
	// - Windows volume names
	if !filepath.IsLocal(cleaned) {
		return "", fmt.Errorf("path traversal not allowed: %q", rawPath)
	}

	// Additional check: path must not be empty after cleaning
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("empty path")
	}

	return cleaned, nil
}

// ResolveDocPath validates the user-supplied path and resolves it
// within the specified version docs directory. Returns the absolute
// path or an error.
func ResolveDocPath(docsRoot, version, relativePath string) (string, error) {
	cleanPath, err := ValidateDocPath(relativePath)
	if err != nil {
		return "", err
	}

	versionDir := filepath.Join(docsRoot, version)
	resolved := filepath.Join(versionDir, cleanPath)

	// Defense in depth: verify the resolved path is within docsRoot
	absDocsRoot, err := filepath.Abs(docsRoot)
	if err != nil {
		return "", fmt.Errorf("cannot resolve docs root: %w", err)
	}
	absResolved, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path: %w", err)
	}
	if !strings.HasPrefix(absResolved, absDocsRoot+string(filepath.Separator)) && absResolved != absDocsRoot {
		return "", fmt.Errorf("path outside docs directory")
	}

	return absResolved, nil
}
