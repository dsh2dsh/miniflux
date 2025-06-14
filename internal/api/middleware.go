// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/http/middleware"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/storage"
)

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods",
			"GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers",
			"X-Auth-Token, Authorization, Content-Type, Accept")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Max-Age", "3600")
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := request.User(r)
		if user == nil {
			logging.FromContext(ctx).Warn(
				"[API] No Basic HTTP Authentication header sent with the request",
				slog.Bool("authentication_failed", true))
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			json.Unauthorized(w, r)
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
	fn := func(next http.Handler) http.Handler {
		return &keyAuth{store: store, next: next}
	}
	return fn
}

type keyAuth struct {
	store *storage.Storage
	next  http.Handler
}

func (self *keyAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if user := request.User(r); user != nil {
		self.next.ServeHTTP(w, r)
		return
	}

	token := r.Header.Get("X-Auth-Token")
	if token == "" {
		self.next.ServeHTTP(w, r)
		return
	}

	clientIP := request.ClientIP(r)
	ctx := r.Context()
	log := logging.FromContext(ctx).With(
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()))

	user, apiKey, err := self.store.UserAPIKey(ctx, token)
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if user == nil {
		log.Warn("[API] No user found with the provided API key",
			slog.Bool("authentication_failed", true))
		json.Unauthorized(w, r)
		return
	}

	log = log.With(slog.String("username", user.Username))
	log.Debug(
		"[API] User authenticated successfully with the API Token Authentication",
		slog.Bool("authentication_successful", true))

	g, ctx := errgroup.WithContext(ctx)
	userLastLogin := user.LastLoginAt
	if userLastLogin == nil || time.Since(*userLastLogin) > 5*time.Minute {
		g.Go(func() error {
			if err := self.store.SetLastLogin(ctx, user.ID); err != nil {
				log.Error("[API] failed set last login", slog.Any("error", err))
				return err
			}
			return nil
		})
	}

	keyLastUsed := apiKey.LastUsedAt
	if keyLastUsed == nil || time.Since(*keyLastUsed) > 5*time.Minute {
		g.Go(func() error {
			err := self.store.SetAPIKeyUsedTimestamp(ctx, user.ID, token)
			if err != nil {
				log.Error("[API] failed set key used timestamp", slog.Any("error", err))
				return err
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		json.ServerError(w, r, err)
		return
	}
	self.next.ServeHTTP(w, r.WithContext(request.WithUser(ctx, user)))
}

func WithBasicAuth(store *storage.Storage) middleware.MiddlewareFunc {
	fn := func(next http.Handler) http.Handler {
		return &basicAuth{store: store, next: next}
	}
	return fn
}

type basicAuth struct {
	store *storage.Storage
	next  http.Handler
}

func (self *basicAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if user := request.User(r); user != nil {
		self.next.ServeHTTP(w, r)
		return
	} else if auth := r.Header.Get("Authorization"); auth == "" {
		self.next.ServeHTTP(w, r)
		return
	}

	clientIP := request.ClientIP(r)
	ctx := r.Context()
	log := logging.FromContext(ctx).With(
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()))

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	username, password, authOK := r.BasicAuth()
	if !authOK {
		log.Warn(
			"[API] No Basic HTTP Authentication header sent with the request",
			slog.Bool("authentication_failed", true))
		json.Unauthorized(w, r)
		return
	}

	if username == "" || password == "" {
		log.Warn(
			"[API] Empty username or password provided during Basic HTTP Authentication",
			slog.Bool("authentication_failed", true))
		json.Unauthorized(w, r)
		return
	}
	log = log.With(slog.String("username", username))

	g, ctx := errgroup.WithContext(ctx)
	errNotFound := errors.New("invalid username or password")
	g.Go(func() error {
		if err := self.store.CheckPassword(ctx, username, password); err != nil {
			log.Warn(
				"[API] Invalid username or password provided during Basic HTTP Authentication",
				slog.Bool("authentication_failed", true))
			json.Unauthorized(w, r)
			return fmt.Errorf("%w: %w", errNotFound, err)
		}
		return nil
	})

	var user *model.User
	g.Go(func() (err error) {
		user, err = self.store.UserByUsername(ctx, username)
		return
	})

	if err := g.Wait(); err != nil {
		if errors.Is(err, errNotFound) {
			json.Unauthorized(w, r)
		} else {
			json.ServerError(w, r, err)
		}
		return
	} else if user == nil {
		log.Warn("[API] User not found while using Basic HTTP Authentication",
			slog.Bool("authentication_failed", true))
		json.Unauthorized(w, r)
		return
	}

	log.Debug(
		"[API] User authenticated successfully with the Basic HTTP Authentication",
		slog.Bool("authentication_successful", true))

	ctx = r.Context()
	lastLoginAt := user.LastLoginAt
	if lastLoginAt == nil || time.Since(*lastLoginAt) > 5*time.Minute {
		if err := self.store.SetLastLogin(ctx, user.ID); err != nil {
			log.Error("[API] failed set last login", slog.Any("error", err))
			json.ServerError(w, r, err)
			return
		}
	}
	self.next.ServeHTTP(w, r.WithContext(request.WithUser(ctx, user)))
}
