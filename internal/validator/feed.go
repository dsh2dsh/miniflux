// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package validator // import "miniflux.app/v2/internal/validator"

import (
	"context"

	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/storage"
)

// ValidateFeedCreation validates feed creation.
func ValidateFeedCreation(ctx context.Context, store *storage.Storage,
	userID int64, request *model.FeedCreationRequest,
) *locale.LocalizedError {
	if request.FeedURL == "" || request.CategoryID <= 0 {
		return locale.NewLocalizedError("error.feed_mandatory_fields")
	}

	if !IsValidURL(request.FeedURL) {
		return locale.NewLocalizedError("error.invalid_feed_url")
	}

	if store.FeedURLExists(ctx, userID, request.FeedURL) {
		return locale.NewLocalizedError("error.feed_already_exists")
	}

	if !store.CategoryIDExists(ctx, userID, request.CategoryID) {
		return locale.NewLocalizedError("error.feed_category_not_found")
	}

	if !IsValidRegex(request.BlocklistRules) {
		return locale.NewLocalizedError("error.feed_invalid_blocklist_rule")
	}

	if !IsValidRegex(request.KeeplistRules) {
		return locale.NewLocalizedError("error.feed_invalid_keeplist_rule")
	}

	if request.ProxyURL != "" && !IsValidURL(request.ProxyURL) {
		return locale.NewLocalizedError("error.invalid_feed_proxy_url")
	}

	return nil
}

// ValidateFeedModification validates feed modification.
func ValidateFeedModification(ctx context.Context, store *storage.Storage,
	userID, feedID int64, request *model.FeedModificationRequest,
) *locale.LocalizedError {
	if request.FeedURL != nil {
		if *request.FeedURL == "" {
			return locale.NewLocalizedError("error.feed_url_not_empty")
		}

		if !IsValidURL(*request.FeedURL) {
			return locale.NewLocalizedError("error.invalid_feed_url")
		}

		if store.AnotherFeedURLExists(ctx, userID, feedID, *request.FeedURL) {
			return locale.NewLocalizedError("error.feed_already_exists")
		}
	}

	if request.SiteURL != nil {
		if *request.SiteURL == "" {
			return locale.NewLocalizedError("error.site_url_not_empty")
		}

		if !IsValidURL(*request.SiteURL) {
			return locale.NewLocalizedError("error.invalid_site_url")
		}
	}

	if request.Title != nil {
		if *request.Title == "" {
			return locale.NewLocalizedError("error.feed_title_not_empty")
		}
	}

	if request.CategoryID != nil {
		if !store.CategoryIDExists(ctx, userID, *request.CategoryID) {
			return locale.NewLocalizedError("error.feed_category_not_found")
		}
	}

	if request.BlocklistRules != nil {
		if !IsValidRegex(*request.BlocklistRules) {
			return locale.NewLocalizedError("error.feed_invalid_blocklist_rule")
		}
	}

	if request.KeeplistRules != nil {
		if !IsValidRegex(*request.KeeplistRules) {
			return locale.NewLocalizedError("error.feed_invalid_keeplist_rule")
		}
	}

	if request.ProxyURL != nil {
		if *request.ProxyURL == "" {
			return locale.NewLocalizedError("error.proxy_url_not_empty")
		}

		if !IsValidURL(*request.ProxyURL) {
			return locale.NewLocalizedError("error.invalid_feed_proxy_url")
		}
	}

	if request.CommentsURLTemplate != nil {
		if s := *request.CommentsURLTemplate; s != "" {
			f := model.Feed{Extra: model.FeedExtra{CommentsURLTemplate: s}}
			_, err := f.CommentsURLTemplate()
			if err != nil {
				return locale.NewLocalizedError(
					"Invalid Comments URL template: " + err.Error())
			}
		}
	}
	return nil
}
