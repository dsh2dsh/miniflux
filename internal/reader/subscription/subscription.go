// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package subscription // import "miniflux.app/v2/internal/reader/subscription"

import (
	"fmt"
)

// Subscription represents a feed subscription.
type Subscription struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func NewSubscription(title, url string) *Subscription {
	return &Subscription{Title: title, URL: url}
}

func (s Subscription) String() string {
	return fmt.Sprintf(`Title=%q, URL=%q`, s.Title, s.URL)
}

// Subscriptions represents a list of subscription.
type Subscriptions []*Subscription
