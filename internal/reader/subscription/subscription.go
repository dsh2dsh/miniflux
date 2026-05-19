// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package subscription // import "miniflux.app/v2/internal/reader/subscription"

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/reader/parser"
)

// Subscription represents a feed subscription.
type Subscription struct {
	Title string `json:"title,omitempty"`
	URL   string `json:"url,omitempty"`

	err error
}

func NewSubscription(title, url string) *Subscription {
	return &Subscription{Title: title, URL: url}
}

func (self *Subscription) String() string {
	return fmt.Sprintf(`Title=%q, URL=%q`, self.Title, self.URL)
}

func (self *Subscription) Err() error { return self.err }

func (self *Subscription) Parse(ctx context.Context, client *fetcher.Client,
) error {
	resp, err := client.Request(ctx, self.URL)
	if err != nil {
		return locale.NewLocalizedErrorWrapper(err, "error.http_body_read", err)
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

	feed, err := parser.ParseBytes(resp.EffectiveURL(), body)
	if err != nil {
		return err
	}

	self.Title = feed.Title
	return nil
}

// Subscriptions represents a list of subscription.
type Subscriptions []*Subscription

func (self Subscriptions) Parseable(ctx context.Context, client *fetcher.Client,
) Subscriptions {
	if len(self) == 0 {
		return self
	}

	log := logging.FromContext(ctx).With(
		slog.Int("concurrency", config.WorkerPoolSize()),
		slog.Int("subscriptions", len(self)))
	log.Debug("keep only parseable subscriptions")

	var g errgroup.Group
	g.SetLimit(config.WorkerPoolSize())

	for i := range self {
		if ctx.Err() != nil {
			break
		}
		g.Go(func() error {
			self.parse(ctx, i, client)
			return nil
		})
	}

	log.Debug("waiting for group completion")
	if err := g.Wait(); err != nil {
		log.Debug("group completed with error", slog.Any("error", err))
		return nil
	} else if err := context.Cause(ctx); err != nil {
		log.Debug("parsing of subscriptions cancelled", slog.Any("cause", err))
		return nil
	}

	del := func(s *Subscription) bool { return s.Err() != nil }
	parseable := slices.DeleteFunc(self, del)
	log.Debug("all subscriptions checked", slog.Int("parseable", len(parseable)))
	return parseable
}

func (self Subscriptions) parse(ctx context.Context, i int,
	client *fetcher.Client,
) {
	s := self[i]
	log := logging.FromContext(ctx).With(
		slog.Int("i", i),
		slog.String("url", s.URL))
	log.Debug("parse discovered feed")

	if err := s.Parse(ctx, client); err != nil {
		log.Debug("unable parse discovered feed", slog.Any("error", err))
		s.err = err
		return
	}
	log.Debug("parsed OK discovered feed")
}
