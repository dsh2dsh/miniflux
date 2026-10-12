// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rewrite

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"

	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
)

var customReplaceRuleRegex = regexp.MustCompile(
	`^rewrite\("([^"]+)"\|"([^"]+)"\)$`)

func RewriteEntryURL(ctx context.Context, feed *model.Feed, entry *model.Entry,
) error {
	rules := strings.TrimSpace(feed.UrlRewriteRules)
	if rules == "" {
		return nil
	}

	parts := customReplaceRuleRegex.FindStringSubmatch(rules)
	if len(parts) < 3 {
		return fmt.Errorf("rewrite: too few parts: %d", len(parts))
	}

	re, err := regexp.Compile(parts[1])
	if err != nil {
		return fmt.Errorf("rewrite: compile rule re: %w", err)
	}

	newURL := re.ReplaceAllString(entry.URL, parts[2])
	if newURL == entry.URL {
		return nil
	}

	logging.FromContext(ctx).Debug("Rewriting entry URL",
		slog.String("original_entry_url", entry.URL),
		slog.String("rewritten_entry_url", newURL),
		slog.Int64("feed_id", feed.ID),
		slog.String("feed_url", feed.FeedURL))

	u, err := url.Parse(newURL)
	if err != nil {
		return fmt.Errorf("rewrite: parse rewritten URL: %w", err)
	}

	entry.WithURL(u)
	return nil
}
