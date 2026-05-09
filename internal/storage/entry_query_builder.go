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
		db:         s.db,
		args:       []any{userID},
		conditions: []string{"e.user_id = $1"},
	}
}

// EntryQueryBuilder builds a SQL query to fetch entries.
type EntryQueryBuilder struct {
	db              *pgxpool.Pool
	args            []any
	conditions      []string
	sortExpressions []string
	limit           int
	offset          int
	fetchContent    bool

	numberedRows bool
	highConds    []string
}

func (self *EntryQueryBuilder) appendCondition(prefix string, arg any,
	suffix string,
) string {
	if arg == nil {
		self.conditions = append(self.conditions, prefix)
		return ""
	}

	self.args = append(self.args, arg)
	argPos := strconv.Itoa(len(self.args))

	s := prefix + argPos
	if suffix != "" {
		s += suffix
	}

	self.conditions = append(self.conditions, s)
	return argPos
}

func (self *EntryQueryBuilder) WithContent() *EntryQueryBuilder {
	self.fetchContent = true
	return self
}

func (self *EntryQueryBuilder) WithAuthor(name string) *EntryQueryBuilder {
	self.appendCondition("e.author = $", name, "")
	return self
}

// WithSearchQuery adds full-text search query to the condition.
func (self *EntryQueryBuilder) WithSearchQuery(query string) *EntryQueryBuilder {
	if query == "" {
		return self
	}

	argPos := self.appendCondition(
		"e.document_vectors @@ websearch_to_tsquery($", query, ")")

	// 0.0000001 = 0.1 / (seconds_in_a_day)
	self.sortExpressions = append(self.sortExpressions,
		`ts_rank(e.document_vectors, websearch_to_tsquery($`+argPos+`)) - extract(epoch from now() - e.published_at)::float * 0.0000001 DESC`)
	return self
}

// WithStarred adds starred filter.
func (self *EntryQueryBuilder) WithStarred(starred bool) *EntryQueryBuilder {
	if starred {
		self.appendCondition("e.starred is true", nil, "")
	} else {
		self.appendCondition("e.starred is false", nil, "")
	}
	return self
}

// BeforeChangedDate adds a condition < changed_at
func (self *EntryQueryBuilder) BeforeChangedDate(date time.Time,
) *EntryQueryBuilder {
	self.appendCondition("e.changed_at < $", date, "")
	return self
}

// AfterChangedDate adds a condition > changed_at
func (self *EntryQueryBuilder) AfterChangedDate(date time.Time,
) *EntryQueryBuilder {
	self.appendCondition("e.changed_at > $", date, "")
	return self
}

// BeforePublishedDate adds a condition < published_at
func (self *EntryQueryBuilder) BeforePublishedDate(date time.Time,
) *EntryQueryBuilder {
	self.appendCondition("e.published_at < $", date, "")
	return self
}

// AfterPublishedDate adds a condition > published_at
func (self *EntryQueryBuilder) AfterPublishedDate(date time.Time,
) *EntryQueryBuilder {
	self.appendCondition("e.published_at > $", date, "")
	return self
}

// BeforeEntryID adds a condition < entryID.
func (self *EntryQueryBuilder) BeforeEntryID(entryID int64) *EntryQueryBuilder {
	if entryID != 0 {
		self.appendCondition("e.id < $", entryID, "")
	}
	return self
}

// AfterEntryID adds a condition > entryID.
func (self *EntryQueryBuilder) AfterEntryID(entryID int64) *EntryQueryBuilder {
	if entryID != 0 {
		self.appendCondition("e.id > $", entryID, "")
	}
	return self
}

// WithEntryIDs filter by entry IDs.
func (self *EntryQueryBuilder) WithEntryIDs(entryIDs []int64,
) *EntryQueryBuilder {
	switch len(entryIDs) {
	case 0:
		return self
	case 1:
		return self.WithEntryID(entryIDs[0])
	}

	self.appendCondition("e.id = ANY($", entryIDs, ")")
	return self
}

// WithEntryID filter by entry ID.
func (self *EntryQueryBuilder) WithEntryID(entryID int64) *EntryQueryBuilder {
	if entryID != 0 {
		self.appendCondition("e.id = $", entryID, "")
	}
	return self
}

// WithFeedID filter by feed ID.
func (self *EntryQueryBuilder) WithFeedID(feedID int64) *EntryQueryBuilder {
	if feedID > 0 {
		self.appendCondition("e.feed_id = $", feedID, "")
	}
	return self
}

