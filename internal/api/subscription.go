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

func (h *handler) discoverSubscriptions(w http.ResponseWriter, r *http.Request) {
	var subscriptionDiscoveryRequest model.SubscriptionDiscoveryRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&subscriptionDiscoveryRequest); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	if validationErr := validator.ValidateSubscriptionDiscovery(&subscriptionDiscoveryRequest); validationErr != nil {
		json.BadRequest(w, r, validationErr.Error())
		return
	}

	var rssbridgeURL, rssbridgeToken string
	intg, err := h.store.Integration(r.Context(), request.UserID(r))
	if err == nil && intg != nil && intg.RSSBridgeEnabled {
		rssbridgeURL = intg.RSSBridgeURL
		rssbridgeToken = intg.Extra.RSSBridgeToken
	}

	requestBuilder := fetcher.NewRequestBuilder()
	requestBuilder.WithTimeout(config.Opts.HTTPClientTimeout())
	requestBuilder.WithProxyRotator(proxyrotator.ProxyRotatorInstance)
	requestBuilder.WithCustomFeedProxyURL(subscriptionDiscoveryRequest.ProxyURL)
	requestBuilder.WithCustomApplicationProxyURL(config.Opts.HTTPClientProxyURL())
	requestBuilder.UseCustomApplicationProxyURL(subscriptionDiscoveryRequest.FetchViaProxy)
	requestBuilder.WithUserAgent(subscriptionDiscoveryRequest.UserAgent, config.Opts.HTTPClientUserAgent())
	requestBuilder.WithCookie(subscriptionDiscoveryRequest.Cookie)
	requestBuilder.WithUsernameAndPassword(subscriptionDiscoveryRequest.Username, subscriptionDiscoveryRequest.Password)
	requestBuilder.IgnoreTLSErrors(subscriptionDiscoveryRequest.AllowSelfSignedCertificates)
	requestBuilder.DisableHTTP2(subscriptionDiscoveryRequest.DisableHTTP2)

	subscriptions, localizedError := subscription.
		NewSubscriptionFinder(requestBuilder).
		FindSubscriptions(r.Context(), subscriptionDiscoveryRequest.URL,
			rssbridgeURL, rssbridgeToken)

	if localizedError != nil {
		json.ServerError(w, r, localizedError)
		return
	}

	if len(subscriptions) == 0 {
		json.NotFound(w, r)
		return
	}

	json.OK(w, r, subscriptions)
}
