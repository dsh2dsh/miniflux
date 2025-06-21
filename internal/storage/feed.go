// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"

	"github.com/jackc/pgx/v5"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
)

type byStateAndName struct{ f model.Feeds }

func (l byStateAndName) Len() int      { return len(l.f) }
func (l byStateAndName) Swap(i, j int) { l.f[i], l.f[j] = l.f[j], l.f[i] }
func (l byStateAndName) Less(i, j int) bool {
	// disabled test first, since we don't care about errors if disabled
	if l.f[i].Disabled != l.f[j].Disabled {
		return l.f[j].Disabled
	}
	if l.f[i].ParsingErrorCount != l.f[j].ParsingErrorCount {
		return l.f[i].ParsingErrorCount > l.f[j].ParsingErrorCount
	}
	if l.f[i].UnreadCount != l.f[j].UnreadCount {
		return l.f[i].UnreadCount > l.f[j].UnreadCount
	}
	return l.f[i].Title < l.f[j].Title
}

// FeedExists checks if the given feed exists.
func (s *Storage) FeedExists(ctx context.Context, userID, feedID int64) bool {
	rows, _ := s.db.Query(ctx,
		`SELECT EXISTS(SELECT FROM feeds WHERE user_id=$1 AND id=$2)`,
		userID, feedID)

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		logging.FromContext(ctx).Error("storage: unable check feed exists",
			slog.Int64("user_id", userID),
			slog.Int64("feed_id", feedID),
			slog.Any("error", err))
		return false
	}
	return result
}

// CategoryFeedExists returns true if the given feed exists that belongs to the
// given category.
func (s *Storage) CategoryFeedExists(ctx context.Context, userID, categoryID,
	feedID int64,
) (bool, error) {
	rows, _ := s.db.Query(ctx, `
SELECT EXISTS(
  SELECT FROM feeds WHERE user_id=$1 AND category_id=$2 AND id=$3)`,
		userID, categoryID, feedID)

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		return false, fmt.Errorf("storage: unable check feed exists: %w", err)
	}
	return result, nil
}

// FeedURLExists checks if feed URL already exists.
func (s *Storage) FeedURLExists(ctx context.Context, userID int64,
	feedURL string,
) bool {
	rows, _ := s.db.Query(ctx, `
SELECT EXISTS(
  SELECT FROM feeds WHERE user_id=$1 AND feed_url=$2)`,
		userID, feedURL)

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		logging.FromContext(ctx).Error("storage: unable check feed url exists",
			slog.Int64("user_id", userID),
			slog.String("feed_url", feedURL),
			slog.Any("error", err))
		return false
	}
	return result
}

// AnotherFeedURLExists checks if the user a duplicated feed.
func (s *Storage) AnotherFeedURLExists(ctx context.Context,
	userID, feedID int64, feedURL string,
) bool {
	rows, _ := s.db.Query(ctx, `
SELECT EXISTS(
  SELECT FROM feeds WHERE id <> $1 AND user_id=$2 AND feed_url=$3)`,
		feedID, userID, feedURL)

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		logging.FromContext(ctx).Error(
			"storage: unable check another feed url exists",
			slog.Int64("user_id", userID),
			slog.Int64("feed_id", feedID),
			slog.String("feed_url", feedURL),
			slog.Any("error", err))
		return false
	}
	return result
}

// CountAllFeeds returns the number of feeds in the database.
func (s *Storage) CountAllFeeds(ctx context.Context) (map[string]int64, error) {
	rows, _ := s.db.Query(ctx,
		`SELECT disabled, count(*) FROM feeds GROUP BY disabled`)

	results := map[string]int64{"enabled": 0, "disabled": 0, "total": 0}

	var disabled bool
	var count int64
	_, err := pgx.ForEachRow(rows, []any{&disabled, &count}, func() error {
		if disabled {
			results["disabled"] = count
		} else {
			results["enabled"] = count
		}
		results["total"] += count
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("storage: count all feeds by disabled: %w", err)
	}
	return results, nil
}

// CountUserFeedsWithErrors returns the number of feeds with parsing errors that
// belong to the given user.
func (s *Storage) CountUserFeedsWithErrors(ctx context.Context, userID int64,
) int {
	limit := max(1, config.Opts.PollingParsingErrorLimit())
	rows, _ := s.db.Query(ctx, `
SELECT count(*) FROM feeds WHERE user_id=$1 AND parsing_error_count >= $2`,
		userID, limit)

	count, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int])
	if err != nil {
		logging.FromContext(ctx).Error(
			"storage: unable count user feeds with error",
			slog.Int64("user_id", userID), slog.Any("error", err))
		return 0
	}
	return count
}

