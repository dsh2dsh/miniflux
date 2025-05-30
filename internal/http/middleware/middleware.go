// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/logging"
)

type MiddlewareFunc func(next http.Handler) http.Handler

func ClientIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-Proto") == "https" {
			config.Opts.EnableHTTPS()
		}

		if config.Opts.HTTPS() && config.Opts.HasHSTS() {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}

		ctx := request.WithClientIP(r.Context(), request.FindClientIP(r))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func WithAccessLog(m map[string]struct{}) MiddlewareFunc {
	fn := func(next http.Handler) http.Handler {
		return &AccessLog{m: m, next: next}
	}
	return fn
}

type AccessLog struct {
	m    map[string]struct{}
	next http.Handler
}

func (self *AccessLog) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	self.next.ServeHTTP(w, r)

	ctx := r.Context()
	log := logging.FromContext(ctx).With(
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("proto", r.Proto))

	if u := request.User(r); u != nil {
		log = log.With(slog.Group("user",
			slog.Int64("id", u.ID),
			slog.String("name", u.Username)))
	}

	methodURL := r.Method + " " + r.URL.String()
	log.LogAttrs(ctx, self.level(r), methodURL,
		slog.Duration("elapsed_time", time.Since(startTime)))
}

func (self *AccessLog) level(r *http.Request) slog.Level {
	p := r.URL.Path
	if _, ok := self.m[p]; ok {
		return slog.LevelDebug
	}

	for s := range self.m {
		if strings.HasPrefix(p, s) {
			return slog.LevelDebug
		}
	}
	return slog.LevelInfo
}
