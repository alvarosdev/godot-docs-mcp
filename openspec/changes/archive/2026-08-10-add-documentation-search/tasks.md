## 1. DocStore iteration support

- [x] 1.1 Add `Keys() []string` method to `DocStoreReader` interface in `internal/docs/store.go`
- [x] 1.2 Implement `Keys()` on `DocStore` — allocate a fresh slice, copy keys from the map
- [x] 1.3 Write unit test for `Keys()`: verify length matches `Size()`, verify keys are valid, verify slice is a copy (mutating returned slice doesn't affect store)
- [x] 1.4 Add `.rst` test file to `internal/docs/testdata/store/` and verify `DocStore.Load` loads it
- [x] 1.5 Update `DocStore.Load` test to assert both `.md`, `.rst`, and `.txt` files are loaded
- [x] 1.6 Re-run `go tool mockery` to regenerate `internal/docs/store_mock.go` with the new `Keys()` method
- [x] 1.7 Verify existing tests still pass: `go test ./internal/...`

## 2. Search engine

- [x] 2.1 Create `internal/search/scan.go` with `Searcher` struct and `Search()` method
- [x] 2.2 Implement tokenizer: lowercased, whitespace-split, stop-word removal
- [x] 2.3 Implement token coverage scoring: `matchedTokens / totalTokens`
- [x] 2.4 Implement heading boost: detect `#`/`===`/`---` heading lines, 3x weight
- [x] 2.5 Implement section boost: `classes/` → 2.0x, `tutorials/` → 1.5x, other → 1.0x
- [x] 2.6 Implement proximity bonus: sliding window of 500 chars, match density → 0.5–1.5x
- [x] 2.7 Implement snippet extraction: ±100 chars around first/highest-density match region, strip Markdown markup
- [x] 2.8 Implement version filter: skip keys not matching the requested version prefix

## 3. Search engine tests

- [x] 3.1 Write `TestSearch_SingleToken` — one token, verify top result is correct file
- [x] 3.2 Write `TestSearch_MultiTokenScore` — two tokens, verify both-match > single-match
- [x] 3.3 Write `TestSearch_HeadingBoost` — token in heading scores higher than same token in body
- [x] 3.4 Write `TestSearch_SectionBoost` — `classes/` file ranks above `about/` file for same tokens
- [x] 3.5 Write `TestSearch_ProximityBonus` — tokens close together score higher than tokens far apart
- [x] 3.6 Write `TestSearch_VersionFilter` — only matching version docs returned
- [x] 3.7 Write `TestSearch_NoResults` — query with no matches returns empty list
- [x] 3.8 Write `TestSearch_SnippetContainsContext` — snippet includes surrounding text

## 4. MCP tool integration

- [x] 4.1 Define search tool args struct (`query`, `version`, `limit`) with jsonschema tags
- [x] 4.2 Implement `search_documentation` handler: validate args, call searcher, format results as JSON
- [x] 4.3 Register `search_documentation` tool in `RegisterTools()` alongside existing tools
- [x] 4.4 Add tool description including usage tips for LLM context
- [x] 4.5 Write `TestSearchDocumentation_Success` — mock searcher, verify JSON output contains expected fields
- [x] 4.6 Write `TestSearchDocumentation_EmptyResults` — mock returns no results, verify graceful message
- [x] 4.7 Write `TestSearchDocumentation_LimitEnforcement` — verify limit is capped at 50, defaults to 10

## 5. Server wiring

- [x] 5.1 Instantiate `Searcher` in `cmd/godot-mcp-server/main.go` after doc store load
- [x] 5.2 Pass `Searcher` (or search function) to `tools.RegisterTools()`
- [x] 5.3 Verify server starts and the new tool appears in `tools/list`
- [x] 5.4 Run `go vet ./...` and `go build ./cmd/godot-mcp-server` to verify no issues

## 6. Integration tests

- [x] 6.1 Load real testdata docs into `DocStore`, run search, verify results have expected keys
- [x] 6.2 Verify search respects version filter with multi-version testdata

## 7. RST fallback and Docker fixes

- [x] 7.1 Update `scripts/generate-docs.sh` cleanup to preserve `.rst` files: `! -name '*.rst'`
- [x] 7.2 Add `tree` package to Dockerfile runtime stage: `apk add tree`
- [ ] 7.3 Verify converted docs include `.rst` fallback files for the 6 known pandoc failures

## 8. Final verification

- [x] 8.1 Run full test suite: `go test ./internal/... -v -count=1`
- [x] 8.2 Verify no new dependencies: `go list -m all | wc -l` should not increase
- [x] 8.3 Verify binary size unchanged: `go build -o /tmp/server ./cmd/godot-mcp-server && ls -lh /tmp/server`
