// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package filter // import "miniflux.app/v2/internal/reader/filter"

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
)

func DeleteEntries(ctx context.Context, user *model.User, feed *model.Feed,
) error {
	block, keep, err := feedRules(user, feed)
	if err != nil {
		return fmt.Errorf("reader/filter: feed rules: %w", err)
	}

	log := logging.FromContext(ctx).With(slog.GroupAttrs("feed",
		slog.Int64("id", feed.ID),
		slog.String("url", feed.FeedURL)))
	block = block.WithLogger(log.With(slog.String("filter_action", "block")))
	keep = keep.WithLogger(log.With(slog.String("filter_action", "allow")))

	maxAge := time.Duration(config.Opts.FilterEntryMaxAgeDays()) * 24 * time.Hour
	seen := makeUniqEntries(feed)

	feed.Entries = slices.DeleteFunc(feed.Entries,
		func(e *model.Entry) bool {
			switch {
			case blockedGlobally(ctx, e, maxAge):
				feed.IncRemovedByAge()
				return true
			case block.Match(e) || !keep.Allow(e):
				feed.IncRemovedByFilters()
				return true
			case !seen.Add(e, log):
				return true
			}
			return false
		})
	return nil
}

func feedRules(user *model.User, feed *model.Feed) (*Filter, *Filter, error) {
	block, err := joinRules(user.BlockFilterEntryRules,
		feed.BlockFilterEntryRules())
	if err != nil {
		return nil, nil, fmt.Errorf("building block filter: %w", err)
	}

	keep, err := joinRules(user.KeepFilterEntryRules,
		feed.KeepFilterEntryRules())
	if err != nil {
		return nil, nil, fmt.Errorf("building keep filter: %w", err)
	}
	return block, keep, nil
}

func joinRules(userRules, feedRules string) (*Filter, error) {
	userFilter, err := New(userRules)
	if err != nil {
		return nil, fmt.Errorf("bad user rules: %w", err)
	}

	feedFilter, err := New(feedRules)
	if err != nil {
		return nil, fmt.Errorf("bad feed rules: %w", err)
	}
	return userFilter.Concat(feedFilter), nil
}

func blockedGlobally(ctx context.Context, entry *model.Entry,
	maxAge time.Duration,
) bool {
	if maxAge == 0 {
		return false
	}

	if entry.Date.Add(maxAge).Before(time.Now()) {
		logging.FromContext(ctx).Debug("Entry is blocked globally due to max age",
			slog.String("entry_url", entry.URL),
			slog.Time("entry_date", entry.Date),
			slog.Duration("max_age", maxAge))
		return true
	}
	return false
}

func matchDatePattern(pattern string, entryDate time.Time) bool {
	if pattern == "future" {
		return entryDate.After(time.Now())
	}

	ruleType, inputDate, found := strings.Cut(pattern, ":")
	if !found {
		return false
	}

	switch strings.ToLower(ruleType) {
	case "before":
		targetDate, err := time.Parse("2006-01-02", inputDate)
		if err != nil {
			return false
		}
		return entryDate.Before(targetDate)
	case "after":
		targetDate, err := time.Parse("2006-01-02", inputDate)
		if err != nil {
			return false
		}
		return entryDate.After(targetDate)
	case "between":
		d1, d2, found := strings.Cut(inputDate, ",")
		if !found {
			return false
		}
		startDate, err := time.Parse("2006-01-02", d1)
		if err != nil {
			return false
		}
		endDate, err := time.Parse("2006-01-02", d2)
		if err != nil {
			return false
		}
		return entryDate.After(startDate) && entryDate.Before(endDate)
	case "max-age":
		duration, err := parseDuration(inputDate)
		if err != nil {
			return false
		}
		cutoffDate := time.Now().Add(-duration)
		return entryDate.Before(cutoffDate)
	}
	return false
}

func parseDuration(duration string) (time.Duration, error) {
	// Handle common duration formats like "30d", "7d", "1h", "1m", etc. Go's
	// time.ParseDuration doesn't support days, so we handle them manually.
	if s, ok := strings.CutSuffix(duration, "d"); ok {
		days, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("reader/filter: parse days %q: %w", duration, err)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	// For other durations (hours, minutes, seconds), use Go's built-in parser.
	d, err := time.ParseDuration(duration)
	if err != nil {
		return 0, fmt.Errorf("reader/filter: parse duration %q: %w", duration, err)
	}
	return d, nil
}
