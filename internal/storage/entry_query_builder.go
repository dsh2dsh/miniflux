// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"fmt"
	"iter"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/timezone"
)

// NewEntryQueryBuilder returns a new EntryQueryBuilder.
func (s *Storage) NewEntryQueryBuilder(userID int64) *EntryQueryBuilder {
	return &EntryQueryBuilder{
		store:      s,
		db:         s.db,
		args:       []any{userID},
		conditions: []string{"e.user_id = $1"},
	}
}

// NewAnonymousQueryBuilder returns a new EntryQueryBuilder suitable for
// anonymous users.
func (s *Storage) NewAnonymousQueryBuilder() *EntryQueryBuilder {
	return &EntryQueryBuilder{store: s, db: s.db}
}

// EntryQueryBuilder builds a SQL query to fetch entries.
type EntryQueryBuilder struct {
	store           *Storage
	db              *pgxpool.Pool
	args            []any
	conditions      []string
	sortExpressions []string
	limit           int
	offset          int
	fetchContent    bool
}

func (e *EntryQueryBuilder) WithContent() *EntryQueryBuilder {
	e.fetchContent = true
	return e
}

// WithSearchQuery adds full-text search query to the condition.
func (e *EntryQueryBuilder) WithSearchQuery(query string) *EntryQueryBuilder {
	if query == "" {
		return e
	}

	n := len(e.args) + 1
	e.conditions = append(e.conditions, fmt.Sprintf(
		"e.document_vectors @@ websearch_to_tsquery($%d)", n))
	e.args = append(e.args, query)

	// 0.0000001 = 0.1 / (seconds_in_a_day)
	const tsRank = `ts_rank(document_vectors, websearch_to_tsquery($%d)) - extract(epoch from now() - published_at)::float * 0.0000001`
	e.WithSorting(fmt.Sprintf(tsRank, n), "DESC")
	return e
}

// WithStarred adds starred filter.
func (e *EntryQueryBuilder) WithStarred(starred bool) *EntryQueryBuilder {
	if starred {
		e.conditions = append(e.conditions, "e.starred is true")
	} else {
		e.conditions = append(e.conditions, "e.starred is false")
	}
	return e
}

// BeforeChangedDate adds a condition < changed_at
func (e *EntryQueryBuilder) BeforeChangedDate(date time.Time,
) *EntryQueryBuilder {
	e.conditions = append(e.conditions,
		"e.changed_at < $"+strconv.Itoa(len(e.args)+1))
	e.args = append(e.args, date)
	return e
}

// AfterChangedDate adds a condition > changed_at
func (e *EntryQueryBuilder) AfterChangedDate(date time.Time,
) *EntryQueryBuilder {
	e.conditions = append(e.conditions,
		"e.changed_at > $"+strconv.Itoa(len(e.args)+1))
	e.args = append(e.args, date)
	return e
}

// BeforePublishedDate adds a condition < published_at
func (e *EntryQueryBuilder) BeforePublishedDate(date time.Time,
) *EntryQueryBuilder {
	e.conditions = append(e.conditions,
		"e.published_at < $"+strconv.Itoa(len(e.args)+1))
	e.args = append(e.args, date)
	return e
}

// AfterPublishedDate adds a condition > published_at
func (e *EntryQueryBuilder) AfterPublishedDate(date time.Time,
) *EntryQueryBuilder {
	e.conditions = append(e.conditions,
		"e.published_at > $"+strconv.Itoa(len(e.args)+1))
	e.args = append(e.args, date)
	return e
}

// BeforeEntryID adds a condition < entryID.
func (e *EntryQueryBuilder) BeforeEntryID(entryID int64) *EntryQueryBuilder {
	if entryID != 0 {
		e.conditions = append(e.conditions,
			"e.id < $"+strconv.Itoa(len(e.args)+1))
		e.args = append(e.args, entryID)
	}
	return e
}

