// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
)

// CountAllEntries returns the number of entries for each status in the
// database.
func (s *Storage) CountAllEntries(ctx context.Context) (map[string]int64,
	error,
) {
	rows, _ := s.db.Query(ctx,
		`SELECT status, count(*) AS count FROM entries GROUP BY status`)

	const total = "total"
	results := map[string]int64{
		model.EntryStatusUnread:  0,
		model.EntryStatusRead:    0,
		model.EntryStatusRemoved: 0,
		total:                    0,
	}

	type statusCount struct {
		Status string
		Count  int64
	}

	counts, err := pgx.CollectRows(rows, pgx.RowToStructByName[statusCount])
	if err != nil {
		return nil, fmt.Errorf("storage: count entries by status: %w", err)
	}

	for _, s := range counts {
		results[s.Status] = s.Count
		results[total] += s.Count
	}
	return results, nil
}

// CountUnreadEntries returns the number of unread entries.
func (s *Storage) CountUnreadEntries(ctx context.Context, userID int64) int {
	builder := s.NewEntryQueryBuilder(userID).
		WithStatus(model.EntryStatusUnread).
		WithGloballyVisible()

	n, err := builder.CountEntries(ctx)
	if err != nil {
		logging.FromContext(ctx).Error("Unable to count unread entries",
			slog.Int64("user_id", userID), slog.Any("error", err))
		return 0
	}
	return n
}

// UpdateEntryTitleAndContent updates entry title and content.
func (s *Storage) UpdateEntryTitleAndContent(ctx context.Context,
	entry *model.Entry,
) error {
	_, err := s.db.Exec(ctx, `
UPDATE entries
   SET title = $1,
       content = $2,
       reading_time = $3
 WHERE id = $4 AND user_id = $5`,
		entry.Title,
		entry.Content,
		entry.ReadingTime,
		entry.ID, entry.UserID)
	if err != nil {
		return fmt.Errorf(`store: unable to update entry #%d: %w`, entry.ID, err)
	}
	return nil
}

func (s *Storage) GetReadTime(ctx context.Context, feedID int64,
	entryHash string,
) int {
	// Note: This query uses entries_feed_id_hash_key index
	rows, _ := s.db.Query(ctx,
		`SELECT reading_time FROM entries WHERE feed_id=$1 AND hash=$2`,
		feedID, entryHash)

	readingTime, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int])
	if err != nil {
		logging.FromContext(ctx).Error("store: unable to fetch entry reading_time",
			slog.Int64("feed_id", feedID),
			slog.String("hash", entryHash),
			slog.Any("error", err))
		return 0
	}
	return readingTime
}

// RefreshFeedEntries updates feed entries while refreshing a feed.
func (s *Storage) RefreshFeedEntries(ctx context.Context, userID, feedID int64,
	entries model.Entries, updateExisting bool,
) (*model.FeedRefreshed, error) {
	hashes := make([]string, len(entries))
	for i, e := range entries {
		e.UserID, e.FeedID, e.Status = userID, feedID, model.EntryStatusUnread
		hashes[i] = e.Hash
	}

	var refreshed *model.FeedRefreshed
	if len(entries) > 0 {
		err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
			r, err := s.refreshEntries(ctx, tx, userID, feedID, hashes, entries,
				updateExisting)
			if err != nil {
				return err
			}
			refreshed = r
			return nil
		})
		if err != nil {
			return refreshed, fmt.Errorf("unable refresh feed(#%d) entries: %w",
				feedID, err)
		}
	} else {
		refreshed = &model.FeedRefreshed{}
	}

	if err := s.cleanupEntries(ctx, feedID, hashes); err != nil {
		return nil, err
	}
	return refreshed, nil
}

func (s *Storage) refreshEntries(ctx context.Context, tx pgx.Tx, userID,
	feedID int64, hashes []string, entries model.Entries, update bool,
) (*model.FeedRefreshed, error) {
	refreshed, err := s.knownEntries(ctx, tx, userID, feedID, hashes, entries)
	if err != nil {
		return nil, err
	}

	if update {
		for _, e := range refreshed.UpdatedEntires {
			if err = s.updateEntry(ctx, tx, e); err != nil {
				return refreshed, err
			}
		}
	}

	if len(refreshed.CreatedEntries) == 0 {
		return refreshed, nil
	}
	return refreshed, s.createEntries(ctx, tx, refreshed.CreatedEntries)
}

