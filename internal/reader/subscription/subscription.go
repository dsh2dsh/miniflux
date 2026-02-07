// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package subscription // import "miniflux.app/v2/internal/reader/subscription"

import (
	"fmt"
	"log/slog"
	"slices"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/reader/parser"
)

// Subscription represents a feed subscription.
type Subscription struct {
	Title string `json:"title,omitempty"`
	URL   string `json:"url,omitempty"`
}

func NewSubscription(title, url string) *Subscription {
	return &Subscription{Title: title, URL: url}
}

func (self *Subscription) String() string {
	return fmt.Sprintf(`Title=%q, URL=%q`, self.Title, self.URL)
}

func (self *Subscription) Parse(rb *fetcher.RequestBuilder) error {
	resp, err := rb.Request(self.URL)
	if err != nil {
		return err
	}
	defer resp.Close()

	if lerr := resp.LocalizedError(); lerr != nil {
		return lerr
	}

	body, lerr := resp.ReadBody()
	if lerr != nil {
		return lerr
	}
	resp.Close()

	_, err = parser.ParseBytes(resp.EffectiveURL(), body)
	return err
}

// Subscriptions represents a list of subscription.
type Subscriptions []*Subscription

func (self Subscriptions) Parseable(rb *fetcher.RequestBuilder) Subscriptions {
	if len(self) == 0 {
		return self
	}

	log := logging.FromContext(rb.Context()).With(
		slog.Int("concurrency", config.WorkerPoolSize()),
		slog.Int("subscriptions", len(self)))
	log.Debug("keep parseable subscriptions")

	var g errgroup.Group
	g.SetLimit(config.WorkerPoolSize())

	for i := range self {
		g.Go(func() error {
			if !self.parse(i, rb) {
				self[i] = nil
			}
			return nil
		})
	}

	log.Debug("waiting for group completion")
	_ = g.Wait()

	del := func(s *Subscription) bool { return s == nil }
	parseable := slices.DeleteFunc(self, del)
	log.Debug("all subscriptions checked", slog.Int("parseable", len(parseable)))
	return parseable
}

func (self Subscriptions) parse(i int, rb *fetcher.RequestBuilder) bool {
	s := self[i]
	log := logging.FromContext(rb.Context()).With(
		slog.Int("i", i),
		slog.String("url", s.URL))
	log.Debug("parse discovered feed")

	if err := s.Parse(rb); err != nil {
		log.Debug("unable parse discovered feed", slog.Any("error", err))
		return false
	}

	log.Debug("parsed OK discovered feed")
	return true
}