// AfterEntryID adds a condition > entryID.
func (e *EntryQueryBuilder) AfterEntryID(entryID int64) *EntryQueryBuilder {
	if entryID != 0 {
		e.conditions = append(e.conditions,
			"e.id > $"+strconv.Itoa(len(e.args)+1))
		e.args = append(e.args, entryID)
	}
	return e
}

// WithEntryIDs filter by entry IDs.
func (e *EntryQueryBuilder) WithEntryIDs(entryIDs []int64) *EntryQueryBuilder {
	e.conditions = append(e.conditions,
		fmt.Sprintf("e.id = ANY($%d)", len(e.args)+1))
	e.args = append(e.args, entryIDs)
	return e
}

// WithEntryID filter by entry ID.
func (e *EntryQueryBuilder) WithEntryID(entryID int64) *EntryQueryBuilder {
	if entryID != 0 {
		e.conditions = append(e.conditions,
			"e.id = $"+strconv.Itoa(len(e.args)+1))
		e.args = append(e.args, entryID)
	}
	return e
}

// WithFeedID filter by feed ID.
func (e *EntryQueryBuilder) WithFeedID(feedID int64) *EntryQueryBuilder {
	if feedID > 0 {
		e.conditions = append(e.conditions,
			"e.feed_id = $"+strconv.Itoa(len(e.args)+1))
		e.args = append(e.args, feedID)
	}
	return e
}

// WithCategoryID filter by category ID.
func (e *EntryQueryBuilder) WithCategoryID(categoryID int64) *EntryQueryBuilder {
	if categoryID > 0 {
		e.conditions = append(e.conditions,
			"f.category_id = $"+strconv.Itoa(len(e.args)+1))
		e.args = append(e.args, categoryID)
	}
	return e
}

// WithStatus filter by entry status.
func (e *EntryQueryBuilder) WithStatus(status string) *EntryQueryBuilder {
	if status != "" {
		e.conditions = append(e.conditions,
			"e.status = $"+strconv.Itoa(len(e.args)+1))
		e.args = append(e.args, status)
	}
	return e
}

// WithStatuses filter by a list of entry statuses.
func (e *EntryQueryBuilder) WithStatuses(statuses []string) *EntryQueryBuilder {
	if len(statuses) > 0 {
		e.conditions = append(e.conditions,
			fmt.Sprintf("e.status = ANY($%d)", len(e.args)+1))
		e.args = append(e.args, statuses)
	}
	return e
}

// WithTags filter by a list of entry tags.
func (e *EntryQueryBuilder) WithTags(tags []string) *EntryQueryBuilder {
	if len(tags) > 0 {
		for _, cat := range tags {
			e.conditions = append(e.conditions,
				fmt.Sprintf("LOWER($%d) = ANY(LOWER(e.tags::text)::text[])",
					len(e.args)+1))
			e.args = append(e.args, cat)
		}
	}
	return e
}

// WithoutStatus set the entry status that should not be returned.
func (e *EntryQueryBuilder) WithoutStatus(status string) *EntryQueryBuilder {
	if status != "" {
		e.conditions = append(e.conditions,
			"e.status <> $"+strconv.Itoa(len(e.args)+1))
		e.args = append(e.args, status)
	}
	return e
}

// WithSorting add a sort expression.
func (e *EntryQueryBuilder) WithSorting(column, direction string,
) *EntryQueryBuilder {
	e.sortExpressions = append(e.sortExpressions, column+" "+direction)
	return e
}

// WithLimit set the limit.
func (e *EntryQueryBuilder) WithLimit(limit int) *EntryQueryBuilder {
	if limit > 0 {
		e.limit = limit
	}
	return e
}

// WithOffset set the offset.
func (e *EntryQueryBuilder) WithOffset(offset int) *EntryQueryBuilder {
	if offset > 0 {
		e.offset = offset
	}
	return e
}

func (e *EntryQueryBuilder) WithGloballyVisible() *EntryQueryBuilder {
	e.conditions = append(e.conditions, "c.hide_globally IS FALSE")
	e.conditions = append(e.conditions, "f.hide_globally IS FALSE")
	return e
}

