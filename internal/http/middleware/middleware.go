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
		clientIP := request.FindRemoteIP(r)
		if config.TrustedProxy(clientIP) {
			clientIP = request.FindClientIP(r, config.TrustedProxy)
			if r.Header.Get("X-Forwarded-Proto") == "https" {
				config.EnableHTTPS()
			}
		}

		if config.HTTPS() && config.HasHSTS() {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		ctx := request.WithClientIP(r.Context(), clientIP)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Gzip(next http.Handler) http.Handler {
	return gzhttp.GzipHandler(next)
}

func CrossOriginProtection() MiddlewareFunc {
	c := http.NewCrossOriginProtection()
	if err := c.AddTrustedOrigin(config.RootURL()); err != nil {
		panic(err)
	}
	return func(next http.Handler) http.Handler { return c.Handler(next) }
}
