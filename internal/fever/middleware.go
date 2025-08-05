// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package fever // import "miniflux.app/v2/internal/fever"

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"miniflux.app/v2/internal/http/middleware"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/storage"
)

func requestUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := request.User(r)
		if user == nil {
			logging.FromContext(ctx).Warn("[Fever] No API key provided",
				slog.Bool("authentication_failed", true),
				slog.String("client_ip", request.ClientIP(r)),
				slog.String("user_agent", r.UserAgent()))
			json.OK(w, r, newAuthFailureResponse())
			return
		}

		ctx = context.WithValue(ctx, request.UserIDContextKey, user.ID)
		ctx = context.WithValue(ctx, request.UserTimezoneContextKey, user.Timezone)
		ctx = context.WithValue(ctx, request.IsAdminUserContextKey, user.IsAdmin)
		ctx = context.WithValue(ctx, request.IsAuthenticatedContextKey, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func WithKeyAuth(store *storage.Storage) middleware.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return &keyAuth{store: store, next: next}
	}
}

type keyAuth struct {
	store *storage.Storage
	next  http.Handler
}

func (self *keyAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if user := request.User(r); user != nil {
		middleware.AccessLogUser(ctx, user)
		self.next.ServeHTTP(w, r)
		return
	}

	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		self.next.ServeHTTP(w, r)
		return
	}

	log := logging.FromContext(ctx).With(
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))

	user, err := self.store.UserByFeverToken(ctx, apiKey)
	if err != nil {
		log.Error("[Fever] Unable to fetch user by API key",
			slog.Bool("authentication_failed", true),
			slog.Any("error", err))
		json.OK(w, r, newAuthFailureResponse())
		return
	}

	if user == nil || !user.Integration().FeverEnabled {
		log.Warn("[Fever] No user found with the API key provided",
			slog.Bool("authentication_failed", true))
		json.OK(w, r, newAuthFailureResponse())
		return
	}
	middleware.AccessLogUser(ctx, user)

	log = log.With(slog.Group("user",
		slog.Int64("id", user.ID),
		slog.String("name", user.Username)))
	log.Debug("[Fever] User authenticated successfully",
		slog.Bool("authentication_successful", true))

	userLastLogin := user.LastLoginAt
	if userLastLogin == nil || time.Since(*userLastLogin) > 5*time.Minute {
		if err := self.store.SetLastLogin(ctx, user.ID); err != nil {
			log.Error("[Fever] Failed set last login", slog.Any("error", err))
			json.OK(w, r, newAuthFailureResponse())
			return
		}
	}
	self.next.ServeHTTP(w, r.WithContext(request.WithUser(ctx, user)))
}
