// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package sanitizer

import "strings"

func TruncateHTML(input string, maxLen int) string {
	text := StripTags(input)

	// Collapse multiple spaces into a single space
	text = strings.Join(strings.Fields(text), " ")

	// Convert to runes to be safe with unicode
	runes := []rune(text)
	if len(runes) > maxLen {
		return strings.TrimSpace(string(runes[:maxLen])) + "…"
	}

	return text
}
