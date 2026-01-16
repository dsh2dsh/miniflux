// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import "fmt"

// Category represents a feed category.
type Category struct {
	ID           int64         `json:"id" db:"id"`
	Title        string        `json:"title" db:"title"`
	UserID       int64         `json:"user_id" db:"user_id"`
	HideGlobally bool          `json:"hide_globally" db:"hide_globally"`
	Extra        CategoryExtra `json:"extra,omitzero" db:"extra"`

	// Pointers are needed to avoid breaking /v1/categories?counts=true
	FeedCount   *int `json:"feed_count,omitempty" db:"feed_count"`
	TotalUnread *int `json:"total_unread,omitempty" db:"total_unread"`
}

type CategoryExtra struct {
	HideLabel bool `json:"hide_label,omitempty"`
}

func (self *Category) String() string {
	return fmt.Sprintf("ID=%d, UserID=%d, Title=%s", self.ID, self.UserID,
		self.Title)
}

func (self *Category) HiddenLabel() bool { return self.Extra.HideLabel }

type CategoryCreationRequest struct {
	Title        string `json:"title,omitzero"`
	HideGlobally bool   `json:"hide_globally,omitzero"`
}

type CategoryModificationRequest struct {
	Title        *string `json:"title,omitzero"`
	HideGlobally *bool   `json:"hide_globally,omitzero"`
	HideLabel    *bool   `json:"hide_label,omitzero"`
}

func (self *CategoryModificationRequest) Patch(category *Category) {
	if self.Title != nil {
		category.Title = *self.Title
	}

	if self.HideGlobally != nil {
		category.HideGlobally = *self.HideGlobally
	}

	if self.HideLabel != nil {
		category.Extra.HideLabel = *self.HideLabel
	}
}
