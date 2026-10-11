package fetcher

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_withProtoMinorVersion(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "full version",
			in:   "HTTP/2.0 200 OK",
			want: "HTTP/2.0 200 OK",
		},
		{
			name: "without minor",
			in:   "HTTP/2 200 OK",
			want: "HTTP/2.0 200 OK",
		},
		{
			name: "empty",
		},
		{
			name: "short",
			in:   "HTT",
			want: "HTT",
		},
		{
			name: "not http",
			in:   "foo bar baz",
			want: "foo bar baz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := withProtoMinorVersion(strings.NewReader(tt.in))
			require.NoError(t, err)
			require.NotNil(t, r)

			got, err := io.ReadAll(r)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}
