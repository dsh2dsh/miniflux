// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package sanitizer // import "miniflux.app/v2/internal/reader/sanitizer"

import (
	"errors"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// StripTags removes all HTML/XML tags from the input string.
// This function must *only* be used for cosmetic purposes, not to prevent code injections like XSS.
func StripTags(input string) string {
	tokenizer := html.NewTokenizer(strings.NewReader(input))
	var buffer strings.Builder

	for {
		if tokenizer.Next() == html.ErrorToken {
			err := tokenizer.Err()
			if errors.Is(err, io.EOF) {
				return buffer.String()
			}

			return ""
		}

		token := tokenizer.Token()
		if token.Type == html.TextToken {
			buffer.WriteString(token.Data)
		}
	}
}
