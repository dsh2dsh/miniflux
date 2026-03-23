// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/reader/opml"
)

func (h *handler) exportFeeds(w http.ResponseWriter, r *http.Request) {
	opmlHandler := opml.NewHandler(h.store)
	opmlExport, err := opmlHandler.Export(r.Context(), request.UserID(r))
	if err != nil {
		response.ServerErrorJSON(w, r, err)
		return
	}

	response.New(w, r).
		WithHeader("Content-Type", "text/xml; charset=utf-8").
		WithBodyAsString(opmlExport).
		Write()
}

func (h *handler) importFeeds(w http.ResponseWriter, r *http.Request,
) (*importFeedsResponse, error) {
	opmlHandler := opml.NewHandler(h.store)
	err := opmlHandler.Import(r.Context(), request.UserID(r), r.Body)
	defer r.Body.Close()
	if err != nil {
		return nil, err
	}
	return &importFeedsResponse{Message: "Feeds imported successfully"}, nil
}
