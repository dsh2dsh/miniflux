// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"log/slog"
	"net/http"
	"strings"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/proxyrotator"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/reader/opml"
)

func (h *handler) uploadOPML(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		slog.Error("OPML file upload error",
			slog.Int64("user_id", v.User().ID),
			slog.Any("error", err))
		html.Redirect(w, r, route.Path(h.router, "import"))
		return
	}
	defer file.Close()

	slog.Info("OPML file uploaded",
		slog.Int64("user_id", v.User().ID),
		slog.String("file_name", fileHeader.Filename),
		slog.Int64("file_size", fileHeader.Size))

	v.Set("menu", "feeds")
	if fileHeader.Size == 0 {
		lerr := locale.NewLocalizedError("error.empty_file")
		v.Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("import"))
		return
	}

	err = opml.NewHandler(h.store).Import(r.Context(), v.User().ID, file)
	if err != nil {
		v.Set("errorMessage", err)
		html.OK(w, r, v.Render("import"))
		return
	}
	html.Redirect(w, r, route.Path(h.router, "feeds"))
}

func (h *handler) fetchOPML(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	opmlFileURL := strings.TrimSpace(r.FormValue("url"))
	if opmlFileURL == "" {
		html.Redirect(w, r, route.Path(h.router, "import"))
		return
	}

	slog.Info("Fetching OPML file remotely",
		slog.Int64("user_id", v.User().ID),
		slog.String("opml_file_url", opmlFileURL))

	requestBuilder := fetcher.NewRequestBuilder().
		WithTimeout(config.Opts.HTTPClientTimeout()).
		WithProxyRotator(proxyrotator.ProxyRotatorInstance)

	//nolint:bodyclose // responseHandler close it
	responseHandler := fetcher.NewResponseHandler(
		requestBuilder.ExecuteRequest(opmlFileURL))
	defer responseHandler.Close()

	v.Set("menu", "feeds")

	if lerr := responseHandler.LocalizedError(); lerr != nil {
		slog.Warn("Unable to fetch OPML file",
			slog.String("opml_file_url", opmlFileURL),
			slog.Any("error", lerr))
		v.Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("import"))
		return
	}

	err := opml.NewHandler(h.store).Import(r.Context(), v.User().ID,
		responseHandler.Body(config.Opts.HTTPClientMaxBodySize()))
	if err != nil {
		v.Set("errorMessage", err)
		html.OK(w, r, v.Render("import"))
		return
	}
	html.Redirect(w, r, route.Path(h.router, "feeds"))
}
