// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	json_parser "encoding/json"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/reader/subscription"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) discoverSubscriptions(w http.ResponseWriter, r *http.Request,
) (subscription.Subscriptions, error) {
	var discovery model.SubscriptionDiscoveryRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&discovery); err != nil {
		return nil, response.WrapBadRequest(err)
	}

	if lerr := validator.ValidateSubscriptionDiscovery(&discovery); lerr != nil {
		return nil, response.WrapBadRequest(lerr.Error())
	}

	user := request.User(r)
	requestBuilder := NewRequestDiscovery(&discovery)

	s, lerr := subscription.NewSubscriptionFinder(requestBuilder).
		FindSubscriptions(r.Context(), discovery.URL,
			user.Integration().RSSBridgeURLIfEnabled(),
			user.Integration().RSSBridgeTokenIfEnabled())
	if lerr != nil {
		return nil, lerr
	} else if len(s) == 0 {
		return nil, response.ErrNotFound
	}
	return s, nil
}

func NewRequestDiscovery(d *model.SubscriptionDiscoveryRequest,
) *fetcher.RequestBuilder {
	return fetcher.NewRequestBuilder().
		DisableHTTP2(d.DisableHTTP2).
		IgnoreTLSErrors(d.AllowSelfSignedCertificates).
		UseCustomApplicationProxyURL(d.FetchViaProxy).
		WithCookie(d.Cookie).
		WithCustomFeedProxyURL(d.ProxyURL).
		WithUserAgent(d.UserAgent).
		WithUsernameAndPassword(d.Username, d.Password)
}
