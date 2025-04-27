// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package validator // import "miniflux.app/v2/internal/validator"

import (
	"context"

	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/model"
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
	userID, categoryID int64, request *model.CategoryModificationRequest,
) *locale.LocalizedError {
	if request.Title != nil {
		if *request.Title == "" {
			return locale.NewLocalizedError("error.title_required")
		}

		if store.AnotherCategoryExists(ctx, userID, categoryID, *request.Title) {
			return locale.NewLocalizedError("error.category_already_exists")
		}
	}

	return nil
}