func (s *Storage) knownEntries(ctx context.Context, tx pgx.Tx, userID,
	feedID int64, hashes []string, entries []*model.Entry,
) (*model.FeedRefreshed, error) {
	published, err := s.publishedEntries(ctx, tx, userID, hashes)
	if err != nil {
		return nil, err
	}

	byHash := make(map[string]*model.Entry, len(published))
	for i := range published {
		e := &published[i]
		if _, ok := byHash[e.Hash]; !ok || e.FeedID == feedID {
			byHash[e.Hash] = e
		}
	}

	refreshed := model.NewFeedRefreshed().Append(entries, byHash)
	if dd := DedupEntriesFrom(ctx); dd != nil {
		dd.Filter(userID, refreshed)
	}
	return refreshed, nil
}

func (s *Storage) publishedEntries(ctx context.Context, tx pgx.Tx,
	userID int64, hashes []string,
) ([]model.Entry, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	rows, _ := tx.Query(ctx, `
SELECT feed_id, status, hash, published_at
  FROM entries
 WHERE user_id = $1 AND hash = ANY($2)`, userID, hashes)

	entries, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[model.Entry])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("storage: check entries exist: %w", err)
	}
	return entries, nil
}

// updateEntry updates an entry when a feed is refreshed.
//
// Note: we do not update the published date because some feeds do not contains
// any date, it default to time.Now() which could change the order of items on
// the history page.
func (s *Storage) updateEntry(ctx context.Context, tx pgx.Tx, e *model.Entry,
) error {
	err := tx.QueryRow(ctx, `
UPDATE entries
   SET title = $1,
       url = $2,
       comments_url = $3,
       content = $4,
       author = $5,
       reading_time = $6,
       tags = $10,
       changed_at = now(),
       published_at = $11,
       status = $12,
       extra = $13
 WHERE user_id = $7 AND feed_id = $8 AND hash = $9
RETURNING id, status, changed_at`,
		e.Title,
		e.URL,
		e.CommentsURL,
		e.Content,
		e.Author,
		e.ReadingTime,
		e.UserID, e.FeedID, e.Hash,
		e.Tags,
		e.Date,
		model.EntryStatusUnread,
		&e.Extra,
	).Scan(&e.ID, &e.Status, &e.ChangedAt)
	if err != nil {
		return fmt.Errorf("storage: update entry %q: %w", e.URL, err)
	}
	return nil
}

func (s *Storage) createEntries(ctx context.Context, tx pgx.Tx,
	entries model.Entries,
) error {
	switch len(entries) {
	case 0:
		return nil
	case 1, 2:
		return s.batchCreateEntries(ctx, tx, entries)
	}

	byHash := make(map[string]*model.Entry, len(entries))
	hashes := make([]string, len(entries))
	now := time.Now()

	_, err := tx.CopyFrom(ctx, pgx.Identifier{"entries"},
		[]string{
			"status",
			"title",
			"hash",
			"url",
			"comments_url",
			"published_at",
			"content",
			"author",
			"user_id",
			"feed_id",
			"reading_time",
			"tags",
			"changed_at",
			"extra",
		},
		pgx.CopyFromSlice(len(entries), func(i int) ([]any, error) {
			e := entries[i]
			byHash[e.Hash] = e
			hashes[i] = e.Hash
			return []any{
				e.Status,
				e.Title,
				e.Hash,
				e.URL,
				e.CommentsURL,
				e.Date,
				e.Content,
				e.Author,
				e.UserID,
				e.FeedID,
				e.ReadingTime,
				e.Tags,
				now,
				&e.Extra,
			}, nil
		}))
	if err != nil {
		return fmt.Errorf("storage: copy from entries: %w", err)
	}

	feedID := entries[0].FeedID
	rows, _ := tx.Query(ctx, `
SELECT id, hash, created_at, changed_at
  FROM entries
 WHERE feed_id = $1 AND hash = ANY($2)`, feedID, hashes)

	var id int64
	var hash string
	var createdAt, changedAt time.Time
	scans := []any{&id, &hash, &createdAt, &changedAt}

	_, err = pgx.ForEachRow(rows, scans, func() error {
		e := byHash[hash]
		e.ID = id
		e.CreatedAt = createdAt
		e.ChangedAt = changedAt
		return nil
	})
	if err != nil {
		return fmt.Errorf("storage: returned entries: %w", err)
	}
	return nil
}

