package logger

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiHandler(t *testing.T) {
	var b1, b2 bytes.Buffer
	h1 := NewHumanTextHandler(&b1, nil, false)
	require.NotNil(t, h1)
	h2 := NewHumanTextHandler(&b2, nil, false)
	require.NotNil(t, h2)

	l := slog.New(NewMultiHandler([]slog.Handler{h1, h2}))
	l.Info("Starting HTTP server")
	assert.Equal(t, "INFO Starting HTTP server\n", b1.String())
	assert.Equal(t, b1.String(), b2.String())
}

func TestMultiHandler_level(t *testing.T) {
	var b1, b2 bytes.Buffer
	h1 := NewHumanTextHandler(&b1,
		&slog.HandlerOptions{Level: slog.LevelWarn}, false)
	require.NotNil(t, h1)
	h2 := NewHumanTextHandler(&b2, nil, false)
	require.NotNil(t, h2)

	l := slog.New(NewMultiHandler([]slog.Handler{h1, h2}))
	l.Info("Starting HTTP server")
	assert.Zero(t, b1.Len())
	assert.Equal(t, "INFO Starting HTTP server\n", b2.String())

	b2.Reset()
	l.Warn("Starting HTTP server")
	assert.Equal(t, "WARN Starting HTTP server\n", b1.String())
	assert.Equal(t, b1.String(), b2.String())
}
