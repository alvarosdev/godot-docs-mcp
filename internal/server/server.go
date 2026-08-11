// Package server provides the Godot MCP server setup and configuration.
package server

import (
	"log/slog"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Config holds all runtime configuration read from environment variables.
type Config struct {
	Transport string
	Host      string
	Port      string
	Path      string
	AuthToken string
	DocsRoot  string
}

// LoadConfig reads configuration from environment variables.
// Call once at startup — never in hot paths.
func LoadConfig() Config {
	cfg := Config{
		Transport: envOrDefault("FASTMCP_TRANSPORT", "http"),
		Host:      envOrDefault("FASTMCP_HOST", "127.0.0.1"),
		Port:      envOrDefault("FASTMCP_PORT", "8000"),
		Path:      os.Getenv("FASTMCP_PATH"),
		AuthToken: os.Getenv("MCP_AUTH_TOKEN"),
		DocsRoot:  envOrDefault("DOCS_DIR", "docs"),
	}
	return cfg
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// New creates a new MCP server with all tools and resources registered.
// The caller must populate the doc store and register tools/resources
// before starting the server.
func New(logger *slog.Logger) *mcp.Server {
	opts := &mcp.ServerOptions{
		Instructions: "Use the Godot documentation tools to help with Godot game development. " +
			"Start with get_available_versions() to discover available Godot versions, " +
			"then use get_documentation_tree(version) for an overview and " +
			"get_documentation_file(path, version) to retrieve specific documentation.",
		Logger: logger,
		// KeepAlive detects abandoned sessions and cleans up goroutines
		// (mitigates SDK goroutine leak in streamable HTTP, issue #499).
		KeepAlive:                 5 * time.Minute, // detect abandoned sessions, release goroutines
		KeepAliveFailureThreshold: 3,
	}

	return mcp.NewServer(
		&mcp.Implementation{
			Name:    "godot-docs-server",
			Title:   "Godot Documentation MCP Server",
			Version: "1.0.0",
		},
		opts,
	)
}
