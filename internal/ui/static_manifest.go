// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/static"
)

func (h *handler) showWebManifest(w http.ResponseWriter, r *http.Request) {
	type webManifestShareTargetParams struct {
		URL  string `json:"url"`
		Text string `json:"text"`
	}

	type webManifestShareTarget struct {
		Action  string                       `json:"action"`
		Method  string                       `json:"method"`
		Enctype string                       `json:"enctype"`
		Params  webManifestShareTargetParams `json:"params"`
	}

	type webManifestIcon struct {
		Source  string `json:"src"`
		Sizes   string `json:"sizes"`
		Type    string `json:"type"`
		Purpose string `json:"purpose"`
	}

	type webManifest struct {
		Name            string                 `json:"name"`
		Description     string                 `json:"description"`
		ShortName       string                 `json:"short_name"`
		StartURL        string                 `json:"start_url"`
		Icons           []webManifestIcon      `json:"icons"`
		ShareTarget     webManifestShareTarget `json:"share_target"`
		Display         string                 `json:"display"`
		BackgroundColor string                 `json:"background_color"`
	}

	displayMode := "standalone"
	if user := request.User(r); user != nil {
		displayMode = user.DisplayMode
	}

	json.OK(w, r, &webManifest{
		Name:            "Miniflux",
		ShortName:       "Miniflux",
		Description:     "Minimalist Feed Reader",
		Display:         displayMode,
		StartURL:        route.Path(h.router, "login"),
		BackgroundColor: model.ThemeColor(request.UserTheme(r), "light"),
		Icons: []webManifestIcon{
			{
				Source: route.Path(h.router, "appIcon", "filename",
					static.BinaryFileName("icon-120.png")),
				Sizes:   "120x120",
				Type:    "image/png",
				Purpose: "any",
			},
			{
				Source: route.Path(h.router, "appIcon", "filename",
					static.BinaryFileName("icon-192.png")),
				Sizes:   "192x192",
				Type:    "image/png",
				Purpose: "any",
			},
			{
				Source: route.Path(h.router, "appIcon", "filename",
					static.BinaryFileName("icon-512.png")),
				Sizes:   "512x512",
				Type:    "image/png",
				Purpose: "any",
			},
			{
				Source: route.Path(h.router, "appIcon", "filename",
					static.BinaryFileName("maskable-icon-120.png")),
				Sizes:   "120x120",
				Type:    "image/png",
				Purpose: "maskable",
			},
			{
				Source: route.Path(h.router, "appIcon", "filename",
					static.BinaryFileName("maskable-icon-192.png")),
				Sizes:   "192x192",
				Type:    "image/png",
				Purpose: "maskable",
			},
			{
				Source: route.Path(h.router, "appIcon", "filename",
					static.BinaryFileName("maskable-icon-512.png")),
				Sizes:   "512x512",
				Type:    "image/png",
				Purpose: "maskable",
			},
		},
		ShareTarget: webManifestShareTarget{
			Action:  route.Path(h.router, "bookmarklet"),
			Method:  http.MethodGet,
			Enctype: "application/x-www-form-urlencoded",
			Params:  webManifestShareTargetParams{URL: "uri", Text: "text"},
		},
	})
}
