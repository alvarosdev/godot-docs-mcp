package server

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear env to get defaults
	for _, k := range []string{"FASTMCP_TRANSPORT", "FASTMCP_HOST", "FASTMCP_PORT", "FASTMCP_PATH", "MCP_AUTH_TOKEN", "DOCS_DIR"} {
		t.Setenv(k, "")
	}

	cfg := LoadConfig()
	assert.Equal(t, "http", cfg.Transport)
	assert.Equal(t, "127.0.0.1", cfg.Host)
	assert.Equal(t, "8000", cfg.Port)
	assert.Equal(t, "", cfg.Path)
	assert.Equal(t, "", cfg.AuthToken)
	assert.Equal(t, "docs", cfg.DocsRoot)
}

func TestLoadConfig_Custom(t *testing.T) {
	t.Setenv("FASTMCP_TRANSPORT", "stdio")
	t.Setenv("FASTMCP_HOST", "0.0.0.0")
	t.Setenv("FASTMCP_PORT", "3000")
	t.Setenv("FASTMCP_PATH", "/custom")
	t.Setenv("MCP_AUTH_TOKEN", "test-token")
	t.Setenv("DOCS_DIR", "/custom/docs")

	cfg := LoadConfig()
	assert.Equal(t, "stdio", cfg.Transport)
	assert.Equal(t, "0.0.0.0", cfg.Host)
	assert.Equal(t, "3000", cfg.Port)
	assert.Equal(t, "/custom", cfg.Path)
	assert.Equal(t, "test-token", cfg.AuthToken)
	assert.Equal(t, "/custom/docs", cfg.DocsRoot)
}

func TestLoadConfig_AuthToken(t *testing.T) {
	t.Run("token set", func(t *testing.T) {
		t.Setenv("MCP_AUTH_TOKEN", "secret-123")
		cfg := LoadConfig()
		assert.Equal(t, "secret-123", cfg.AuthToken)
	})

	t.Run("token empty", func(t *testing.T) {
		t.Setenv("MCP_AUTH_TOKEN", "")
		cfg := LoadConfig()
		assert.Empty(t, cfg.AuthToken)
	})
}

func TestNew_CreatesServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	srv := New(logger)

	require.NotNil(t, srv)
}
