package fetcher

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_contextWithValue(t *testing.T) {
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
		"http://localhost", nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	r := requestContext{}
	ctx := contextWithRequest(req.Context(), &r)

	assert.Panics(t, func() { requestFromContext(ctx) })
	r.client = new(Client)

	assert.Panics(t, func() { requestFromContext(ctx) })
	r.req = req

	require.NotPanics(t, func() { requestFromContext(ctx) })
	got := requestFromContext(ctx)
	require.NotNil(t, got)
	assert.Same(t, &r, got)
	assert.Same(t, req, got.req)
}
