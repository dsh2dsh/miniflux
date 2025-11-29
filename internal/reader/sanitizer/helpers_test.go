package sanitizer

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripTracking(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		baseUrl  string
		feedUrl  string
	}{
		{
			name:     "URL with tracking parameters",
			input:    "https://example.com/page?id=123&utm_source=newsletter&utm_medium=email&fbclid=abc123",
			expected: "https://example.com/page?id=123",
		},
		{
			name:     "URL with only tracking parameters",
			input:    "https://example.com/page?utm_source=newsletter&utm_medium=email",
			expected: "https://example.com/page",
		},
		{
			name:     "URL with no tracking parameters",
			input:    "https://example.com/page?id=123&foo=bar",
			expected: "https://example.com/page?id=123&foo=bar",
		},
		{
			name:     "URL with no parameters",
			input:    "https://example.com/page",
			expected: "https://example.com/page",
		},
		{
			name:     "URL with mixed case tracking parameters",
			input:    "https://example.com/page?UTM_SOURCE=newsletter&utm_MEDIUM=email",
			expected: "https://example.com/page",
		},
		{
			name:     "URL with tracking parameters and fragments",
			input:    "https://example.com/page?id=123&utm_source=newsletter#section1",
			expected: "https://example.com/page?id=123#section1",
		},
		{
			name:     "URL with only tracking parameters and fragments",
			input:    "https://example.com/page?utm_source=newsletter#section1",
			expected: "https://example.com/page#section1",
		},
		{
			name:     "URL with only one tracking parameter",
			input:    "https://example.com/page?utm_source=newsletter",
			expected: "https://example.com/page",
		},
		{
			name:     "URL with encoded characters",
			input:    "https://example.com/page?name=John%20Doe&utm_source=newsletter",
			expected: "https://example.com/page?name=John+Doe",
		},
		{
			name:     "ref parameter for another url",
			input:    "https://example.com/page?ref=test.com",
			baseUrl:  "https://example.com/page",
			expected: "https://example.com/page?ref=test.com",
		},
		{
			name:     "ref parameter for feed url",
			input:    "https://example.com/page?ref=feed.com",
			baseUrl:  "https://example.com/page",
			expected: "https://example.com/page",
			feedUrl:  "http://feed.com",
		},
		{
			name:     "ref parameter for site url",
			input:    "https://example.com/page?ref=example.com",
			baseUrl:  "https://example.com/page",
			expected: "https://example.com/page",
		},
		{
			name:     "ref parameter for base url",
			input:    "https://example.com/page?ref=example.com",
			expected: "https://example.com/page",
			baseUrl:  "https://example.com",
			feedUrl:  "https://feedburned.com/example",
		},
		{
			name:     "ref parameter for base url on subdomain",
			input:    "https://blog.exploits.club/some-path?ref=blog.exploits.club",
			expected: "https://blog.exploits.club/some-path",
			baseUrl:  "https://blog.exploits.club/some-path",
			feedUrl:  "https://feedburned.com/exploit.club",
		},
		{
			name:     "Non-standard URL parameter with no tracker",
			input:    "https://example.com/foo.jpg?crop/1420x708/format/webp",
			expected: "https://example.com/foo.jpg?crop/1420x708/format/webp",
			baseUrl:  "https://example.com/page",
		},
		{
			name:     "Matomo tracking URL",
			input:    "https://example.com/?mtm_campaign=2020_august_promo&mtm_source=newsletter&mtm_medium=email&mtm_content=primary-cta",
			expected: "https://example.com/",
			baseUrl:  "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, err := url.Parse(tt.baseUrl)
			require.NoError(t, err)

			feedURL, _ := url.Parse(tt.feedUrl)
			require.NoError(t, err)

			u, err := url.Parse(tt.input)
			require.NoError(t, err)

			StripTracking(u, baseURL.Hostname(), feedURL.Hostname())
			assert.Equal(t, tt.expected, u.String())
		})
	}
}
