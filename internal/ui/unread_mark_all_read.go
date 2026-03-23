// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
)

func (h *handler) markAllAsRead(w http.ResponseWriter, r *http.Request,
) (string, error) {
	err := h.store.MarkGloballyVisibleFeedsAsRead(r.Context(), request.UserID(r))
	if err != nil {
		return "", response.WrapServerError(err)
	}
	return "OK", nil
}
