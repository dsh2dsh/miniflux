package template

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"miniflux.app/v2/internal/model"
)

func TestEntry_URLSafe(t *testing.T) {
	tests := []struct {
		rawURL   string
		expected bool
	}{
		// Allowed: web schemes.
		{
			rawURL:   "http://example.org/article",
			expected: true,
		},
		{
			rawURL:   "https://example.org/article",
			expected: true,
		},

		// Allowed: a sample of the broader feed-content schemes.
		{
			rawURL:   "mailto:author@example.org",
			expected: true,
		},
		{
			rawURL:   "magnet:?xt=urn:btih:abc",
			expected: true,
		},
		{
			rawURL:   "tel:+15551234567",
			expected: true,
		},
		{
			rawURL:   "ftp://example.org/file",
			expected: true,
		},
		{
			rawURL:   "feed:https://example.org/",
			expected: true,
		},
		{
			rawURL:   "webcal://example.org/cal",
			expected: true,
		},

		// Rejected: schemes that enable script execution or local resource access.
		{
			rawURL:   "javascript:alert(1)",
			expected: false,
		},
		{
			rawURL:   "data:text/html,<script>alert(1)</script>",
			expected: false,
		},
		{
			rawURL:   "vbscript:msgbox(1)",
			expected: false,
		},
		{
			rawURL:   "file:///etc/passwd",
			expected: false,
		},

		// Rejected: missing or malformed scheme.
		{
			rawURL:   "",
			expected: false,
		},
		{
			rawURL:   "example.org",
			expected: false,
		},
		{
			rawURL:   "/relative/path",
			expected: false,
		},
		{
			rawURL:   "//evil.example.org/path",
			expected: false,
		},

		// Allowed: scheme matching is case-insensitive (RFC 3986 §3.1).
		{
			rawURL:   "HTTPS://example.org",
			expected: true,
		},
		{
			rawURL:   "MailTo:author@host",
			expected: true,
		},
		{
			rawURL:   "SVN+SSH://example.org",
			expected: true,
		},

		// Rejected: case-insensitive match still rejects disallowed schemes.
		{
			rawURL:   "JavaScript:alert(1)",
			expected: false,
		},
		{
			rawURL:   "VBScript:msgbox(1)",
			expected: false,
		},
	}

	for _, tt := range tests {
		entry := model.Entry{URL: tt.rawURL}
		wrappedEntry := Entry{Entry: &entry}
		assert.Equal(t, tt.expected, wrappedEntry.URLSafe(),
			"Unexpected result for URL %q", tt.rawURL)
	}
}
