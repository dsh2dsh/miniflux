// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package icon // import "miniflux.app/v2/internal/reader/icon"

import (
	"context"
	"log/slog"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/storage"
)

type IconChecker struct {
	store *storage.Storage
	feed  *model.Feed
}

func NewIconChecker(store *storage.Storage, feed *model.Feed) *IconChecker {
	return &IconChecker{
		store: store,
		feed:  feed,
	}
}

func (c *IconChecker) fetchAndStoreIcon(ctx context.Context) {
	log := logging.FromContext(ctx).With(
		slog.Int64("feed_id", c.feed.ID),
		slog.String("website_url", c.feed.SiteURL),
		slog.String("feed_icon_url", c.feed.IconURL))

	iconFinder, err := NewIconFinder(fetcher.NewRequestFeed(c.feed),
		c.feed.SiteURL, c.feed.IconURL, config.Opts.PreferSiteIcon())
	if err != nil {
		log.Debug("Unable to find feed icon", slog.Any("error", err))
		return
	}

	icon, err := iconFinder.FindIcon(ctx)
	if err != nil {
		log.Debug("Unable to find feed icon", slog.Any("error", err))
		return
	} else if icon == nil {
		log.Debug("No icon found")
		return
	}

	if err := c.store.StoreFeedIcon(ctx, c.feed.ID, icon); err != nil {
		log.Error("Unable to store feed icon", slog.Any("error", err))
		return
	}

	log.Debug("Feed icon stored", slog.GroupAttrs("icon",
		slog.Int64("id", icon.ID),
		slog.String("hash", icon.Hash)))
}

func (c *IconChecker) CreateFeedIconIfMissing(ctx context.Context) {
	if c.store.HasFeedIcon(ctx, c.feed.ID) {
		logging.FromContext(ctx).Debug("Feed icon already exists",
			slog.Int64("feed_id", c.feed.ID))
		return
	}
	c.fetchAndStoreIcon(ctx)
}

func (c *IconChecker) UpdateOrCreateFeedIcon(ctx context.Context) {
	c.fetchAndStoreIcon(ctx)
}
