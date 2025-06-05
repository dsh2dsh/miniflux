// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	json_parser "encoding/json"
	"net/http"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/proxyrotator"
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

	ctx := r.Context()
	var rssbridgeURL, rssbridgeToken string
	intg, err := h.store.Integration(ctx, request.UserID(r))
	if err == nil && intg != nil && intg.RSSBridgeEnabled {
		rssbridgeURL = intg.RSSBridgeURL
		rssbridgeToken = intg.Extra.RSSBridgeToken
	}

	requestBuilder := fetcher.NewRequestBuilder().
		WithTimeout(config.Opts.HTTPClientTimeout()).
		WithProxyRotator(proxyrotator.ProxyRotatorInstance).
		WithCustomFeedProxyURL(discovery.ProxyURL).
		WithCustomApplicationProxyURL(config.Opts.HTTPClientProxyURL()).
		UseCustomApplicationProxyURL(discovery.FetchViaProxy).
		WithUserAgent(discovery.UserAgent, config.Opts.HTTPClientUserAgent()).
		WithCookie(discovery.Cookie).
		WithUsernameAndPassword(discovery.Username, discovery.Password).
		IgnoreTLSErrors(discovery.AllowSelfSignedCertificates).
		DisableHTTP2(discovery.DisableHTTP2)

	s, lerr := subscription.NewSubscriptionFinder(requestBuilder).
		FindSubscriptions(ctx, discovery.URL, rssbridgeURL, rssbridgeToken)
	if lerr != nil {
		json.ServerError(w, r, lerr)
		return
	} else if len(s) == 0 {
		json.NotFound(w, r)
		return
	}
	json.OK(w, r, s)
}
