package tools

import (
	"regexp"
	"strings"
)

var (
	inheritsRe = regexp.MustCompile(`\*\*Inherits:\*\* (.*)`)
	memberRe   = regexp.MustCompile(`>\s*\| ([^|]+) \| ([^|]+) \|`)
	headingRe  = regexp.MustCompile(`^#{1,6}\s+(.+)$`)
)

// ClassOverview holds structured metadata extracted from a class GFM file.
type ClassOverview struct {
	Class      string   `json:"class"`
	Inherits   string   `json:"inherits"`
	Summary    string   `json:"summary"`
	Counts     Counts   `json:"counts"`
	Methods    []Member `json:"methods"`
	Properties []Member `json:"properties,omitempty"`
	Signals    []Member `json:"signals,omitempty"`
}

// Counts holds the number of members per section.
type Counts struct {
	Properties int `json:"properties"`
	Methods    int `json:"methods"`
	Signals    int `json:"signals"`
	Enums      int `json:"enums"`
	Constants  int `json:"constants"`
}

// Member represents a single class member (method, property, signal, etc.).
type Member struct {
	Name   string   `json:"name"`
	Type   string   `json:"type,omitempty"`
	Badges []string `json:"badges,omitempty"`
}

// extractOverview parses GFM content and returns a ClassOverview.
func extractOverview(gfmContent string) (*ClassOverview, error) {
	ov := &ClassOverview{}
	if strings.TrimSpace(gfmContent) == "" {
		return ov, nil
	}
	lines := strings.Split(gfmContent, "\n")

	// 1. Class name — first H1.
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		m := headingRe.FindStringSubmatch(trimmed)
		if m == nil {
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			ov.Class = strings.TrimSpace(m[1])
			break
		}
	}
	if ov.Class == "" {
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if m := headingRe.FindStringSubmatch(trimmed); m != nil {
				ov.Class = strings.TrimSpace(m[1])
				break
			}
		}
	}

	// 2. Inheritance.
	if m := inheritsRe.FindStringSubmatch(gfmContent); m != nil {
		ov.Inherits = cleanInherits(m[1])
	}

	// 3. Summary — first paragraph after ## Description.
	if descSection, ok := findSectionContent(gfmContent, "Description"); ok {
		ov.Summary = extractFirstParagraph(descSection)
	} else if descSection, ok := findSection(gfmContent, "Description"); ok {
		ov.Summary = extractFirstParagraph(descSection)
	}

	// 4. Member tables.
	ov.Properties = extractMembers(gfmContent, "Properties")
	ov.Methods = extractMembers(gfmContent, "Methods")
	ov.Signals = extractMembers(gfmContent, "Signals")
	enums := extractMembers(gfmContent, "Enumerations")
	if enums == nil {
		enums = extractMembers(gfmContent, "Enums")
	}
	constants := extractMembers(gfmContent, "Constants")

	ov.Counts = Counts{
		Methods:    len(ov.Methods),
		Properties: len(ov.Properties),
		Signals:    len(ov.Signals),
		Enums:      len(enums),
		Constants:  len(constants),
	}

	return ov, nil
}

func cleanInherits(raw string) string {
	// Strip cross-ref markup like `CanvasItem<class_CanvasItem>` -> CanvasItem
	crossRefRe := regexp.MustCompile("`([^<`]+)(?:<[^>]+>)?`")
	raw = crossRefRe.ReplaceAllString(raw, "$1")
	// Also handle plain cross-refs without backticks: Node3D<class_Node3D> -> Node3D
	plainRefRe := regexp.MustCompile(`([A-Za-z0-9_]+)<[^>]+>`)
	raw = plainRefRe.ReplaceAllString(raw, "$1")
	raw = strings.ReplaceAll(raw, "**", "")
	raw = strings.ReplaceAll(raw, "`", "")
	// Trim leading blockquote marker if present.
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, ">")
	raw = strings.TrimSpace(raw)
	// Collapse multiple spaces but preserve "<" separators.
	raw = regexp.MustCompile(`\s+`).ReplaceAllString(raw, " ")
	return strings.TrimSpace(raw)
}

