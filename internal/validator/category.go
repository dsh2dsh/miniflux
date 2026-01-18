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

// ValidateCategoryCreation validates category creation.
func ValidateCategoryCreation(ctx context.Context, store *storage.Storage,
	userID int64, request *model.CategoryCreationRequest,
) *locale.LocalizedError {
	if request.Title == "" {
		return locale.NewLocalizedError("error.title_required")
	}

	if store.CategoryTitleExists(ctx, userID, request.Title) {
		return locale.NewLocalizedError("error.category_already_exists")
	}

	return nil
}

// ValidateCategoryModification validates category modification.
func ValidateCategoryModification(ctx context.Context, store *storage.Storage,
	userID, categoryID int64, r *model.CategoryModificationRequest,
) *locale.LocalizedError {
	if r.Title != nil {
		if *r.Title == "" {
			return locale.NewLocalizedError("error.title_required")
		}

		if store.AnotherCategoryExists(ctx, userID, categoryID, *r.Title) {
			return locale.NewLocalizedError("error.category_already_exists")
		}
	}

	if s := model.OptionalValue(r.BlockFilter); s != "" {
		if _, err := filter.New(s); err != nil {
			return locale.NewLocalizedError(
				"The block list rule is invalid: " + err.Error())
		}
	}
	return nil
}
