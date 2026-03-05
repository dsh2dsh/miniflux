package fetcher

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextWithRequest(t *testing.T) {
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
		"http://localhost", nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	ctx := contextWithRequest(req.Context(), req)
	assert.Same(t, req, requestFromContext(ctx))
}
