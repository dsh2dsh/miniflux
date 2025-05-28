// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/logging"
)

func ClientIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-Proto") == "https" {
			config.Opts.EnableHTTPS()
		}

		if config.Opts.HTTPS() && config.Opts.HasHSTS() {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}

		clientIP := request.FindClientIP(r)
		ctx := context.WithValue(r.Context(), request.ClientIPContextKey, clientIP)

		startTime := time.Now()
		next.ServeHTTP(w, r.WithContext(ctx))

		methodURL := r.Method + " " + r.URL.String()
		logging.FromContext(ctx).Debug(methodURL,
			slog.String("client_ip", clientIP),
			slog.String("protocol", r.Proto),
			slog.Duration("execution_time", time.Since(startTime)))
	})
}
