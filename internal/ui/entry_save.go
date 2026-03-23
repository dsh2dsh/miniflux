// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/integration"
	"miniflux.app/v2/internal/model"
)

func (h *handler) saveEntry(w http.ResponseWriter, r *http.Request,
) (map[string]string, error) {
	user := request.User(r)
	id := request.RouteInt64Param(r, "entryID")
	builder := h.store.NewEntryQueryBuilder(user.ID).
		WithEntryID(id).
		WithoutStatus(model.EntryStatusRemoved)

	entry, err := builder.GetEntry(r.Context())
	if err != nil {
		return nil, response.WrapServerError(err)
	} else if entry == nil {
		return nil, response.ErrNotFound
	}

	integration.SendEntry(r.Context(), entry, user)
	return map[string]string{"message": "saved"}, nil
}