// CountAllFeedsWithErrors returns the number of feeds with parsing errors.
func (s *Storage) CountAllFeedsWithErrors(ctx context.Context) (int, error) {
	limit := max(1, config.Opts.PollingParsingErrorLimit())
	rows, _ := s.db.Query(ctx,
		`SELECT count(*) FROM feeds WHERE parsing_error_count >= $1`, limit)

	count, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int])
	if err != nil {
		return 0, fmt.Errorf("storage: unable count all feeds with error: %w", err)
	}
	return count, nil
}

// Feeds returns all feeds that belongs to the given user.
func (s *Storage) Feeds(ctx context.Context, userID int64,
) (model.Feeds, error) {
	builder := s.NewFeedQueryBuilder(userID).
		WithSorting(model.DefaultFeedSorting, model.DefaultFeedSortingDirection)
	return builder.GetFeeds(ctx)
}

func getFeedsSorted(ctx context.Context, builder *FeedQueryBuilder,
) (model.Feeds, error) {
	result, err := builder.GetFeeds(ctx)
	if err != nil {
		return result, err
	}
	sort.Sort(byStateAndName{result})
	return result, nil
}

// FeedsWithCounters returns all feeds of the given user with counters of read
// and unread entries.
func (s *Storage) FeedsWithCounters(ctx context.Context, userID int64,
) (model.Feeds, error) {
	builder := s.NewFeedQueryBuilder(userID).
		WithCounters().
		WithSorting(model.DefaultFeedSorting, model.DefaultFeedSortingDirection)
	return getFeedsSorted(ctx, builder)
}

// Return read and unread count.
func (s *Storage) FetchCounters(ctx context.Context, userID int64,
) (model.FeedCounters, error) {
	builder := s.NewFeedQueryBuilder(userID).WithCounters()
	reads, unreads, err := builder.fetchFeedCounter(ctx)
	return model.FeedCounters{ReadCounters: reads, UnreadCounters: unreads}, err
}

// FeedsByCategoryWithCounters returns all feeds of the given user/category with
// counters of read and unread entries.
func (s *Storage) FeedsByCategoryWithCounters(ctx context.Context,
	userID, categoryID int64,
) (model.Feeds, error) {
	builder := s.NewFeedQueryBuilder(userID).
		WithCategoryID(categoryID).
		WithCounters().
		WithSorting(model.DefaultFeedSorting, model.DefaultFeedSortingDirection)
	return getFeedsSorted(ctx, builder)
}

// FeedByID returns a feed by the ID.
func (s *Storage) FeedByID(ctx context.Context, userID, feedID int64,
) (*model.Feed, error) {
	feed, err := s.NewFeedQueryBuilder(userID).GetFeedByID(ctx, feedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("unable to fetch feed #%d: %w", feedID, err)
	}
	return feed, nil
}

// CreateFeed creates a new feed.
func (s *Storage) CreateFeed(ctx context.Context, feed *model.Feed) error {
	if err := s.createFeed(ctx, feed); err != nil {
		return err
	} else if len(feed.Entries) == 0 {
		return nil
	}

	for _, entry := range feed.Entries {
		entry.FeedID = feed.ID
		entry.UserID = feed.UserID
	}

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		return s.createEntries(ctx, tx, feed.Entries)
	})
	if err != nil {
		return fmt.Errorf("unable to create entries(%v, feed #%d): %w",
			len(feed.Entries), feed.ID, err)
	}
	return nil
}

