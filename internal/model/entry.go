// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import (
	"bytes"
	"context"
	"fmt"
	"iter"
	"log/slog"
	"net/url"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/dsh2dsh/gofeed/v2/atom"

	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/logging"
)

// Entry statuses and default sorting order.
const (
	EntryStatusUnread       = "unread"
	EntryStatusRead         = "read"
	EntryStatusRemoved      = "removed"
	DefaultSortingOrder     = "published_at"
	DefaultSortingDirection = "asc"
)

// Entry represents a feed item in the system.
type Entry struct {
	ID          int64      `json:"id" db:"id"`
	UserID      int64      `json:"user_id" db:"user_id"`
	FeedID      int64      `json:"feed_id" db:"feed_id"`
	Status      string     `json:"status" db:"status"`
	Hash        string     `json:"hash" db:"hash"`
	Title       string     `json:"title" db:"title"`
	URL         string     `json:"url" db:"url"`
	CommentsURL string     `json:"comments_url" db:"comments_url"`
	Date        time.Time  `json:"published_at" db:"published_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	ChangedAt   time.Time  `json:"changed_at" db:"changed_at"`
	Content     string     `json:"content" db:"content"`
	Author      string     `json:"author" db:"author"`
	Starred     bool       `json:"starred" db:"starred"`
	ReadingTime int        `json:"reading_time" db:"reading_time"`
	Feed        *Feed      `json:"feed,omitempty" db:"feed"`
	Tags        []string   `json:"tags" db:"tags"`
	Extra       EntryExtra `json:"extra,omitzero" db:"extra"`

	atom     *atom.Entry
	imported bool
	stored   bool
}

type EntryExtra struct {
	Enclosures EnclosureList `json:"enclosures,omitempty"`
}

func (self *Entry) WithAtom(atom *atom.Entry) *Entry {
	self.atom = atom
	return self
}

func (self *Entry) Atom() *atom.Entry { return self.atom }

// ShouldMarkAsReadOnView Return whether the entry should be marked as viewed
// considering all user settings and entry state.
func (self *Entry) ShouldMarkAsReadOnView(user *User) bool {
	// Already read, no need to mark as read again. Removed entries are not marked
	// as read
	if self.Status != EntryStatusUnread {
		return false
	}

	// There is an enclosure, markAsRead will happen at enclosure completion time,
	// no need to mark as read on view
	if user.MarkReadOnMediaPlayerCompletion && self.Enclosures().ContainsAudioOrVideo() {
		return false
	}

	// The user wants to mark as read on view
	return user.MarkReadOnView
}

func (self *Entry) SetCommentsURL(rawURL string) (err error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		self.CommentsURL = rawURL
		return nil
	}

	var u *url.URL
	switch {
	case path.IsAbs(rawURL):
		u, err = url.Parse(self.URL)
		if err != nil {
			return fmt.Errorf("model: parse entry url %q: %w", self.URL, err)
		}
		u = u.JoinPath(rawURL)
	default:
		u, err = url.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("model: parse new comments url %q: %w", rawURL, err)
		}
		u.Path = path.Clean(u.Path)
	}

	if strings.HasSuffix(rawURL, "/") && !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	self.CommentsURL = u.String()
	return nil
}

func (self *Entry) MarkStored()  { self.stored = true }
func (self *Entry) Stored() bool { return self.stored }

func (self *Entry) Enclosures() EnclosureList { return self.Extra.Enclosures }

func (self *Entry) AppendEnclosures(encList EnclosureList) {
	if len(encList) == 0 {
		return
	}
	if self.Extra.Enclosures == nil {
		self.Extra.Enclosures = encList
		return
	}
	self.Extra.Enclosures = append(self.Extra.Enclosures, encList...)
}

func (self *Entry) HashFrom(s string) { self.Hash = crypto.HashFromString(s) }

func (self *Entry) Imported() bool { return self.imported }

func (self *Entry) KeepImportedStatus(value string) {
	if !self.imported {
		self.Status = value
	}
}

func (self *Entry) Unread() bool { return self.Status == EntryStatusUnread }

func NewEntryFrom(ext *ExternalEntry) *Entry {
	entry := &Entry{
		Status:      ext.Status,
		Title:       ext.Title,
		URL:         ext.URL,
		CommentsURL: ext.CommentsURL,
		Content:     ext.Content,
		Author:      ext.Author,
		Starred:     ext.Starred,
		Tags:        ext.Tags,
		imported:    true,
	}
	entry.HashFrom(ext.URL)

	if ext.PublishedAt > 0 {
		entry.Date = time.Unix(ext.PublishedAt, 0).UTC()
	} else {
		entry.Date = time.Now().UTC()
	}

	if ext.Title == "" {
		entry.Title = ext.URL
	}
	return entry
}

type ImportEntries struct {
	FeedURL string           `json:"feed_url,omitzero"`
	Entries []*ExternalEntry `json:"entries,omitzero"`
}

type ExternalEntry struct {
	Status      string   `json:"status,omitzero"`
	Title       string   `json:"title,omitzero"`
	URL         string   `json:"url,omitzero"`
	CommentsURL string   `json:"comments_url,omitzero"`
	PublishedAt int64    `json:"published_at,omitzero"`
	Content     string   `json:"content,omitzero"`
	Author      string   `json:"author,omitzero"`
	Starred     bool     `json:"starred,omitzero"`
	Tags        []string `json:"tags,omitzero"`
}

// Entries represents a list of entries.
type Entries []*Entry

func (self Entries) Enclosures() []Enclosure {
	size := 0
	for _, e := range self {
		size += len(e.Enclosures())
	}

	encList := make([]Enclosure, 0, size)
	for _, e := range self {
		if len(e.Enclosures()) > 0 {
			encList = append(encList, e.Enclosures()...)
		}
	}
	return encList
}

func (self Entries) MakeCommentURLs(ctx context.Context) {
	log := logging.FromContext(ctx)
	feedTemplates := make(map[int64]*template.Template)
	var b bytes.Buffer

	for _, e := range self {
		t, ok := feedTemplates[e.FeedID]
		if !ok {
			t2, err := e.Feed.CommentsURLTemplate()
			switch {
			case err != nil:
				log.Error("model: failed parse comments_url_template",
					slog.Int64("feed_id", e.FeedID),
					slog.Any("error", err))
				fallthrough
			case t2 == nil:
				feedTemplates[e.FeedID] = nil
				continue
			}

			t = t2
			feedTemplates[e.FeedID] = t
		} else if t == nil {
			continue
		}

		b.Reset()
		if err := t.Execute(&b, e); err != nil {
			log.Error("model: failed execute comments_url_template",
				slog.Int64("feed_id", e.FeedID),
				slog.Int64("entry_id", e.ID),
				slog.Any("error", err))
			feedTemplates[e.FeedID] = nil
			continue
		}

		if err := e.SetCommentsURL(b.String()); err != nil {
			log.Error("model: failed set templated comments url",
				slog.Int64("feed_id", e.FeedID),
				slog.Int64("entry_id", e.ID),
				slog.Any("error", err))
			feedTemplates[e.FeedID] = nil
		}
	}
}

func (self Entries) Unread() iter.Seq2[int, *Entry] {
	return func(yield func(int, *Entry) bool) {
		for i, e := range self {
			if e.Status == EntryStatusUnread {
				if !yield(i, e) {
					return
				}
			}
		}
	}
}

func (self Entries) RefreshFeed(userID, feedID int64) []string {
	for _, e := range self {
		e.UserID, e.FeedID = userID, feedID
		if e.Status == "" {
			e.Status = EntryStatusUnread
		}
	}
	return self.Hashes()
}

func (self Entries) Hashes() []string {
	if len(self) == 0 {
		return nil
	}

	hashes := make([]string, len(self))
	for i, e := range self {
		hashes[i] = e.Hash
	}
	return hashes
}

func (self Entries) ByHash() map[string]*Entry {
	entries := make(map[string]*Entry, len(self))
	for _, e := range self {
		entries[e.Hash] = e
	}
	return entries
}

// EntriesStatusUpdateRequest represents a request to change entries status.
type EntriesStatusUpdateRequest struct {
	EntryIDs []int64 `json:"entry_ids"`
	Status   string  `json:"status"`
}

// EntryUpdateRequest represents a request to update an entry.
type EntryUpdateRequest struct {
	Title   *string `json:"title"`
	Content *string `json:"content"`
}

func (e *EntryUpdateRequest) Patch(entry *Entry) {
	if e.Title != nil && *e.Title != "" {
		entry.Title = *e.Title
	}

	if e.Content != nil && *e.Content != "" {
		entry.Content = *e.Content
	}
}