// CountEntries count the number of entries that match the condition.
func (e *EntryQueryBuilder) CountEntries(ctx context.Context) (int, error) {
	query := `
SELECT count(*)
  FROM entries e
	     JOIN feeds f ON f.id = e.feed_id
	     JOIN categories c ON c.id = f.category_id
 WHERE ` + e.buildCondition()

	rows, _ := e.db.Query(ctx, query, e.args...)
	count, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int])
	if err != nil {
		return 0, fmt.Errorf("store: unable to count entries: %w", err)
	}
	return count, nil
}

// GetEntry returns a single entry that match the condition.
func (e *EntryQueryBuilder) GetEntry(ctx context.Context,
) (*model.Entry, error) {
	entries, err := e.WithLimit(1).WithContent().GetEntries(ctx)
	if err != nil {
		return nil, err
	} else if len(entries) != 1 {
		return nil, nil
	}
	return entries[0], nil
}

// GetEntries returns a list of entries that match the condition.
func (e *EntryQueryBuilder) GetEntries(ctx context.Context,
) (model.Entries, error) {
	withContent := func() string {
		if e.fetchContent {
			return ", e.content"
		}
		return ""
	}

	query := `
SELECT
  e.id,
  e.user_id,
  e.feed_id,
  e.hash,
  e.published_at,
  e.title,
  e.url,
  e.comments_url,
  e.author,
  e.status,
  e.starred,
  e.reading_time,
  e.created_at,
  e.changed_at,
  e.tags,
  e.extra,
  f.title as feed_title,
  f.feed_url,
  f.site_url,
  f.description,
  f.checked_at,
  f.category_id,
  c.title as category_title,
  c.hide_globally as category_hidden,
  f.scraper_rules,
  f.rewrite_rules,
  f.crawler,
  f.user_agent,
  f.cookie,
  f.hide_globally,
  f.no_media_player,
  f.webhook_url,
  COALESCE(f.extra ->> 'comments_url_template', ''),
  fi.icon_id, i.hash AS icon_hash,
  u.timezone` + withContent() + `
FROM entries e
     LEFT JOIN feeds f ON f.id=e.feed_id
     LEFT JOIN categories c ON c.id=f.category_id
     LEFT JOIN feed_icons fi ON fi.feed_id=f.id
     LEFT JOIN icons i ON i.id=fi.icon_id
     LEFT JOIN users u ON u.id=e.user_id
WHERE ` + e.buildCondition() + " " + e.buildSorting()

	rows, err := e.db.Query(ctx, query, e.args...)
	if err != nil {
		return nil, fmt.Errorf("storage: unable to get entries: %w", err)
	}
	defer rows.Close()

	dest := make([]any, 0, 37)
	var entries model.Entries
	var hasCommentsURLTemplate bool
	entryMap := make(map[int64]*model.Entry)

	for rows.Next() {
		var iconID pgtype.Int8
		var iconHash pgtype.Text
		var tz string

		entry := &model.Entry{
			Date: time.Now(),
			Feed: &model.Feed{
				Category: &model.Category{},
				Icon:     &model.FeedIcon{},
			},
			Tags: []string{},
		}

		dest = append(dest,
			&entry.ID,
			&entry.UserID,
			&entry.FeedID,
			&entry.Hash,
			&entry.Date,
			&entry.Title,
			&entry.URL,
			&entry.CommentsURL,
			&entry.Author,
			&entry.Status,
			&entry.Starred,
			&entry.ReadingTime,
			&entry.CreatedAt,
			&entry.ChangedAt,
			&entry.Tags,
			&entry.Extra,
			&entry.Feed.Title,
			&entry.Feed.FeedURL,
			&entry.Feed.SiteURL,
			&entry.Feed.Description,
			&entry.Feed.CheckedAt,
			&entry.Feed.Category.ID,
			&entry.Feed.Category.Title,
			&entry.Feed.Category.HideGlobally,
			&entry.Feed.ScraperRules,
			&entry.Feed.RewriteRules,
			&entry.Feed.Crawler,
			&entry.Feed.UserAgent,
			&entry.Feed.Cookie,
			&entry.Feed.HideGlobally,
			&entry.Feed.NoMediaPlayer,
			&entry.Feed.WebhookURL,
			&entry.Feed.Extra.CommentsURLTemplate,
			&iconID, &iconHash,
			&tz)

		if e.fetchContent {
			dest = append(dest, &entry.Content)
		}

		err := rows.Scan(dest...)
		if err != nil {
			return nil, fmt.Errorf("storage: unable to fetch entry row: %w", err)
		}
		dest = dest[:0]

		hasCommentsURLTemplate = hasCommentsURLTemplate ||
			entry.Feed.Extra.CommentsURLTemplate != ""

		if iconID.Valid && iconHash.Valid && iconHash.String != "" {
			*entry.Feed.Icon = model.FeedIcon{
				FeedID: entry.FeedID,
				IconID: iconID.Int64,
				Hash:   iconHash.String,
			}
		} else {
			entry.Feed.Icon.IconID = 0
		}

		// Make sure that timestamp fields contain timezone information (API)
		timezone.Convert(tz,
			&entry.Date,
			&entry.CreatedAt,
			&entry.ChangedAt,
			&entry.Feed.CheckedAt)

		entry.Feed.ID = entry.FeedID
		entry.Feed.UserID = entry.UserID
		entry.Feed.Icon.FeedID = entry.FeedID
		entry.Feed.Category.UserID = entry.UserID

		entries = append(entries, entry)
		entryMap[entry.ID] = entry
	}

	if hasCommentsURLTemplate {
		entries.MakeCommentURLs(ctx)
	}
	return entries, nil
}

