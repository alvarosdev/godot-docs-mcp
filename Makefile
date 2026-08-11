# Godot MCP Docs Server — build helpers
# All compiled artifacts go to dist/ (gitignored).

BINARY  := godot-mcp-server
DIST    := dist
BINPATH := $(DIST)/$(BINARY)

.PHONY: build build-arm64 clean test vet fmt run

build: ## Build the server binary into dist/
	mkdir -p $(DIST)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINPATH) ./cmd/godot-mcp-server
	@echo "✓ $(BINPATH)"

build-arm64: ## Cross-compile for linux/arm64
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(DIST)/$(BINARY)-arm64 ./cmd/godot-mcp-server
	@echo "✓ $(DIST)/$(BINARY)-arm64"

clean: ## Remove dist/
	rm -rf $(DIST)

test: ## Run all tests
	go test ./... -count=1

vet: ## Run go vet
	go vet ./...

fmt: ## Format all Go files
	gofmt -w ./cmd ./internal

run: build ## Build and run the server (HTTP, default config)
	FASTMCP_TRANSPORT=http ./$(BINPATH)