func (s *Storage) batchCreateEntries(ctx context.Context, tx pgx.Tx,
	entries model.Entries,
) error {
	var batch pgx.Batch
	for _, e := range entries {
		s.queueCreateEntry(&batch, e)
	}

	if err := tx.SendBatch(ctx, &batch).Close(); err != nil {
		return fmt.Errorf("storage: batch create entries(%d): %w",
			len(entries), err)
	}
	return nil
}

func (s *Storage) queueCreateEntry(batch *pgx.Batch, e *model.Entry) {
	batch.Queue(`
INSERT INTO entries (
  status,
  title,
  hash,
  url,
  comments_url,
  published_at,
  content,
  author,
  user_id,
  feed_id,
  reading_time,
  tags,
  extra,
  changed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, now())
RETURNING id, created_at, changed_at`,
		e.Status,
		e.Title,
		e.Hash,
		e.URL,
		e.CommentsURL,
		e.Date,
		e.Content,
		e.Author,
		e.UserID,
		e.FeedID,
		e.ReadingTime,
		e.Tags,
		&e.Extra,
	).QueryRow(func(row pgx.Row) error {
		err := row.Scan(&e.ID, &e.CreatedAt, &e.ChangedAt)
		if err != nil {
			return fmt.Errorf("queued create entry %q (feed #%d): %w",
				e.URL, e.FeedID, err)
		}
		return nil
	})
}

// cleanupEntries deletes from the database entries marked as "removed" and not
// visible anymore in the feed.
func (s *Storage) cleanupEntries(ctx context.Context, feedID int64,
	hashes []string,
) error {
	var err error
	if len(hashes) == 0 {
		_, err = s.db.Exec(ctx,
			`DELETE FROM entries WHERE feed_id = $1 AND status = $2`,
			feedID, model.EntryStatusRemoved)
	} else {
		_, err = s.db.Exec(ctx, `
DELETE FROM entries
 WHERE feed_id = $1 AND status = $2 AND NOT (hash = ANY($3))`,
			feedID, model.EntryStatusRemoved, hashes)
	}
	if err != nil {
		return fmt.Errorf(`store: unable to cleanup entries: %w`, err)
	}
	return nil
}

// ArchiveEntries changes the status of entries to "removed" after the given
// number of days.
func (s *Storage) ArchiveEntries(ctx context.Context, status string,
	days, limit int,
) (int64, error) {
	if days < 0 || limit <= 0 {
		return 0, nil
	}

	result, err := s.db.Exec(ctx, `
UPDATE entries
   SET status=$1
 WHERE id IN (
   SELECT id
     FROM entries
    WHERE status=$2 AND starred is false AND share_code=''
          AND changed_at < now () - $3::interval
    ORDER BY changed_at ASC LIMIT $4)`,
		model.EntryStatusRemoved, status,
		strconv.FormatInt(int64(days), 10)+" days", limit)
	if err != nil {
		return 0, fmt.Errorf(`store: unable to archive %s entries: %w`,
			status, err)
	}
	return result.RowsAffected(), nil
}

// SetEntriesStatus update the status of the given list of entries.
func (s *Storage) SetEntriesStatus(ctx context.Context, userID int64,
	entryIDs []int64, status string,
) error {
	_, err := s.db.Exec(ctx, `
UPDATE entries
   SET status=$1, changed_at=now()
 WHERE user_id=$2 AND id=ANY($3)`,
		status, userID, entryIDs)
	if err != nil {
		return fmt.Errorf(`store: unable to update entries statuses %v: %w`,
			entryIDs, err)
	}
	return nil
}