// WithCategoryID filter by category ID.
func (self *EntryQueryBuilder) WithCategoryID(categoryID int64) *EntryQueryBuilder {
	if categoryID > 0 {
		self.appendCondition("f.category_id = $", categoryID, "")
	}
	return self
}

// WithStatus filter by entry status.
func (self *EntryQueryBuilder) WithStatus(status string) *EntryQueryBuilder {
	if status != "" {
		self.appendCondition("e.status = $", status, "")
	}
	return self
}

// WithStatuses filter by a list of entry statuses.
func (self *EntryQueryBuilder) WithStatuses(statuses []string,
) *EntryQueryBuilder {
	switch len(statuses) {
	case 0:
		return self
	case 1:
		return self.WithStatus(statuses[0])
	}

	self.appendCondition("e.status = ANY($", statuses, ")")
	return self
}

// WithTags filter by a list of entry tags.
func (self *EntryQueryBuilder) WithTags(tags []string) *EntryQueryBuilder {
	if len(tags) == 0 {
		return self
	}

	for _, s := range tags {
		self.appendCondition("LOWER($", s, ") = ANY(LOWER(e.tags::text)::text[])")
	}
	return self
}

// WithoutStatus set the entry status that should not be returned.
func (self *EntryQueryBuilder) WithoutStatus(status string) *EntryQueryBuilder {
	if status != "" {
		self.appendCondition("e.status <> $", status, "")
	}
	return self
}

// WithSorting add a sort expression.
func (self *EntryQueryBuilder) WithSorting(column, direction string,
) *EntryQueryBuilder {
	self.sortExpressions = append(self.sortExpressions, "e."+column+" "+direction)
	return self
}

// WithLimit set the limit.
func (self *EntryQueryBuilder) WithLimit(limit int) *EntryQueryBuilder {
	if limit > 0 {
		self.limit = limit
	}
	return self
}

// WithOffset set the offset.
func (self *EntryQueryBuilder) WithOffset(offset int) *EntryQueryBuilder {
	if offset > 0 {
		self.offset = offset
	}
	return self
}

func (self *EntryQueryBuilder) WithOffsetID(id int64) *EntryQueryBuilder {
	if id == 0 {
		return self
	}

	self.args = append(self.args, id)
	pos := strconv.Itoa(len(self.args))
	self.highConds = append(self.highConds,
		`row_number > COALESCE((SELECT row_number FROM t WHERE id = $`+pos+`), 0)`)
	self.numberedRows = true
	return self
}

func (self *EntryQueryBuilder) WithGloballyVisible() *EntryQueryBuilder {
	self.appendCondition("c.hide_globally IS FALSE", nil, "")
	self.appendCondition("f.hide_globally IS FALSE", nil, "")
	return self
}

// CountEntries count the number of entries that match the condition.
func (self *EntryQueryBuilder) CountEntries(ctx context.Context) (int, error) {
	query := `
SELECT count(*)
  FROM entries e
	     JOIN feeds f ON f.id = e.feed_id
	     JOIN categories c ON c.id = f.category_id
 WHERE ` + self.buildCondition()

	rows, _ := self.db.Query(ctx, query, self.args...)
	count, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int])
	if err != nil {
		return 0, fmt.Errorf("store: unable to count entries: %w", err)
	}
	return count, nil
}

// GetEntry returns a single entry that match the condition.
func (self *EntryQueryBuilder) GetEntry(ctx context.Context,
) (*model.Entry, error) {
	entries, err := self.WithLimit(1).WithContent().GetEntries(ctx)
	if err != nil {
		return nil, err
	} else if len(entries) != 1 {
		return nil, nil
	}
	return entries[0], nil
}

