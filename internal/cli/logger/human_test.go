package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHumanTextHandler(t *testing.T) {
	var b bytes.Buffer
	var called int

	tests := []struct {
		name    string
		h       slog.Handler
		wantStr string
		wantRe  string
		assert  func(t *testing.T)
	}{
		{
			name:    "without time",
			h:       NewHumanTextHandler(&b, nil, false),
			wantStr: "INFO Starting HTTP server listen_address=127.0.0.1:8080\n",
		},
		{
			name:   "with time",
			h:      NewHumanTextHandler(&b, nil, true),
			wantRe: `^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} INFO `,
		},
		{
			name: "with ReplaceAttr",
			h: NewHumanTextHandler(&b,
				&slog.HandlerOptions{
					ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
						called++
						return a
					},
				}, false),
			assert: func(t *testing.T) { assert.Equal(t, 1, called) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotNil(t, tt.h)
			b.Reset()
			l := slog.New(tt.h)
			l.Info("Starting HTTP server",
				slog.String("listen_address", "127.0.0.1:8080"))
			t.Log(strings.TrimSpace(b.String()))
			switch {
			case tt.assert != nil:
				tt.assert(t)
			case tt.wantStr != "":
				assert.Equal(t, tt.wantStr, b.String())
			case tt.wantRe != "":
				assert.Regexp(t, tt.wantRe, b.String())
			}
		})
	}
}
