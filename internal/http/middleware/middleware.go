// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"net/http"

	"github.com/klauspost/compress/gzhttp"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/mux"
	"miniflux.app/v2/internal/http/request"
)

type MiddlewareFunc = mux.MiddlewareFunc

func ClientIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-Proto") == "https" {
			config.Opts.EnableHTTPS()
		}

		if config.Opts.HTTPS() && config.Opts.HasHSTS() {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}

		ctx := r.Context()
		clientIP := request.FindRemoteIP(r)
		if config.Opts.TrustedProxy(clientIP) {
			clientIP = request.FindClientIP(r, config.Opts.TrustedProxy)
		}
		next.ServeHTTP(w, r.WithContext(request.WithClientIP(ctx, clientIP)))
	})
}

func Gzip(next http.Handler) http.Handler {
	return gzhttp.GzipHandler(next)
}
