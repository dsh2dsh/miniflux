// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/timezone"
)

// NewFeedQueryBuilder returns a new FeedQueryBuilder.
func (s *Storage) NewFeedQueryBuilder(userID int64) *FeedQueryBuilder {
	return &FeedQueryBuilder{
		db:                s.db,
		args:              []any{userID},
		conditions:        []string{"f.user_id = $1"},
		counterArgs:       []any{userID, model.EntryStatusRead, model.EntryStatusUnread},
		counterConditions: []string{"e.user_id = $1", "e.status IN ($2, $3)"},
	}
}

// FeedQueryBuilder builds a SQL query to fetch feeds.
type FeedQueryBuilder struct {
	db                *pgxpool.Pool
	args              []any
	conditions        []string
	sortExpressions   []string
	limit             int
	offset            int
	withCounters      bool
	counterJoinFeeds  bool
	counterArgs       []any
	counterConditions []string
}

// WithCategoryID filter by category ID.
func (f *FeedQueryBuilder) WithCategoryID(categoryID int64) *FeedQueryBuilder {
	if categoryID > 0 {
		f.conditions = append(f.conditions,
			"f.category_id = $"+strconv.Itoa(len(f.args)+1))
		f.args = append(f.args, categoryID)
		f.counterConditions = append(f.counterConditions,
			"f.category_id = $"+strconv.Itoa(len(f.counterArgs)+1))
		f.counterArgs = append(f.counterArgs, categoryID)
		f.counterJoinFeeds = true
	}
	return f
}

// WithCounters let the builder return feeds with counters of statuses of
// entries.
func (f *FeedQueryBuilder) WithCounters() *FeedQueryBuilder {
	f.withCounters = true
	return f
}

// WithSorting add a sort expression.
func (f *FeedQueryBuilder) WithSorting(column, direction string,
) *FeedQueryBuilder {
	f.sortExpressions = append(f.sortExpressions, column+" "+direction)
	return f
}

// WithLimit set the limit.
func (f *FeedQueryBuilder) WithLimit(limit int) *FeedQueryBuilder {
	f.limit = limit
	return f
}

// WithOffset set the offset.
func (f *FeedQueryBuilder) WithOffset(offset int) *FeedQueryBuilder {
	f.offset = offset
	return f
}

func (f *FeedQueryBuilder) buildCondition() string {
	return strings.Join(f.conditions, " AND ")
}

func (f *FeedQueryBuilder) buildCounterCondition() string {
	return strings.Join(f.counterConditions, " AND ")
}

func (f *FeedQueryBuilder) buildSorting() string {
	var parts string
	if len(f.sortExpressions) > 0 {
		parts = " ORDER BY " + strings.Join(f.sortExpressions, ", ") +
			", lower(f.title) ASC"
	}

	if f.limit > 0 {
		parts += " LIMIT " + strconv.FormatInt(int64(f.limit), 10)
	}

	if f.offset > 0 {
		parts += " OFFSET " + strconv.FormatInt(int64(f.offset), 10)
	}
	return parts
}

// GetFeedByID returns a single feed that match feedID.
func (f *FeedQueryBuilder) GetFeedByID(ctx context.Context, feedID int64,
) (*model.Feed, error) {
	f.conditions = append(f.conditions, "f.id = $"+strconv.Itoa(len(f.args)+1))
	f.args = append(f.args, feedID)

	feeds, err := f.queryFeeds(ctx)
	if err != nil {
		return nil, err
	} else if len(feeds) != 1 {
		return nil, nil
	}
	return feeds[0], nil
}

