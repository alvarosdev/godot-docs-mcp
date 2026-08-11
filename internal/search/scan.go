// Package search provides zero-dependency full-text search over the
// in-memory Godot documentation store.
package search

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/alvarosdev/godot-docs-mcp/internal/docs"
)

// Result holds a single search hit.
type Result struct {
	Path    string  `json:"path"`
	Version string  `json:"version"`
	Score   float64 `json:"score"`
	Snippet string  `json:"snippet"`
}

// Searcher performs token-based search over a DocStoreReader.
type Searcher struct {
	store    docs.DocStoreReader
	versions []string
}

// New creates a Searcher backed by the given store and version metadata.
func New(store docs.DocStoreReader, versions []string) *Searcher {
	return &Searcher{store: store, versions: versions}
}

// Search runs a query and returns ranked results up to limit.
// If version is non-empty, only that version's docs are searched.
// If category is non-empty, only docs whose path starts with category+"/" are searched.
func (s *Searcher) Search(query string, version string, category string, limit int) []Result {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	tokens := tokenize(query)
	if len(tokens) == 0 {
		return nil
	}

	var results []Result
	for _, key := range s.store.Keys() {
		docVersion, docPath := splitVersion(key)
		if version != "" && docVersion != version {
			continue
		}
		if category != "" && !strings.HasPrefix(docPath, category+"/") {
			continue
		}

		content, ok := s.store.Get(key)
		if !ok {
			continue
		}

		score := score(tokens, content, docPath)
		if score == 0 {
			continue
		}

		results = append(results, Result{
			Path:    docPath,
			Version: docVersion,
			Score:   score,
			Snippet: extractSnippet(content, tokens),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

// ─── tokenizer ───────────────────────────────────────────────────────────

// stopWords is hoisted to package level to avoid reallocation per call.
var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "is": true, "are": true,
	"was": true, "were": true, "be": true, "been": true, "being": true,
	"of": true, "in": true, "to": true, "for": true, "on": true,
	"with": true, "as": true, "at": true, "by": true, "or": true,
	"and": true, "not": true, "from": true, "this": true, "that": true,
	"it": true, "its": true, "can": true, "will": true, "has": true,
	"have": true, "do": true, "does": true, "but": true, "if": true,
}

func tokenize(query string) []string {
	// Lowercase once for the entire query to avoid per-token lowercasing.
	lowerQuery := strings.ToLower(query)
	words := strings.Fields(lowerQuery)
	tokens := make([]string, 0, len(words))
	seen := make(map[string]bool, len(words))
	const maxTokens = 50
	for _, w := range words {
		w = strings.TrimFunc(w, isPunct)
		if len(w) < 2 || stopWords[w] || seen[w] {
			continue
		}
		seen[w] = true
		tokens = append(tokens, w)
		if len(tokens) >= maxTokens {
			break
		}
	}
	return tokens
}

func isPunct(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

// ─── scoring ─────────────────────────────────────────────────────────────

// filenameBoost is the multiplier applied when a query token matches the
// file's class name stem (e.g., "CharacterBody2D" → "characterbody2d").
const filenameBoost = 10.0

// headingRe is hoisted to package level to avoid recompilation.
// Matches GFM ATX headings: ^#{1,6}\s+
var headingRe = regexp.MustCompile(`^#{1,6}\s+`)

func score(tokens []string, content string, path string) float64 {
	// Lowercase content once; reused for coverage and snippet.
	contentLower := strings.ToLower(content)

	// Token coverage: fraction of query tokens found.
	matched := 0
	positions := make(map[string][]int, len(tokens))
	for _, t := range tokens {
		pos := findPositions(contentLower, t)
		if len(pos) > 0 {
			matched++
			positions[t] = pos
		}
	}
	coverage := float64(matched) / float64(len(tokens))

	// Heading boost: 3× for matches in heading lines.
	headingBoost := headingScore(contentLower, tokens)

	// Section boost: classes/ > tutorials/ > other.
	sectionBoost := sectionScore(path)

	// Proximity bonus: how close together the tokens appear.
	proximity := proximityScore(positions)

	// Filename match boost: query tokens matching the class name get 10×.
	className := extractClassName(path)
	fnBoost := 1.0
	for _, t := range tokens {
		if t == className {
			fnBoost = filenameBoost
			break
		}
	}

	return coverage * headingBoost * sectionBoost * proximity * fnBoost
}

func findPositions(content, token string) []int {
	var pos []int
	offset := 0
	for {
		idx := strings.Index(content[offset:], token)
		if idx < 0 {
			break
		}
		pos = append(pos, offset+idx)
		offset += idx + len(token)
	}
	return pos
}

// headingScore returns 3.0 if any token appears in a heading line, 1.0 otherwise.
// Content is already lowercased; no additional lowercasing per line.
func headingScore(contentLower string, tokens []string) float64 {
	lines := strings.Split(contentLower, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !headingRe.MatchString(trimmed) {
			continue
		}
		for _, t := range tokens {
			if strings.Contains(trimmed, t) {
				return 3.0
			}
		}
	}
	return 1.0
}

// sectionScore returns a boost based on the document's section.
func sectionScore(path string) float64 {
	if strings.HasPrefix(path, "classes/") {
		return 2.0
	}
	if strings.HasPrefix(path, "tutorials/") {
		return 1.5
	}
	return 1.0
}

// proximityScore returns 0.5–1.5 based on how clustered the token
// matches are within a 500-char window.
func proximityScore(positions map[string][]int) float64 {
	all := make([]int, 0)
	for _, p := range positions {
		all = append(all, p...)
	}
	if len(all) < 2 {
		return 1.0
	}
	sort.Ints(all)

	const window = 500
	maxDensity := 0
	for _, start := range all {
		end := start + window
		count := 0
		for _, p := range all {
			if p >= start && p <= end {
				count++
			}
		}
		if count > maxDensity {
			maxDensity = count
		}
	}

	density := float64(maxDensity) / float64(len(all))
	// Map density 0.2 → 0.5, density 1.0 → 1.5
	return 0.5 + density
}

// ─── snippet extraction ──────────────────────────────────────────────────

// markupReplacer is hoisted to avoid rebuilding on each snippet call.
var markupReplacer = strings.NewReplacer(
	"```", "",
	"`", "",
	"**", "",
	"__", "",
	"*", "",
	"_", "",
	"#", "",
)

func extractSnippet(content string, tokens []string) string {
	// Single-pass: lowercase once, compute positions once, find best window via positions.
	contentLower := strings.ToLower(content)

	// Collect positions for each token in a single pass per token (via findPositions).
	positions := make(map[string][]int, len(tokens))
	var allPositions []int
	for _, t := range tokens {
		ps := findPositions(contentLower, t)
		if len(ps) > 0 {
			positions[t] = ps
			allPositions = append(allPositions, ps...)
		}
	}

	const window = 200
	bestStart := -1

	if len(allPositions) > 0 {
		sort.Ints(allPositions)
		bestCount := 0
		// Single-pass over sorted positions to find highest-density window.
		for _, start := range allPositions {
			count := 0
			end := start + window
			for _, t := range tokens {
				for _, p := range positions[t] {
					if p >= start && p < end {
						count++
						break
					}
				}
			}
			if count > bestCount {
				bestCount = count
				bestStart = start
				if bestCount == len(tokens) {
					break
				}
			}
		}
		// Provide a little leading context when possible.
		if bestStart > 20 {
			bestStart -= 20
		}
		if bestStart < 0 {
			bestStart = 0
		}
	} else {
		bestStart = 0
	}

	snippetEnd := min(bestStart+window, len(content))
	snippet := content[bestStart:snippetEnd]

	snippet = stripMarkup(snippet)

	if bestStart > 0 {
		snippet = "…" + snippet
	}
	if snippetEnd < len(content) {
		snippet = snippet + "…"
	}
	return snippet
}

func stripMarkup(s string) string {
	// Single-pass via replacer hoisted to package level.
	s = markupReplacer.Replace(s)
	return strings.TrimSpace(s)
}

// ─── helpers ─────────────────────────────────────────────────────────────

// extractClassName derives a lowercase class name stem from a doc path.
// "classes/class_characterbody2d.rst" → "characterbody2d"
// "tutorials/2d/movement.rst" → "movement"
func extractClassName(path string) string {
	base := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		base = path[idx+1:]
	}
	// Remove known prefixes and extensions.
	base = strings.TrimSuffix(base, ".rst")
	base = strings.TrimSuffix(base, ".md")
	base = strings.TrimPrefix(base, "class_")
	return strings.ToLower(base)
}

// splitVersion splits "4.7/classes/class_node.md" → ("4.7", "classes/class_node.md").
func splitVersion(key string) (version, rest string) {
	idx := strings.Index(key, "/")
	if idx < 0 {
		return "", key
	}
	return key[:idx], key[idx+1:]
}
