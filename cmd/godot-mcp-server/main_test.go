package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testToken is an obviously-fake fixture value. Never use a real secret here.
const testToken = "unit-test-token-not-a-real-secret"

// okHandler is a minimal upstream handler the auth middleware wraps.
func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func authedHandler(token string) http.Handler {
	return staticTokenAuthMiddleware(token)(http.HandlerFunc(okHandler))
}

func doAuthedRequest(middleware http.Handler, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	return rec
}

func TestAuth_ValidTokenAccepted(t *testing.T) {
	rec := doAuthedRequest(authedHandler(testToken), testToken)
	assert.Equal(t, http.StatusOK, rec.Code, "valid static token must be accepted")
	assert.Equal(t, "ok", rec.Body.String())
}

func TestAuth_InvalidTokenRejected(t *testing.T) {
	rec := doAuthedRequest(authedHandler(testToken), "wrong-token")
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "invalid token must be rejected")
}

func TestAuth_MissingTokenRejected(t *testing.T) {
	rec := doAuthedRequest(authedHandler(testToken), "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "missing Authorization header must be rejected")
}

func TestAuth_MalformedHeaderRejected(t *testing.T) {
	mw := authedHandler(testToken)
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Basic not-a-bearer-token")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "malformed auth header must be rejected")
}

// TestAuth_StaticTokenNoExpiration is the regression test for the bug where
// every request returned 401 "token missing expiration". The static token
// TokenInfo carries no Expiration, so AllowMissingExpiration must be set.
func TestAuth_StaticTokenNoExpiration(t *testing.T) {
	rec := doAuthedRequest(authedHandler(testToken), testToken)
	require.Equal(t, http.StatusOK, rec.Code,
		"static token without expiration must be accepted (AllowMissingExpiration regression)")
}
