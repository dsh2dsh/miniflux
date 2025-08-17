// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"
	"runtime"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/version"
)

func (h *handler) showAboutPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)

	var dbVersion string
	v.Go(func(ctx context.Context) error {
		dbVersion = h.store.DatabaseVersion(ctx)
		return nil
	})

	var dbSize string
	v.Go(func(ctx context.Context) error {
		size, err := h.store.DBSize(ctx)
		if err != nil {
			dbSize = err.Error()
		} else {
			dbSize = size
		}
		return nil
	})

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "settings").
		Set("version", version.New()).
		Set("globalConfigOptions", config.Opts.SortedOptions(true)).
		Set("postgres_version", dbVersion).
		Set("db_usage", dbSize).
		Set("go_version", runtime.Version())
	html.OK(w, r, v.Render("about"))
}
