// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import "strings"

// maxLanguageLen bounds accepted language tags. RFC 5646 recommends supporting
// tags of at least 35 characters; anything much longer is garbage.
const maxLanguageLen = 50

// normalizeLanguage cleans up a language tag declared by a feed so it is
// suitable for use as an HTML lang attribute. It trims surrounding whitespace,
// lower-cases the value, and replaces underscores with hyphens (e.g. "en_US" ->
// "en-us"). No strict BCP-47 validation is performed: many real feeds use loose
// values and silently dropping them yields worse downstream behaviour than
// passing them through.
//
// The value is feed-controlled and is persisted and rendered as-is, so anything
// longer than maxLength is rejected: such a value carries no usable language
// information, and stripping bad characters could assemble a wrong tag.
func normalizeLanguage(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > maxLanguageLen {
		return ""
	}
	return strings.ReplaceAll(strings.ToLower(s), "_", "-")
}
