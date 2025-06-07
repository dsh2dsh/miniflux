// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/integration"
	"miniflux.app/v2/internal/model"
)

func (h *handler) saveEntry(w http.ResponseWriter, r *http.Request) {
	user := request.User(r)
	id := request.RouteInt64Param(r, "entryID")
	builder := h.store.NewEntryQueryBuilder(user.ID).
		WithEntryID(id).
		WithoutStatus(model.EntryStatusRemoved)

	entry, err := builder.GetEntry(r.Context())
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if entry == nil {
		json.NotFound(w, r)
		return
	}

	integration.SendEntry(entry, user)
	json.Created(w, r, map[string]string{"message": "saved"})
}