func (s *Storage) SetEntriesStatusCount(ctx context.Context, userID int64,
	entryIDs []int64, status string,
) (int, error) {
	err := s.SetEntriesStatus(ctx, userID, entryIDs, status)
	if err != nil {
		return 0, err
	}

	rows, _ := s.db.Query(ctx, `
SELECT count(*)
  FROM entries e
		   JOIN feeds f ON (f.id = e.feed_id)
		   JOIN categories c ON (c.id = f.category_id)
 WHERE e.user_id = $1 AND e.id = ANY($2) AND NOT f.hide_globally
	     AND NOT c.hide_globally`,
		userID, entryIDs)

	visible, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int])
	if err != nil {
		return 0, fmt.Errorf(`store: unable to query entries visibility %v: %w`,
			entryIDs, err)
	}
	return visible, nil
}

// SetEntriesBookmarked update the bookmarked state for the given list of
// entries.
func (s *Storage) SetEntriesBookmarkedState(ctx context.Context, userID int64,
	entryIDs []int64, starred bool,
) error {
	result, err := s.db.Exec(ctx, `
UPDATE entries
   SET starred=$1, changed_at=now()
 WHERE user_id=$2 AND id=ANY($3)`,
		starred, userID, entryIDs)
	if err != nil {
		return fmt.Errorf(`store: unable to update the bookmarked state %v: %w`,
			entryIDs, err)
	}

	if result.RowsAffected() == 0 {
		return errors.New(`store: nothing has been updated`)
	}
	return nil
}

// ToggleBookmark toggles entry bookmark value.
func (s *Storage) ToggleBookmark(ctx context.Context, userID, entryID int64,
) error {
	result, err := s.db.Exec(ctx, `
UPDATE entries
   SET starred = NOT starred, changed_at=now()
 WHERE user_id=$1 AND id=$2`,
		userID, entryID)
	if err != nil {
		return fmt.Errorf(
			`store: unable to toggle bookmark flag for entry #%d: %w`, entryID, err)
	}

	if result.RowsAffected() == 0 {
		return errors.New(`store: nothing has been updated`)
	}
	return nil
}

// FlushHistory changes all entries with the status "read" to "removed".
func (s *Storage) FlushHistory(ctx context.Context, userID int64) error {
	_, err := s.db.Exec(ctx, `
UPDATE entries
   SET status=$1, changed_at=now()
 WHERE user_id=$2 AND status=$3 AND starred is false AND share_code=''`,
		model.EntryStatusRemoved, userID, model.EntryStatusRead)
	if err != nil {
		return fmt.Errorf(`store: unable to flush history: %w`, err)
	}
	return nil
}

// MarkAllAsRead updates all user entries to the read status.
func (s *Storage) MarkAllAsRead(ctx context.Context, userID int64) error {
	result, err := s.db.Exec(ctx, `
UPDATE entries
   SET status=$1, changed_at=now()
 WHERE user_id=$2 AND status=$3`,
		model.EntryStatusRead, userID, model.EntryStatusUnread)
	if err != nil {
		return fmt.Errorf(`store: unable to mark all entries as read: %w`, err)
	}

	logging.FromContext(ctx).Debug("Marked all entries as read",
		slog.Int64("user_id", userID),
		slog.Int64("nb_entries", result.RowsAffected()))
	return nil
}

// MarkAllAsReadBeforeDate updates all user entries to the read status before
// the given date.
func (s *Storage) MarkAllAsReadBeforeDate(ctx context.Context, userID int64,
	before time.Time,
) error {
	result, err := s.db.Exec(ctx, `
UPDATE entries
   SET status=$1, changed_at=now()
 WHERE user_id=$2 AND status=$3 AND published_at < $4`,
		model.EntryStatusRead, userID, model.EntryStatusUnread, before)
	if err != nil {
		return fmt.Errorf(
			"store: unable to mark all entries as read before %s: %w",
			before.Format(time.RFC3339), err)
	}

	slog.Debug("Marked all entries as read before date",
		slog.Int64("user_id", userID),
		slog.Int64("nb_entries", result.RowsAffected()),
		slog.String("before", before.Format(time.RFC3339)))
	return nil
}

// MarkGloballyVisibleFeedsAsRead updates all user entries to the read status.
func (s *Storage) MarkGloballyVisibleFeedsAsRead(ctx context.Context,
	userID int64,
) error {
	result, err := s.db.Exec(ctx, `
UPDATE entries
   SET status=$1, changed_at=now()
  FROM feeds
 WHERE entries.feed_id = feeds.id AND entries.user_id=$2 AND entries.status=$3
       AND feeds.hide_globally=$4`,
		model.EntryStatusRead, userID, model.EntryStatusUnread, false)
	if err != nil {
		return fmt.Errorf(
			`store: unable to mark globally visible feeds as read: %w`, err)
	}

	logging.FromContext(ctx).Debug(
		"Marked globally visible feed entries as read",
		slog.Int64("user_id", userID),
		slog.Int64("nb_entries", result.RowsAffected()))
	return nil
}

