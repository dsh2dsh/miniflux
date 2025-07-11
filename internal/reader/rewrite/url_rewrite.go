// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rewrite // import "miniflux.app/v2/internal/reader/rewrite"

import (
	"context"
	"log/slog"
	"regexp"

	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
)

var customReplaceRuleRegex = regexp.MustCompile(
	`^rewrite\("([^"]+)"\|"([^"]+)"\)$`)

func RewriteEntryURL(ctx context.Context, feed *model.Feed, entry *model.Entry,
) {
	if feed.UrlRewriteRules == "" {
		return
	}

	log := logging.FromContext(ctx)
	parts := customReplaceRuleRegex.FindStringSubmatch(feed.UrlRewriteRules)
	if len(parts) < 3 {
		log.Debug("Cannot find search and replace terms for replace rule",
			slog.String("entry_url", entry.URL),
			slog.Int64("feed_id", feed.ID),
			slog.String("feed_url", feed.FeedURL),
			slog.String("url_rewrite_rules", feed.UrlRewriteRules))
		return
	}

	re, err := regexp.Compile(parts[1])
	if err != nil {
		log.Error("Failed on regexp compilation",
			slog.String("url_rewrite_rules", feed.UrlRewriteRules),
			slog.Any("error", err))
		return
	}

	rewrittenURL := re.ReplaceAllString(entry.URL, parts[2])
	log.Debug("Rewriting entry URL",
		slog.String("original_entry_url", entry.URL),
		slog.String("rewritten_entry_url", rewrittenURL),
		slog.Int64("feed_id", feed.ID),
		slog.String("feed_url", feed.FeedURL))
	entry.URL = rewrittenURL
}