func (f *FeedQueryBuilder) queryFeeds(ctx context.Context) (model.Feeds, error,
) {
	rows, err := f.db.Query(ctx, `
SELECT
  f.id,
  f.feed_url,
  f.site_url,
  f.title,
  f.description,
  f.etag_header,
  f.last_modified_header,
  f.user_id,
  f.checked_at,
  f.next_check_at,
  f.parsing_error_count,
  f.parsing_error_msg,
  f.scraper_rules,
  f.rewrite_rules,
  f.url_rewrite_rules,
  f.blocklist_rules,
  f.keeplist_rules,
  f.crawler,
  f.user_agent,
  f.cookie,
  f.username,
  f.password,
  f.ignore_http_cache,
  f.allow_self_signed_certificates,
  f.fetch_via_proxy,
  f.disabled,
  f.no_media_player,
  f.hide_globally,
  f.category_id,
  c.title as category_title,
  c.hide_globally as category_hidden,
  fi.icon_id,
  i.external_id,
  u.timezone,
  f.apprise_service_urls,
  f.webhook_url,
  f.disable_http2,
  f.ntfy_enabled,
  f.ntfy_priority,
  f.ntfy_topic,
  f.pushover_enabled,
  f.pushover_priority,
  f.proxy_url,
  f.extra
FROM feeds f
     LEFT JOIN categories c ON c.id=f.category_id
     LEFT JOIN feed_icons fi ON fi.feed_id=f.id
     LEFT JOIN icons i ON i.id=fi.icon_id
     LEFT JOIN users u ON u.id=f.user_id
WHERE `+f.buildCondition()+" "+f.buildSorting(), f.args...)
	if err != nil {
		return nil, fmt.Errorf("storage: query feeds: %w", err)
	}
	defer rows.Close()

	feeds := make(model.Feeds, 0)
	for rows.Next() {
		var iconID pgtype.Int8
		var externalIconID pgtype.Text
		var tz string

		feed := model.NewFeed()
		err := rows.Scan(
			&feed.ID,
			&feed.FeedURL,
			&feed.SiteURL,
			&feed.Title,
			&feed.Description,
			&feed.EtagHeader,
			&feed.LastModifiedHeader,
			&feed.UserID,
			&feed.CheckedAt,
			&feed.NextCheckAt,
			&feed.ParsingErrorCount,
			&feed.ParsingErrorMsg,
			&feed.ScraperRules,
			&feed.RewriteRules,
			&feed.UrlRewriteRules,
			&feed.BlocklistRules,
			&feed.KeeplistRules,
			&feed.Crawler,
			&feed.UserAgent,
			&feed.Cookie,
			&feed.Username,
			&feed.Password,
			&feed.IgnoreHTTPCache,
			&feed.AllowSelfSignedCertificates,
			&feed.FetchViaProxy,
			&feed.Disabled,
			&feed.NoMediaPlayer,
			&feed.HideGlobally,
			&feed.Category.ID,
			&feed.Category.Title,
			&feed.Category.HideGlobally,
			&iconID,
			&externalIconID,
			&tz,
			&feed.AppriseServiceURLs,
			&feed.WebhookURL,
			&feed.DisableHTTP2,
			&feed.NtfyEnabled,
			&feed.NtfyPriority,
			&feed.NtfyTopic,
			&feed.PushoverEnabled,
			&feed.PushoverPriority,
			&feed.ProxyURL,
			&feed.Extra,
		)
		if err != nil {
			return nil, fmt.Errorf("storage: fetch feeds row: %w", err)
		}

		if iconID.Valid && externalIconID.Valid {
			*feed.Icon = model.FeedIcon{
				FeedID:         feed.ID,
				IconID:         iconID.Int64,
				ExternalIconID: externalIconID.String,
			}
		} else {
			feed.Icon.FeedID = feed.ID
		}

		feed.CheckedAt = timezone.Convert(tz, feed.CheckedAt)
		feed.NextCheckAt = timezone.Convert(tz, feed.NextCheckAt)
		feed.Category.UserID = feed.UserID
		feeds = append(feeds, feed)
	}
	return feeds, nil
}

// GetFeeds returns a list of feeds that match the condition.
func (f *FeedQueryBuilder) GetFeeds(ctx context.Context) (model.Feeds, error) {
	if !f.withCounters {
		return f.queryFeeds(ctx)
	}

	g, ctx := errgroup.WithContext(ctx)
	var feeds model.Feeds
	g.Go(func() (err error) {
		feeds, err = f.queryFeeds(ctx)
		return
	})

	var read, unread map[int64]int
	g.Go(func() (err error) {
		read, unread, err = f.fetchFeedCounter(ctx)
		return
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("group wait: %w", err)
	}

	f.fillFeedCounters(feeds, read, unread)
	return feeds, nil
}

func (f *FeedQueryBuilder) fetchFeedCounter(ctx context.Context,
) (map[int64]int, map[int64]int, error) {
	query := `
SELECT e.feed_id, e.status, count(*)
  FROM entries e %s
 WHERE %s
GROUP BY e.feed_id, e.status`

	join := ""
	if f.counterJoinFeeds {
		join = "LEFT JOIN feeds f ON f.id=e.feed_id"
	}

	rows, _ := f.db.Query(ctx, fmt.Sprintf(
		query, join, f.buildCounterCondition()), f.counterArgs...)

	readCounters := make(map[int64]int)
	unreadCounters := make(map[int64]int)
	var feedID int64
	var status string
	var count int

	_, err := pgx.ForEachRow(rows, []any{&feedID, &status, &count},
		func() error {
			switch status {
			case model.EntryStatusRead:
				readCounters[feedID] = count
			case model.EntryStatusUnread:
				unreadCounters[feedID] = count
			}
			return nil
		})
	if err != nil {
		return nil, nil, fmt.Errorf(`store: unable to fetch feed counts: %w`, err)
	}
	return readCounters, unreadCounters, nil
}

func (f *FeedQueryBuilder) fillFeedCounters(feeds model.Feeds, readCounters,
	unreadCounters map[int64]int,
) {
	for _, feed := range feeds {
		if count, ok := readCounters[feed.ID]; ok {
			feed.ReadCount = count
		}
		if count, ok := unreadCounters[feed.ID]; ok {
			feed.UnreadCount = count
		}
		feed.NumberOfVisibleEntries = feed.ReadCount + feed.UnreadCount
	}
}
