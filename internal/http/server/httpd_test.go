package server

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"miniflux.app/v2/internal/config"
)

func TestNewHTTPServerConfiguresTimeouts(t *testing.T) {
	require.NoError(t, config.Load(""))
	timeout := config.HTTPServerTimeout()
	want := http.Server{
		ReadTimeout:       timeout,
		ReadHeaderTimeout: timeout,
		WriteTimeout:      timeout,
		IdleTimeout:       timeout,
	}
	assert.EqualExportedValues(t, &want, newHttpServer())
}
