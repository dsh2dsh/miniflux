// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

// Job represents a payload sent to the processing queue.
type Job struct {
	UserID  int64  `db:"user_id"`
	FeedID  int64  `db:"id"`
	FeedURL string `db:"feed_url"`
}
