// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package form // import "miniflux.app/v2/internal/ui/form"

import (
	"net/http"
)

// CategoryForm represents a feed form in the UI
type CategoryForm struct {
	Title        string
	HideGlobally bool
	HideLabel    bool
	BlockFilter  string
}

// NewCategoryForm returns a new CategoryForm.
func NewCategoryForm(r *http.Request) *CategoryForm {
	return &CategoryForm{
		Title:        r.FormValue("title"),
		HideGlobally: r.FormValue("hide_globally") == "1",
		HideLabel:    r.FormValue("hideLabel") == "1",
		BlockFilter:  r.FormValue("blockFilter"),
	}
}
