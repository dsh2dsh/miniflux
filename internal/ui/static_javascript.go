// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"
	"strings"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/ui/static"
)

const (
	licensePrefix = "//@license magnet:?xt=urn:btih:8e4f440f4c65981c5bf93c76d35135ba5064d8b7&dn=apache-2.0.txt Apache-2.0\n"
	licenseSuffix = "\n//@license-end"
)

func (h *handler) showJavascript(w http.ResponseWriter, r *http.Request) {
	filename := request.RouteStringParam(r, "name")
	b, found := static.JavascriptBundles[filename]
	if !found {
		html.NotFound(w, r)
		return
	}

	if filename == "service-worker" {
		variables := `const OFFLINE_URL="` + route.Path(h.router, "offline") + `";`
		b = append([]byte(variables), b...)
	}

	// cloning the prefix since `append` mutates its first argument
	b = append([]byte(strings.Clone(licensePrefix)), b...)
	b = append(b, []byte(licenseSuffix)...)

	response.New(w, r).
		WithLongCaching().
		WithHeader("Content-Type", "text/javascript; charset=utf-8").
		WithBody(b).
		Write()
}
