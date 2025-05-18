// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"
	"runtime"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/version"
)

func (h *handler) showAboutPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "settings").
		Set("version", version.Version).
		Set("commit", version.Commit).
		Set("build_date", version.BuildDate).
		Set("globalConfigOptions", config.Opts.SortedOptions(true)).
		Set("postgres_version", h.store.DatabaseVersion(r.Context())).
		Set("go_version", runtime.Version())

	dbSize, dbErr := h.store.DBSize(r.Context())
	if dbErr != nil {
		v.Set("db_usage", dbErr)
	} else {
		v.Set("db_usage", dbSize)
	}
	html.OK(w, r, v.Render("about"))
}
