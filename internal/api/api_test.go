package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/mux"
)

func TestBasicAuthIsRejectedWhenLocalAuthDisabled(t *testing.T) {
	t.Setenv("DISABLE_LOCAL_AUTH", "1")
	t.Setenv("OAUTH2_PROVIDER", "oidc")
	t.Setenv("OAUTH2_CLIENT_ID", "client")
	t.Setenv("OAUTH2_CLIENT_SECRET", "secret")
	t.Setenv("OAUTH2_REDIRECT_URL", "https://example.org/oauth2/oidc/callback")
	t.Setenv("OAUTH2_OIDC_DISCOVERY_ENDPOINT", "https://example.org")

	require.NoError(t, config.Load(""))

	handler := mux.New()
	Serve(handler, nil, nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	r.SetBasicAuth("admin", "password")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	resp := w.Result()
	t.Cleanup(func() { resp.Body.Close() })

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
		"Unexpected status code")
}
