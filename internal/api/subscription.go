// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	json_parser "encoding/json"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/reader/subscription"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) discoverSubscriptions(w http.ResponseWriter, r *http.Request,
) {
	var discovery model.SubscriptionDiscoveryRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&discovery); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	if lerr := validator.ValidateSubscriptionDiscovery(&discovery); lerr != nil {
		json.BadRequest(w, r, lerr.Error())
		return
	}

	user := request.User(r)
	requestBuilder := fetcher.NewRequestDiscovery(&discovery)

	s, lerr := subscription.NewSubscriptionFinder(requestBuilder).
		FindSubscriptions(r.Context(), discovery.URL,
			user.Integration().RSSBridgeURLIfEnabled(),
			user.Integration().RSSBridgeTokenIfEnabled())
	if lerr != nil {
		json.ServerError(w, r, lerr)
		return
	} else if len(s) == 0 {
		json.NotFound(w, r)
		return
	}
	json.OK(w, r, s)
}
