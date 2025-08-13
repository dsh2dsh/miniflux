// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"
	"regexp"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
)

// Best effort url extraction regexp
var urlRe = regexp.MustCompile(
	`(?i)(?:https?://)?[0-9a-z.]+[.][a-z]+(?::[0-9]+)?(?:/[^ ]+|/)?`)

func (h *handler) bookmarklet(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)

	var categories []model.Category
	v.Go(func(ctx context.Context) (err error) {
		categories, err = h.store.Categories(ctx, v.UserID())
		return
	})

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	bookmarkletURL := request.QueryStringParam(r, "uri", "")

	// Extract URL from text supplied by Web Share Target API.
	//
	// This is because Android intents have no concept of URL, so apps
	// just shove a URL directly into the EXTRA_TEXT intent field.
	//
	// See https://bugs.chromium.org/p/chromium/issues/detail?id=789379.
	text := request.QueryStringParam(r, "text", "")
	if text != "" && bookmarkletURL == "" {
		bookmarkletURL = urlRe.FindString(text)
	}

	v.Set("menu", "feeds").
		Set("form", form.SubscriptionForm{URL: bookmarkletURL}).
		Set("categories", categories).
		Set("defaultUserAgent", config.Opts.HTTPClientUserAgent()).
		Set("hasProxyConfigured", config.Opts.HasHTTPClientProxyURLConfigured())
	html.OK(w, r, v.Render("add_subscription"))
}
