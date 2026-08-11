// Package docs provides utilities for working with versioned Godot documentation.
package docs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// VersionMetadata holds the content of docs/versions.json.
type VersionMetadata struct {
	Versions []string `json:"versions"`
	Latest   string   `json:"latest"`
	Built    string   `json:"built"`
}

// versionDirRe matches "major.minor" directory names.
var versionDirRe = regexp.MustCompile(`^\d+\.\d+$`)

// LoadVersionMetadata reads versions.json from the docs root.
// If the file is missing or malformed, falls back to scanning
// docs/ for version-pattern directories.
func LoadVersionMetadata(docsRoot string) (*VersionMetadata, error) {
	meta, err := readVersionsJSON(docsRoot)
	if err == nil {
		return meta, nil
	}
	return scanVersionDirs(docsRoot)
}

// readVersionsJSON reads and parses docs/versions.json.
func readVersionsJSON(docsRoot string) (*VersionMetadata, error) {
	path := filepath.Join(docsRoot, "versions.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta VersionMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	if len(meta.Versions) == 0 {
		return nil, fmt.Errorf("empty versions list")
	}
	return &meta, nil
}

// scanVersionDirs scans docs/ for directories matching the "major.minor"
// pattern and builds a minimal VersionMetadata.
func scanVersionDirs(docsRoot string) (*VersionMetadata, error) {
	entries, err := os.ReadDir(docsRoot)
	if err != nil {
		return nil, fmt.Errorf("cannot read docs directory: %w", err)
	}

	var versions []string
	for _, e := range entries {
		if e.IsDir() && versionDirRe.MatchString(e.Name()) {
			versions = append(versions, e.Name())
		}
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no version directories found in %s", docsRoot)
	}

	sortVersions(versions)
	return &VersionMetadata{
		Versions: versions,
		Latest:   versions[len(versions)-1],
		Built:    "unknown",
	}, nil
}

// sortVersions sorts version strings in ascending order (e.g., ["3.6", "4.7"]).
func sortVersions(versions []string) {
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i], versions[j]) < 0
	})
}

// compareVersions compares two "major.minor" strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareVersions(a, b string) int {
	ma, _ := strconv.Atoi(strings.Split(a, ".")[0])
	mb, _ := strconv.Atoi(strings.Split(b, ".")[0])
	if ma != mb {
		return ma - mb
	}
	mina, _ := strconv.Atoi(strings.Split(a, ".")[1])
	minb, _ := strconv.Atoi(strings.Split(b, ".")[1])
	return mina - minb
}

// GetVersionDocsDir returns the version-specific docs directory path.
func GetVersionDocsDir(docsRoot, version string) string {
	return filepath.Join(docsRoot, version)
}