func (s *Storage) createFeed(ctx context.Context, feed *model.Feed) error {
	err := s.db.QueryRow(ctx, `
INSERT INTO feeds (
  feed_url,
  site_url,
  title,
  category_id,
  user_id,
  etag_header,
  last_modified_header,
  crawler,
  user_agent,
  cookie,
  username,
  password,
  disabled,
  scraper_rules,
  rewrite_rules,
  blocklist_rules,
  keeplist_rules,
  ignore_http_cache,
  allow_self_signed_certificates,
  fetch_via_proxy,
  hide_globally,
  url_rewrite_rules,
  no_media_player,
  apprise_service_urls,
  webhook_url,
  disable_http2,
  description,
  proxy_url,
  extra,
  runtime)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
        $17, $18, $19, $20, $21, $22,  $23, $24, $25, $26, $27, $28, $29, $30)
RETURNING id`,
		feed.FeedURL,
		feed.SiteURL,
		feed.Title,
		feed.Category.ID,
		feed.UserID,
		feed.EtagHeader,
		feed.LastModifiedHeader,
		feed.Crawler,
		feed.UserAgent,
		feed.Cookie,
		feed.Username,
		feed.Password,
		feed.Disabled,
		feed.ScraperRules,
		feed.RewriteRules,
		feed.BlocklistRules,
		feed.KeeplistRules,
		feed.IgnoreHTTPCache,
		feed.AllowSelfSignedCertificates,
		feed.FetchViaProxy,
		feed.HideGlobally,
		feed.UrlRewriteRules,
		feed.NoMediaPlayer,
		feed.AppriseServiceURLs,
		feed.WebhookURL,
		feed.DisableHTTP2,
		feed.Description,
		feed.ProxyURL,
		&feed.Extra,
		&feed.Runtime).Scan(&feed.ID)
	if err != nil {
		return fmt.Errorf(`store: unable to create feed %q: %w`, feed.FeedURL, err)
	}
	return nil
}

// UpdateFeed updates an existing feed.
func (s *Storage) UpdateFeed(ctx context.Context, feed *model.Feed) error {
	_, err := s.db.Exec(ctx, `
UPDATE feeds
SET
	feed_url = $1,
	site_url = $2,
	title = $3,
	category_id = $4,
	scraper_rules = $5,
	rewrite_rules = $6,
	blocklist_rules = $7,
	keeplist_rules = $8,
	crawler = $9,
	user_agent = $10,
	cookie = $11,
	username = $12,
	password = $13,
	disabled = $14,
	ignore_http_cache = $15,
	allow_self_signed_certificates = $16,
	fetch_via_proxy = $17,
	hide_globally = $18,
	url_rewrite_rules = $19,
	no_media_player = $20,
	apprise_service_urls = $21,
	webhook_url = $22,
	disable_http2 = $23,
	description = $24,
	ntfy_enabled = $25,
	ntfy_priority = $26,
	ntfy_topic = $27,
	pushover_enabled = $28,
	pushover_priority = $29,
	proxy_url = $30,
  extra = $31
WHERE id = $32 AND user_id = $33`,
		feed.FeedURL,
		feed.SiteURL,
		feed.Title,
		feed.Category.ID,
		feed.ScraperRules,
		feed.RewriteRules,
		feed.BlocklistRules,
		feed.KeeplistRules,
		feed.Crawler,
		feed.UserAgent,
		feed.Cookie,
		feed.Username,
		feed.Password,
		feed.Disabled,
		feed.IgnoreHTTPCache,
		feed.AllowSelfSignedCertificates,
		feed.FetchViaProxy,
		feed.HideGlobally,
		feed.UrlRewriteRules,
		feed.NoMediaPlayer,
		feed.AppriseServiceURLs,
		feed.WebhookURL,
		feed.DisableHTTP2,
		feed.Description,
		feed.NtfyEnabled,
		feed.NtfyPriority,
		feed.NtfyTopic,
		feed.PushoverEnabled,
		feed.PushoverPriority,
		feed.ProxyURL,
		&feed.Extra,
		feed.ID, feed.UserID)
	if err != nil {
		return fmt.Errorf("storage: unable to update feed #%d (%s): %w",
			feed.ID, feed.FeedURL, err)
	}
	return nil
}