// MarkFeedAsRead updates all feed entries to the read status.
func (s *Storage) MarkFeedAsRead(ctx context.Context, userID, feedID int64,
	before time.Time,
) (bool, error) {
	result, err := s.db.Exec(ctx, `
UPDATE entries
   SET status=$1, changed_at=now()
 WHERE user_id=$2 AND feed_id=$3 AND status=$4 AND published_at < $5`,
		model.EntryStatusRead, userID, feedID, model.EntryStatusUnread, before)
	if err != nil {
		return false, fmt.Errorf(
			"storage: unable to mark feed entries as read: %w", err)
	}

	logging.FromContext(ctx).Debug("Marked feed entries as read",
		slog.Int64("user_id", userID),
		slog.Int64("feed_id", feedID),
		slog.Int64("nb_entries", result.RowsAffected()),
		slog.String("before", before.Format(time.RFC3339)))
	return result.RowsAffected() != 0, nil
}

// MarkCategoryAsRead updates all category entries to the read status.
func (s *Storage) MarkCategoryAsRead(ctx context.Context, userID,
	categoryID int64, before time.Time,
) (bool, error) {
	result, err := s.db.Exec(ctx, `
UPDATE entries
   SET status=$1, changed_at=now()
  FROM feeds
 WHERE feed_id=feeds.id AND feeds.user_id=$2 AND status=$3 AND published_at < $4
       AND feeds.category_id=$5`,
		model.EntryStatusRead, userID, model.EntryStatusUnread, before, categoryID)
	if err != nil {
		return false, fmt.Errorf(
			"storage: unable to mark category entries as read: %w", err)
	}

	logging.FromContext(ctx).Debug("Marked category entries as read",
		slog.Int64("user_id", userID),
		slog.Int64("category_id", categoryID),
		slog.Int64("nb_entries", result.RowsAffected()),
		slog.String("before", before.Format(time.RFC3339)))
	return result.RowsAffected() != 0, nil
}

// EntryShareCode returns the share code of the provided entry. It generates a
// new one if not already defined.
func (s *Storage) EntryShareCode(ctx context.Context, userID, entryID int64,
) (string, error) {
	rows, _ := s.db.Query(ctx,
		`SELECT share_code FROM entries WHERE user_id=$1 AND id=$2`,
		userID, entryID)

	shareCode, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[string])
	if err != nil {
		return "", fmt.Errorf(
			`store: unable to get share code for entry #%d: %w`, entryID, err)
	}

	if shareCode != "" {
		return shareCode, nil
	}
	shareCode = crypto.GenerateRandomStringHex(20)

	_, err = s.db.Exec(ctx,
		`UPDATE entries SET share_code = $1 WHERE user_id=$2 AND id=$3`,
		shareCode, userID, entryID)
	if err != nil {
		return "", fmt.Errorf(`store: unable to set share code for entry #%d: %w`,
			entryID, err)
	}
	return shareCode, nil
}

// UnshareEntry removes the share code for the given entry.
func (s *Storage) UnshareEntry(ctx context.Context, userID, entryID int64,
) error {
	_, err := s.db.Exec(ctx,
		`UPDATE entries SET share_code='' WHERE user_id=$1 AND id=$2`,
		userID, entryID)
	if err != nil {
		return fmt.Errorf(
			`store: unable to remove share code for entry #%d: %w`, entryID, err)
	}
	return nil
}

func (s *Storage) KnownEntryHashes(ctx context.Context, feedID int64,
	hashes []string,
) ([]string, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	rows, _ := s.db.Query(ctx,
		`SELECT hash FROM entries WHERE feed_id = $1 AND hash = ANY($2)`,
		feedID, hashes)
	known, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf(
			"storage: check entries exist: feed=%v hashes=%v: %w",
			feedID, len(hashes), err)
	}
	return known, nil
}
