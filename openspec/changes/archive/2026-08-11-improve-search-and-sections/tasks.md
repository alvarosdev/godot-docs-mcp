## 1. Section parsing

- [x] 1.1 Implement `findSection(content, name)` — locate section by heading name (case-insensitive), return content between heading underline and next section boundary
- [x] 1.2 Implement `listSections(content)` — return all section heading names found in the file
- [x] 1.3 Write unit tests: find existing section, non-existent section, case-insensitive match, last section (no next boundary), empty file

## 2. get_documentation_file with section parameter

- [x] 2.1 Add `Section string` field to `getDocFileArgs` struct with jsonschema tag
- [x] 2.2 Update `makeGetDocumentationFile` handler: when `section` is set, parse the file content and extract only that section
- [x] 2.3 When section is not found, return an error listing available sections
- [x] 2.4 Add size warning: when no section is specified and `len(content) > 50000`, append a hint to the response
- [x] 2.5 Write unit tests: section found, section not found with suggestions, size warning appended

## 3. Filename match boost in search

- [x] 3.1 Implement `extractClassName(path)` — derive class name stem from filename (e.g., `classes/class_characterbody2d.rst` → `characterbody2d`)
- [x] 3.2 Add filename match check to `score()`: if any query token matches the filename stem, multiply final score by 10
- [x] 3.3 Write unit tests: exact class name match gets 10× boost, unrelated query gets no boost, mixed query gets boost only for matching tokens

## 4. Category filter in search

- [x] 4.1 Add `category` field to `searchDocArgs` struct with jsonschema tag
- [x] 4.2 Add `category` parameter to `Searcher.Search()` method signature
- [x] 4.3 Implement category filter: when non-empty, skip paths not starting with `category + "/"`
- [x] 4.4 Write unit tests: category="classes" filters correctly, category="" searches all, unknown category returns empty

## 5. Final verification

- [x] 5.1 Run full test suite: `go test ./internal/... -v -count=1`
- [x] 5.2 Run `go vet ./...` and `go build ./cmd/godot-mcp-server`
- [x] 5.3 Verify backward compatibility: existing tools work unchanged when new params are omitted
