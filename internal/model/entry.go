// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import (
	"time"
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
	ID          int64         `json:"id" db:"id"`
	UserID      int64         `json:"user_id" db:"user_id"`
	FeedID      int64         `json:"feed_id" db:"feed_id"`
	Status      string        `json:"status" db:"status"`
	Hash        string        `json:"hash" db:"hash"`
	Title       string        `json:"title" db:"title"`
	URL         string        `json:"url" db:"url"`
	CommentsURL string        `json:"comments_url" db:"comments_url"`
	Date        time.Time     `json:"published_at" db:"published_at"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
	ChangedAt   time.Time     `json:"changed_at" db:"changed_at"`
	Content     string        `json:"content" db:"content"`
	Author      string        `json:"author" db:"author"`
	ShareCode   string        `json:"share_code" db:"share_code"`
	Starred     bool          `json:"starred" db:"starred"`
	ReadingTime int           `json:"reading_time" db:"reading_time"`
	Enclosures  EnclosureList `json:"enclosures"`
	Feed        *Feed         `json:"feed,omitempty" db:"feed"`
	Tags        []string      `json:"tags" db:"tags"`
}

func NewEntry() *Entry {
	return &Entry{
		Enclosures: make(EnclosureList, 0),
		Tags:       make([]string, 0),
		Feed: &Feed{
			Category: &Category{},
			Icon:     &FeedIcon{},
		},
	}
}

// ShouldMarkAsReadOnView Return whether the entry should be marked as viewed considering all user settings and entry state.
func (e *Entry) ShouldMarkAsReadOnView(user *User) bool {
	// Already read, no need to mark as read again. Removed entries are not marked as read
	if e.Status != EntryStatusUnread {
		return false
	}

	// There is an enclosure, markAsRead will happen at enclosure completion time, no need to mark as read on view
	if user.MarkReadOnMediaPlayerCompletion && e.Enclosures.ContainsAudioOrVideo() {
		return false
	}

	// The user wants to mark as read on view
	return user.MarkReadOnView
}

// Entries represents a list of entries.
type Entries []*Entry

func (self Entries) Enclosures() []*Enclosure {
	size := 0
	for _, e := range self {
		size += len(e.Enclosures)
	}

	encList := make([]*Enclosure, 0, size)
	for _, e := range self {
		if len(e.Enclosures) > 0 {
			encList = append(encList, e.Enclosures...)
		}
	}
	return encList
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
