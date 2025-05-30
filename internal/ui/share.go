// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"
	"time"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/ui/view"
)

func (h *handler) createSharedEntry(w http.ResponseWriter, r *http.Request) {
	id := request.RouteInt64Param(r, "entryID")
	shareCode, err := h.store.EntryShareCode(r.Context(), request.UserID(r), id)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	html.Redirect(w, r,
		route.Path(h.router, "sharedEntry", "shareCode", shareCode))
}

func (h *handler) unshareEntry(w http.ResponseWriter, r *http.Request) {
	id := request.RouteInt64Param(r, "entryID")
	err := h.store.UnshareEntry(r.Context(), request.UserID(r), id)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	html.Redirect(w, r, route.Path(h.router, "sharedEntries"))
}

func (h *handler) sharedEntry(w http.ResponseWriter, r *http.Request) {
	shareCode := request.RouteStringParam(r, "shareCode")
	if shareCode == "" {
		html.NotFound(w, r)
		return
	}

	etag := shareCode
	response.New(w, r).WithCaching(etag, 72*time.Hour, func(b *response.Builder) {
		builder := h.store.NewAnonymousQueryBuilder().
			WithShareCode(shareCode)

		entry, err := builder.GetEntry(r.Context())
		if err != nil || entry == nil {
			html.NotFound(w, r)
			return
		}

		v := view.New(h.tpl, r, nil).
			Set("entry", entry)

		b.WithHeader("Content-Type", "text/html; charset=utf-8")
		b.WithBody(v.Render("entry"))
		b.Write()
	})
}
