package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"miniflux.app/v2/internal/config"
)

func TestRequestBuilderWithJSON(t *testing.T) {
	t.Setenv("INTEGRATION_ALLOW_PRIVATE_NETWORKS", "1")
	require.NoError(t, config.Load(""))

	var gotMethod, gotContentType, gotUserAgent, gotAuth, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		gotUserAgent = r.Header.Get("User-Agent")
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(server.Close)

	response, err := NewRequestBuilder(server.URL).
		WithMethod(http.MethodPost).
		WithHeader("Authorization", "Bearer secret").
		WithJSON(map[string]string{"hello": "world"}).
		Do(t.Context())
	require.NoError(t, err, "request execution failed")
	t.Cleanup(response.Close)

	assert.Equal(t, http.StatusCreated, response.StatusCode(),
		"expected status http.StatusCreated")
	assert.Equal(t, http.MethodPost, gotMethod, "expected method POST")
	assert.Equal(t, "application/json", gotContentType,
		"expected Content-Type application/json")
	assert.Equal(t, config.HTTPClientUserAgent(), gotUserAgent,
		"unexpected User-Agent")
	assert.Equal(t, "Bearer secret", gotAuth, "unexpected Authorization")
	assert.JSONEq(t, `{"hello":"world"}`, gotBody, "unexpected body")
}

func TestRequestBuilderWithInvalidEndpoint(t *testing.T) {
	_, err := NewRequestBuilder("://invalid").
		WithMethod(http.MethodPost).
		WithJSON(nil).
		Do(t.Context())
	require.Error(t, err, "expected an error for an invalid endpoint")
}