func extractFirstParagraph(section string) string {
	section = strings.TrimSpace(section)
	if section == "" {
		return ""
	}
	paragraphs := strings.Split(strings.TrimSpace(section), "\n\n")
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Skip table-like paragraphs.
		if strings.HasPrefix(p, "|") || strings.HasPrefix(p, ">") {
			continue
		}
		lines := strings.Split(p, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "|") || strings.HasPrefix(line, ">") {
				continue
			}
			if headingRe.MatchString(line) {
				continue
			}
			return line
		}
	}
	return ""
}

func extractMembers(content, sectionName string) []Member {
	section, ok := findSectionContent(content, sectionName)
	if !ok {
		// Fallback to existing findSection (uses gfmHeadingRe) for compatibility.
		var found bool
		section, found = findSection(content, sectionName)
		if !found {
			return nil
		}
	}
	var members []Member
	for _, line := range strings.Split(section, "\n") {
		m := memberRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		typeStr := strings.TrimSpace(m[1])
		namePart := strings.TrimSpace(m[2])
		if typeStr == "" || namePart == "" {
			continue
		}
		// Skip header rows.
		if strings.EqualFold(typeStr, "Type") && strings.EqualFold(namePart, "Name") {
			continue
		}
		if isAllDashes(typeStr) || isAllDashes(namePart) {
			continue
		}
		if typeStr == "---" || namePart == "---" {
			continue
		}
		// Clean type.
		typeStr = cleanMemberType(typeStr)
		name, badges := parseMemberName(namePart)
		if name == "" || strings.EqualFold(name, "Name") || isAllDashes(name) {
			continue
		}
		if strings.EqualFold(typeStr, "Type") {
			continue
		}
		members = append(members, Member{Name: name, Type: typeStr, Badges: badges})
	}
	return members
}

func findSectionContent(content, name string) (string, bool) {
	lowerName := strings.ToLower(strings.TrimSpace(name))
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		m := headingRe.FindStringSubmatch(trimmed)
		if m == nil {
			continue
		}
		if strings.ToLower(strings.TrimSpace(m[1])) != lowerName {
			continue
		}
		level := headingLevelOverview(trimmed)
		start := i + 1
		end := len(lines)
		for j := start; j < len(lines); j++ {
			t := strings.TrimSpace(lines[j])
			if mm := headingRe.FindStringSubmatch(t); mm != nil {
				if headingLevelOverview(t) <= level {
					end = j
					break
				}
			}
		}
		return strings.Join(lines[start:end], "\n"), true
	}
	return "", false
}

func headingLevelOverview(line string) int {
	count := 0
	for _, c := range strings.TrimSpace(line) {
		if c == '#' {
			count++
		} else {
			break
		}
	}
	return count
}

func isAllDashes(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, c := range s {
		if c != '-' {
			return false
		}
	}
	return true
}

func cleanMemberType(raw string) string {
	re := regexp.MustCompile("`([^<`]+)(?:<[^>]+>)?`")
	raw = re.ReplaceAllString(raw, "$1")
	plainRe := regexp.MustCompile(`([A-Za-z0-9_]+)<[^>]+>`)
	raw = plainRe.ReplaceAllString(raw, "$1")
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "`")
	return strings.TrimSpace(raw)
}

var knownBadges = []string{"virtual", "const", "static", "vararg", "constructor", "operator", "bitfield", "required"}

func parseMemberName(raw string) (string, []string) {
	re := regexp.MustCompile("`([^<`]+)(?:<[^>]+>)?`")
	raw = re.ReplaceAllString(raw, "$1")
	plainRe := regexp.MustCompile(`([A-Za-z0-9_]+)<[^>]+>`)
	raw = plainRe.ReplaceAllString(raw, "$1")
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "`")
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return "", nil
	}
	nameField := fields[0]
	// Remove parentheses and params e.g. get_contact_body() -> get_contact_body
	parenRe := regexp.MustCompile(`\(.*\)`)
	nameField = parenRe.ReplaceAllString(nameField, "")
	nameField = strings.TrimSpace(nameField)
	nameField = strings.Trim(nameField, "`")

	var badges []string
	for _, f := range fields[1:] {
		lower := strings.ToLower(f)
		// Clean field from punctuation
		clean := strings.Trim(lower, "`,|")
		for _, b := range knownBadges {
			if clean == b || strings.Contains(clean, b) {
				found := false
				for _, existing := range badges {
					if existing == b {
						found = true
						break
					}
				}
				if !found {
					badges = append(badges, b)
				}
			}
		}
	}
	if len(badges) == 0 {
		badges = nil
	}
	return nameField, badges
}
