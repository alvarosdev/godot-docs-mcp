## 1. Dependencies and tooling

- [ ] 1.1 Add `github.com/stretchr/testify` to `go.mod`: `go get github.com/stretchr/testify@latest`
- [ ] 1.2 Add mockery as a `tool` in `go.mod`: `go get -tool github.com/vektra/mockery/v3@latest`
- [ ] 1.3 Create `.mockery.yml` at project root with testify template, inpackage mode, `DocStoreReader` interface
- [ ] 1.4 Verify `go tool mockery` runs successfully (generates mocks)

## 2. DocStoreReader interface

- [ ] 2.1 Define `DocStoreReader` interface in `internal/docs/store.go` with `Get(key string) (string, bool)` and `Size() int`
- [ ] 2.2 Run `go tool mockery` to generate `internal/docs/store_mock.go`
- [ ] 2.3 Verify `MockDocStoreReader` compiles and implements the interface

## 3. Refactor consumers to accept interface

- [ ] 3.1 Change `tools.RegisterTools` signature: `*docs.DocStore` → `docs.DocStoreReader`
- [ ] 3.2 Change `tools.makeGetDocumentationTree` parameter: `*docs.DocStore` → `docs.DocStoreReader`
- [ ] 3.3 Change `tools.makeGetDocumentationFile` parameter: `*docs.DocStore` → `docs.DocStoreReader`
- [ ] 3.4 Change `resources.makeDocResourceHandler` parameter: `*docs.DocStore` → `docs.DocStoreReader`
- [ ] 3.5 Verify `cmd/godot-mcp-server` compiles unchanged (`*DocStore` satisfies interface implicitly)
- [ ] 3.6 Run existing tests to verify no regressions: `go test ./internal/...`

## 4. Rewrite docs package tests with testify

- [ ] 4.1 Rewrite `internal/docs/docs_test.go`: replace manual `t.Errorf` with `assert.Equal` and `require.NoError`
- [ ] 4.2 Add `TestNewDocStore_Empty` — verify new store has Size() == 0
- [ ] 4.3 Verify all docs tests pass: `go test ./internal/docs/ -v`

## 5. Rewrite tools package tests with testify + mock

- [ ] 5.1 Delete `internal/tools/testdata/` directory
- [ ] 5.2 Rewrite `internal/tools/navigation_test.go` using testify assertions
- [ ] 5.3 Add `TestGetAvailableVersions_NilMeta` using testify
- [ ] 5.4 Add `TestGetAvailableVersions_Success` using testify
- [ ] 5.5 Add `TestGetDocumentationTree_Latest` — mock returns tree content
- [ ] 5.6 Add `TestGetDocumentationTree_SpecificVersion` — mock returns version-specific tree
- [ ] 5.7 Add `TestGetDocumentationTree_VersionNotFound` — mock returns not found
- [ ] 5.8 Add `TestGetDocumentationFile_Found` — mock returns file content
- [ ] 5.9 Add `TestGetDocumentationFile_InvalidPath` — validate rejects traversal
- [ ] 5.10 Add `TestGetDocumentationFile_NotFound` — mock returns not found
- [ ] 5.11 Verify all tools tests pass: `go test ./internal/tools/ -v`

## 6. Rewrite resources package tests with testify + mock

- [ ] 6.1 Delete `internal/resources/testdata/` directory
- [ ] 6.2 Rewrite `internal/resources/docs_test.go` using testify assertions
- [ ] 6.3 Add `TestResourceHandler_WithVersionPrefix` — mock returns content for "4.7/classes/node.md"
- [ ] 6.4 Add `TestResourceHandler_WithoutVersionPrefix` — mock returns content for latest version
- [ ] 6.5 Add `TestResourceHandler_NotFound` — handler returns resource-not-found error
- [ ] 6.6 Add `TestResourceHandler_TraversalPath` — handler rejects path with `..`
- [ ] 6.7 Retain `TestParseVersionPrefix*` tests (pure logic, no mock needed)
- [ ] 6.8 Verify all resources tests pass: `go test ./internal/resources/ -v`

## 7. Add server package tests

- [ ] 7.1 Create `internal/server/server_test.go` with testify
- [ ] 7.2 Add `TestLoadConfig_Defaults` — verify default values when env vars are unset
- [ ] 7.3 Add `TestLoadConfig_Custom` — set env vars via `t.Setenv`, verify all fields
- [ ] 7.4 Add `TestLoadConfig_AuthToken` — verify token is captured from env
- [ ] 7.5 Add `TestNew_CreatesServer` — verify `mcp.Server` is created with correct name
- [ ] 7.6 Verify server tests pass: `go test ./internal/server/ -v`

## 8. Project cleanup and rebranding

- [ ] 8.1 Delete `deploy/Dockerfile` — obsolete Python Dockerfile, replaced by root `Dockerfile`
- [ ] 8.2 Delete `deploy/` directory if empty after Dockerfile removal
- [ ] 8.3 Update `LICENSE`: replace `Copyright (c) 2025 Nihilantropy` with `Copyright (c) 2026 Álvaro G.`
- [ ] 8.4 Update `README.md`: replace all `godot-mcp-docs-http` references with `godot-docs-mcp` (clone URL, image URLs, Docker pull commands)
- [ ] 8.5 Add attribution line in `README.md` acknowledging Nihilantropy's original Python MCP server as the idea base for this project
- [ ] 8.6 Change git remote: `git remote set-url origin git@github.com:alvarosdev/godot-docs-mcp.git`
- [ ] 8.7 Remove fork relationship — verify no fork references remain in repo metadata or docs

## 9. Final verification

- [ ] 9.1 Run `go mod tidy` to finalize dependencies
- [ ] 9.2 Run full test suite: `go test ./internal/... -v -count=1`
- [ ] 9.3 Run `go build ./cmd/godot-mcp-server` to verify no compilation issues
- [ ] 9.4 Run `go vet ./...` to verify no static analysis warnings
- [ ] 9.5 Verify no remaining references to old repo name: `grep -r "godot-mcp-docs-http\|Nihilantropy" --include="*.go" --include="*.md" --include="*.yml" --include="*.yaml" --include="Dockerfile" --include="*.sh" .`
