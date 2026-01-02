package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"miniflux.app/v2/internal/model"
)

func TestNewAuthors_Match(t *testing.T) {
	tests := []struct {
		name    string
		author  string
		blocked bool
	}{
		{
			name: "empty author",
		},
		{
			author:  "Team AA",
			blocked: true,
		},
		{
			author:  "Promoted",
			blocked: true,
		},
		{
			author: "John Doe",
		},
	}

	block := NewAuthors([]string{"Team AA", "Promoted"})

	for _, tt := range tests {
		name := tt.name
		if name == "" {
			name = tt.author
		}

		t.Run(name, func(t *testing.T) {
			entry := model.Entry{
				Author: tt.author,
				URL:    "http://example.com/entry",
			}
			assert.Equal(t, tt.blocked, block.Match(&entry))
		})
	}
}
