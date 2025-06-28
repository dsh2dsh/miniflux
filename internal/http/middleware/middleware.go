// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/klauspost/compress/gzhttp"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/mux"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/storage"
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

func WithAccessLog(prefixes ...string) MiddlewareFunc {
	m := make(map[string]struct{})
	for _, prefix := range prefixes {
		m[prefix] = struct{}{}
	}

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
	ctx := storage.WithTraceStat(r.Context())
	startTime := time.Now()
	self.next.ServeHTTP(w, r.WithContext(ctx))

	log := logging.FromContext(ctx).With(
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("proto", r.Proto))

	if u := request.User(r); u != nil {
		log = log.With(slog.Group("user",
			slog.Int64("id", u.ID),
			slog.String("name", u.Username)))
	}

	if t := storage.TraceStatFrom(ctx); t != nil && t.Queries > 0 {
		log = log.With(slog.Group("storage",
			slog.Int64("queries", t.Queries),
			slog.Duration("elapsed", t.Elapsed)))
	}

	methodURL := r.Method + " " + r.URL.Path
	log.LogAttrs(ctx, self.level(r), methodURL,
		slog.Duration("request_time", time.Since(startTime)))
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

func Gzip(next http.Handler) http.Handler {
	return gzhttp.GzipHandler(next)
}
