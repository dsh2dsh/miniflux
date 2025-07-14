package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"miniflux.app/v2/internal/model"
)

func Test_sanitizeTitle(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "plain text",
			title: "Foo bar baz",
			want:  "Foo bar baz",
		},
		{
			name:  "with html",
			title: "Foo <string>bar</strong> baz",
			want:  "Foo bar baz",
		},
		{
			name:  "broken html",
			title: "Foo <string>bar baz",
			want:  "Foo bar baz",
		},
		{
			name:  "with spaces",
			title: " Foo bar <b>baz</b>",
			want:  "Foo bar baz",
		},
		{
			name:  "with br",
			title: "Foo\n<br>\nbar\n<br>\nbaz",
			want:  "Foo\n\nbar\n\nbaz",
		},
		{
			name:  "with entities",
			title: "&amp;Foo &lt; bar &gt; baz",
			want:  "&amp;Foo &lt; bar &gt; baz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := model.Entry{Title: tt.title}
			sanitizeTitle(&entry)
			assert.Equal(t, tt.want, entry.Title)
		})
	}
}
