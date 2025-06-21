// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package client // import "miniflux.app/v2/client"

import (
	"fmt"

	"miniflux.app/v2/internal/model"
)

// Subscription represents a feed subscription.
type Subscription struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Type  string `json:"type"`
}

func (s Subscription) String() string {
	return fmt.Sprintf(`Title=%q, URL=%q, Type=%q`, s.Title, s.URL, s.Type)
}

// Subscriptions represents a list of subscriptions.
type Subscriptions []*Subscription

// FeedIcon represents the feed icon.
type FeedIcon struct {
	ID       int64  `json:"id"`
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

const (
	FilterNotStarred  = "0"
	FilterOnlyStarred = "1"
)

// Filter is used to filter entries.
type Filter struct {
	Status          string
	Offset          int
	Limit           int
	Order           string
	Direction       string
	Starred         string
	Before          int64
	After           int64
	PublishedBefore int64
	PublishedAfter  int64
	ChangedBefore   int64
	ChangedAfter    int64
	BeforeEntryID   int64
	AfterEntryID    int64
	Search          string
	CategoryID      int64
	FeedID          int64
	Statuses        []string
	GloballyVisible bool
}

// EntryResultSet represents the response when fetching entries.
type EntryResultSet struct {
	Total   int           `json:"total"`
	Entries model.Entries `json:"entries"`
}

// VersionResponse represents the version and the build information of the
// Miniflux instance.
type VersionResponse struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Compiler  string `json:"compiler"`
	Arch      string `json:"arch"`
	OS        string `json:"os"`
}
