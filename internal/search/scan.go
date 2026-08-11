// Package search provides zero-dependency full-text search over the
// in-memory Godot documentation store.
package search

import (
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
func (s *Searcher) Search(query string, version string, limit int) []Result {
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
	words := strings.Fields(query)
	tokens := make([]string, 0, len(words))
	seen := make(map[string]bool, len(words))
	const maxTokens = 50
	for _, w := range words {
		w = strings.ToLower(strings.TrimFunc(w, isPunct))
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

func score(tokens []string, content string, path string) float64 {
	contentLower := strings.ToLower(content)

	// Token coverage: fraction of query tokens found.
	matched := 0
	positions := make(map[string][]int)
	for _, t := range tokens {
		pos := findPositions(contentLower, t)
		if len(pos) > 0 {
			matched++
			positions[t] = pos
		}
	}
	coverage := float64(matched) / float64(len(tokens))

	// Heading boost: 3× for matches in heading lines.
	headingBoost := headingScore(contentLower, tokens, positions)

	// Section boost: classes/ > tutorials/ > other.
	sectionBoost := sectionScore(path)

	// Proximity bonus: how close together the tokens appear.
	proximity := proximityScore(positions)

	return coverage * headingBoost * sectionBoost * proximity
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

// headingScore returns 3.0 if any token appears in a heading line,
// 1.0 otherwise.
func headingScore(content string, tokens []string, positions map[string][]int) float64 {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		// Markdown ATX headings (# or ##) and RST-style underlines (=== or ---)
		if strings.HasPrefix(trimmed, "#") ||
			(strings.HasPrefix(trimmed, "=") && !strings.Contains(trimmed, " ")) ||
			(len(trimmed) > 2 && strings.Trim(trimmed, "=-") == "") {
			for _, t := range tokens {
				if strings.Contains(lower, t) {
					return 3.0
				}
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

func extractSnippet(content string, tokens []string) string {
	contentLower := strings.ToLower(content)

	// Find the highest-density region of matches.
	bestStart := -1
	bestCount := 0
	const window = 200
	for i := 0; i < len(contentLower); i += 50 {
		end := min(i+window, len(contentLower))
		slice := contentLower[i:end]
		count := 0
		for _, t := range tokens {
			if strings.Contains(slice, t) {
				count++
			}
		}
		if count > bestCount {
			bestCount = count
			bestStart = i
		}
	}

	if bestStart < 0 {
		// Fallback: use the first match of any token.
		for _, t := range tokens {
			if idx := strings.Index(contentLower, t); idx >= 0 {
				bestStart = max(0, idx-100)
				break
			}
		}
	}
	if bestStart < 0 {
		bestStart = 0
	}

	snippetEnd := min(bestStart+200, len(content))
	snippet := content[bestStart:snippetEnd]

	// Strip common Markdown markup for readability.
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
	// Remove code fences, heading markers, bold/italic.
	s = strings.ReplaceAll(s, "```", "")
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "#", "")
	return strings.TrimSpace(s)
}

// ─── helpers ─────────────────────────────────────────────────────────────

// splitVersion splits "4.7/classes/class_node.md" → ("4.7", "classes/class_node.md").
func splitVersion(key string) (version, rest string) {
	idx := strings.Index(key, "/")
	if idx < 0 {
		return "", key
	}
	return key[:idx], key[idx+1:]
}
