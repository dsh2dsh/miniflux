// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeLanguage(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"   ", ""},
		{"en", "en"},
		{"EN", "en"},
		{"en_US", "en-us"},
		{"EN-us", "en-us"},
		{"pt-BR", "pt-br"},
		{"  fr-FR  ", "fr-fr"},
		{"zh-hant-cn-x-private1-private2", "zh-hant-cn-x-private1-private2"},

		// Values longer than 50 characters are rejected.
		{strings.Repeat("a", 51), ""},
		{"en-" + strings.Repeat("a", 100), ""},
	}
	for _, c := range cases {
		got := normalizeLanguage(c.in)
		assert.Equal(t, c.want, got, "NormalizeLanguage(%q)", c.in)
	}
}
