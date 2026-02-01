// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package validator // import "miniflux.app/v2/internal/validator"

import (
	"context"

	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/filter"
	"miniflux.app/v2/internal/storage"
)

// ValidateFeedCreation validates feed creation.
func ValidateFeedCreation(ctx context.Context, store *storage.Storage,
	userID int64, r *model.FeedCreationRequest,
) *locale.LocalizedError {
	if r.FeedURL == "" || r.CategoryID <= 0 {
		return locale.NewLocalizedError("error.feed_mandatory_fields")
	}

	if !IsValidURL(r.FeedURL) {
		return locale.NewLocalizedError("error.invalid_feed_url")
	}

	if store.FeedURLExists(ctx, userID, r.FeedURL) {
		return locale.NewLocalizedError("error.feed_already_exists")
	}

	if !store.CategoryIDExists(ctx, userID, r.CategoryID) {
		return locale.NewLocalizedError("error.feed_category_not_found")
	}

	if r.ProxyURL != "" && !IsValidURL(r.ProxyURL) {
		return locale.NewLocalizedError("error.invalid_feed_proxy_url")
	}

	if r.BlockFilterEntryRules != "" {
		if _, err := filter.New(r.BlockFilterEntryRules); err != nil {
			return locale.NewLocalizedError(
				"The block list rule is invalid: " + err.Error())
		}
	}

	if r.KeepFilterEntryRules != "" {
		if _, err := filter.New(r.KeepFilterEntryRules); err != nil {
			return locale.NewLocalizedError(
				"The keep list rule is invalid: " + err.Error())
		}
	}
	return nil
}

// ValidateFeedModification validates feed modification.
func ValidateFeedModification(ctx context.Context, store *storage.Storage,
	userID, feedID int64, r *model.FeedModificationRequest,
) *locale.LocalizedError {
	if r.FeedURL != nil {
		if *r.FeedURL == "" {
			return locale.NewLocalizedError("error.feed_url_not_empty")
		}

		if !IsValidURL(*r.FeedURL) {
			return locale.NewLocalizedError("error.invalid_feed_url")
		}

		if store.AnotherFeedURLExists(ctx, userID, feedID, *r.FeedURL) {
			return locale.NewLocalizedError("error.feed_already_exists")
		}
	}

	if r.SiteURL != nil {
		if *r.SiteURL == "" {
			return locale.NewLocalizedError("error.site_url_not_empty")
		}

		if !IsValidURL(*r.SiteURL) {
			return locale.NewLocalizedError("error.invalid_site_url")
		}
	}

	if r.Title != nil {
		if *r.Title == "" {
			return locale.NewLocalizedError("error.feed_title_not_empty")
		}
	}

	if r.CategoryID != nil {
		if !store.CategoryIDExists(ctx, userID, *r.CategoryID) {
			return locale.NewLocalizedError("error.feed_category_not_found")
		}
	}

	if r.ProxyURL != nil {
		if *r.ProxyURL == "" {
			return locale.NewLocalizedError("error.proxy_url_not_empty")
		}

		if !IsValidURL(*r.ProxyURL) {
			return locale.NewLocalizedError("error.invalid_feed_proxy_url")
		}
	}

	if s := model.OptionalValue(r.CommentsURLTemplate); s != "" {
		var f model.Feed
		_, err := f.WithCommentsURLTemplate(s).CommentsURLTemplate()
		if err != nil {
			return locale.NewLocalizedError(
				"Invalid Comments URL template: " + err.Error())
		}
	}

	if s := model.OptionalValue(r.BlockFilterEntryRules); s != "" {
		if _, err := filter.New(s); err != nil {
			return locale.NewLocalizedError(
				"The block list rule is invalid: " + err.Error())
		}
	}

	if s := model.OptionalValue(r.KeepFilterEntryRules); s != "" {
		if _, err := filter.New(s); err != nil {
			return locale.NewLocalizedError(
				"The keep list rule is invalid: " + err.Error())
		}
	}
	return nil
}