// GetEntries returns a list of entries that match the condition.
func (self *EntryQueryBuilder) GetEntries(ctx context.Context,
) (model.Entries, error) {
	query := self.entriesQuery()
	rows, err := self.db.Query(ctx, query, self.args...)
	if err != nil {
		return nil, fmt.Errorf("storage: unable to get entries: %w", err)
	}
	defer rows.Close()

	dest := make([]any, 0, 38)
	var entries model.Entries
	entryMap := make(map[int64]*model.Entry)

	for rows.Next() {
		var iconID pgtype.Int8
		var iconHash pgtype.Text
		var tz string
		var rowNumber int64

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
			&entry.Feed.Category.Extra,
			&entry.Feed.ScraperRules,
			&entry.Feed.RewriteRules,
			&entry.Feed.Crawler,
			&entry.Feed.UserAgent,
			&entry.Feed.Cookie,
			&entry.Feed.HideGlobally,
			&entry.Feed.NoMediaPlayer,
			&entry.Feed.WebhookURL,
			&iconID, &iconHash,
			&tz)

		if self.fetchContent {
			dest = append(dest, &entry.Content)
		}

		if self.numberedRows {
			dest = append(dest, &rowNumber)
		}

		err := rows.Scan(dest...)
		if err != nil {
			return nil, fmt.Errorf("storage: unable to fetch entry row: %w", err)
		}
		dest = dest[:0]

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
		entry.MarkStored()

		entries = append(entries, entry)
		entryMap[entry.ID] = entry
	}
	return entries, nil
}

func (self *EntryQueryBuilder) entriesQuery() string {
	body := `
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
	f.title AS feed_title,
	f.feed_url,
	f.site_url,
	f.description,
	f.checked_at,
	f.category_id,
	c.title AS category_title,
	c.hide_globally AS category_hidden,
	c.extra AS category_extra,
	f.scraper_rules,
	f.rewrite_rules,
	f.crawler,
	f.user_agent,
	f.cookie,
	f.hide_globally,
	f.no_media_player,
	f.webhook_url,
	fi.icon_id, i.hash AS icon_hash,
	u.timezone` + self.withContentField() + self.withRowNumberField() + `
FROM entries e
		 LEFT JOIN feeds f ON f.id=e.feed_id
		 LEFT JOIN categories c ON c.id=f.category_id
		 LEFT JOIN feed_icons fi ON fi.feed_id=f.id
		 LEFT JOIN icons i ON i.id=fi.icon_id
		 LEFT JOIN users u ON u.id=e.user_id
WHERE ` + self.buildCondition() + self.buildSorting()

	if len(self.highConds) == 0 {
		return body + self.buildLimitOffset()
	}

	return `
WITH t AS (` + body + `) SELECT * FROM t
WHERE ` + self.buildHighConds() + self.buildLimitOffset()
}

func (self *EntryQueryBuilder) withContentField() string {
	if self.fetchContent {
		return ", e.content"
	}
	return ""
}

func (self *EntryQueryBuilder) withRowNumberField() string {
	if self.numberedRows {
		return `, row_number() OVER(` + self.buildSorting() + `) AS row_number`
	}
	return ""
}

func (self *EntryQueryBuilder) buildHighConds() string {
	if len(self.highConds) != 0 {
		return strings.Join(self.highConds, " AND ")
	}
	return ""
}

// GetEntryIDs returns a list of entry IDs that match the condition.
func (self *EntryQueryBuilder) GetEntryIDs(ctx context.Context) ([]int64, error) {
	rows, _ := self.db.Query(ctx, `
SELECT e.id
  FROM entries e LEFT JOIN feeds f ON f.id=e.feed_id
 WHERE `+self.buildCondition()+self.buildSorting()+self.buildLimitOffset(),
		self.args...)

	entryIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return nil, fmt.Errorf("store: unable to get entries: %w", err)
	}
	return entryIDs, nil
}

func (self *EntryQueryBuilder) buildCondition() string {
	if len(self.conditions) != 0 {
		return strings.Join(self.conditions, " AND ")
	}
	return ""
}

func (self *EntryQueryBuilder) buildSorting() (s string) {
	if len(self.sortExpressions) > 0 {
		s = " ORDER BY " + strings.Join(self.sortExpressions, ", ")
	}
	return s
}

func (self *EntryQueryBuilder) buildLimitOffset() (s string) {
	if self.limit > 0 {
		s = " LIMIT " + strconv.FormatInt(int64(self.limit), 10)
	}
	if self.offset > 0 {
		s += " OFFSET " + strconv.FormatInt(int64(self.offset), 10)
	}
	return s
}

func (self *EntryQueryBuilder) CountStatusStarred(ctx context.Context,
) (iter.Seq2[*model.Entry, int], error) {
	rows, _ := self.db.Query(ctx, `
SELECT e.status, e.starred, count(*) AS count
  FROM entries e, feeds f, categories c
 WHERE f.id = e.feed_id AND c.id = f.category_id AND
       `+self.buildCondition()+`
GROUP BY e.status, e.starred`, self.args...)

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