func (s *Storage) UpdateFeedRuntime(ctx context.Context, feed *model.Feed,
) error {
	_, err := s.db.Exec(ctx, `
UPDATE feeds
SET
	etag_header = $1,
	last_modified_header = $2,
	checked_at = $3,
	parsing_error_msg = $4,
	parsing_error_count = $5,
	next_check_at = $6,
  runtime = $7
WHERE id = $8 AND user_id = $9`,
		feed.EtagHeader,
		feed.LastModifiedHeader,
		feed.CheckedAt,
		feed.ParsingErrorMsg,
		feed.ParsingErrorCount,
		feed.NextCheckAt,
		&feed.Runtime,
		feed.ID, feed.UserID)
	if err != nil {
		return fmt.Errorf("storage: unable to update feed runtime #%d (%s): %w",
			feed.ID, feed.FeedURL, err)
	}
	return nil
}

// IncFeedError updates feed errors.
func (s *Storage) IncFeedError(ctx context.Context, feed *model.Feed,
) error {
	rows, _ := s.db.Query(ctx, `
UPDATE feeds
   SET parsing_error_msg = $1,
       parsing_error_count = parsing_error_count + 1,
       checked_at = $2,
       next_check_at = $3
 WHERE id = $4 AND user_id = $5
RETURNING parsing_error_count`,
		feed.ParsingErrorMsg,
		feed.CheckedAt,
		feed.NextCheckAt,
		feed.ID, feed.UserID)

	errCount, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int])
	if err != nil {
		return fmt.Errorf("storage: unable to update feed error #%d (%s): %w",
			feed.ID, feed.FeedURL, err)
	}
	feed.ParsingErrorCount = errCount
	return nil
}

// RemoveFeed removes a feed and all entries.
func (s *Storage) RemoveFeed(ctx context.Context, userID, feedID int64) (bool,
	error,
) {
	var affected bool
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`DELETE FROM entries WHERE user_id=$1 AND feed_id=$2`, userID, feedID)
		if err != nil {
			return fmt.Errorf("delete entries: %w", err)
		}

		result, err := tx.Exec(ctx,
			`DELETE FROM feeds WHERE id=$1 AND user_id=$2`, feedID, userID)
		if err != nil {
			return fmt.Errorf("delete feed: %w", err)
		}
		affected = result.RowsAffected() != 0
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("storage: unable to delete feed #%d: %w", feedID,
			err)
	}
	return affected, nil
}

func (s *Storage) RemoveMultipleFeeds(ctx context.Context, userID int64,
	feedIDs []int64,
) error {
	logging.FromContext(ctx).Debug("Deleting multiple feeds",
		slog.Int64("user_id", userID),
		slog.Int("num_of_feeds", len(feedIDs)),
		slog.Any("feed_ids", feedIDs))

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`DELETE FROM entries WHERE user_id=$1 AND feed_id=ANY($2)`,
			userID, feedIDs)
		if err != nil {
			return fmt.Errorf("delete entries: %w", err)
		}

		_, err = tx.Exec(ctx,
			`DELETE FROM feeds WHERE id=ANY($1) AND user_id=$2`, feedIDs, userID)
		if err != nil {
			return fmt.Errorf("delete feed: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("storage: unable to delete multiple feeds(%d): %w",
			len(feedIDs), err)
	}
	return nil
}

// ResetFeedErrors removes all feed errors.
func (s *Storage) ResetFeedErrors(ctx context.Context) error {
	_, err := s.db.Exec(ctx,
		`UPDATE feeds SET parsing_error_count=0, parsing_error_msg=''`)
	if err != nil {
		return fmt.Errorf("storage: failed reset feed errors: %w", err)
	}
	return nil
}

func (s *Storage) ResetNextCheckAt(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `UPDATE feeds SET next_check_at=now()`)
	if err != nil {
		return fmt.Errorf("storage: failed reset next check: %w", err)
	}
	return nil
}
