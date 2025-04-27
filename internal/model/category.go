// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import "fmt"

// Category represents a feed category.
type Category struct {
	ID           int64  `json:"id" db:"id"`
	Title        string `json:"title" db:"title"`
	UserID       int64  `json:"user_id" db:"user_id"`
	HideGlobally bool   `json:"hide_globally" db:"hide_globally"`
	FeedCount    *int   `json:"feed_count,omitempty" db:"feed_count"`
	TotalUnread  *int   `json:"total_unread,omitempty" db:"total_unread"`
}

func (c *Category) String() string {
	return fmt.Sprintf("ID=%d, UserID=%d, Title=%s", c.ID, c.UserID, c.Title)
}

type CategoryCreationRequest struct {
	Title        string `json:"title"`
	HideGlobally bool   `json:"hide_globally"`
}

type CategoryModificationRequest struct {
	Title        *string `json:"title"`
	HideGlobally *bool   `json:"hide_globally"`
}

func (c *CategoryModificationRequest) Patch(category *Category) {
	if c.Title != nil {
		category.Title = *c.Title
	}

	if c.HideGlobally != nil {
		category.HideGlobally = *c.HideGlobally
	}
}