// GetEntryIDs returns a list of entry IDs that match the condition.
func (e *EntryQueryBuilder) GetEntryIDs(ctx context.Context) ([]int64, error) {
	rows, _ := e.db.Query(ctx, `
SELECT e.id
  FROM entries e
       LEFT JOIN feeds f ON f.id=e.feed_id
 WHERE `+e.buildCondition()+" "+e.buildSorting(), e.args...)

	entryIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return nil, fmt.Errorf("store: unable to get entries: %w", err)
	}
	return entryIDs, nil
}

func (e *EntryQueryBuilder) buildCondition() string {
	return strings.Join(e.conditions, " AND ")
}

func (e *EntryQueryBuilder) buildSorting() string {
	var parts string
	if len(e.sortExpressions) > 0 {
		parts = " ORDER BY " + strings.Join(e.sortExpressions, ", ")
	}

	if e.limit > 0 {
		parts += " LIMIT " + strconv.FormatInt(int64(e.limit), 10)
	}

	if e.offset > 0 {
		parts += " OFFSET " + strconv.FormatInt(int64(e.offset), 10)
	}
	return parts
}

func (e *EntryQueryBuilder) CountStatusStarred(ctx context.Context,
) (iter.Seq2[*model.Entry, int], error) {
	rows, _ := e.db.Query(ctx, `
SELECT e.status, e.starred, count(*) AS count
  FROM entries e, feeds f, categories c
 WHERE f.id = e.feed_id AND c.id = f.category_id AND
       `+e.buildCondition()+`
GROUP BY e.status, e.starred`, e.args...)

	type groupCount struct {
		Status  string
		Starred bool
		Count   int
	}

	counts, err := pgx.CollectRows(rows, pgx.RowToStructByName[groupCount])
	if err != nil {
		return nil, fmt.Errorf("storage: count entries GROUP BY: %w", err)
	}

	seqFunc := func(yield func(*model.Entry, int) bool) {
		var entry model.Entry
		for _, item := range counts {
			entry.Status, entry.Starred = item.Status, item.Starred
			if !yield(&entry, item.Count) {
				return
			}
		}
	}
	return seqFunc, nil
}
