// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package googlereader // import "miniflux.app/v2/internal/googlereader"

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"miniflux.app/v2/internal/http/middleware"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/storage"
)

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods",
			"GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestUserSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := logging.FromContext(ctx).With(
			slog.String("client_ip", request.ClientIP(r)),
			slog.String("user_agent", r.UserAgent()))

		user := request.User(r)
		if user == nil {
			log.Warn(
				"[GoogleReader] No user found with the given Google Reader credentials",
				slog.Bool("authentication_failed", true))
			sendUnauthorizedResponse(w)
			return
		}

		sess := request.Session(r)
		if sess == nil {
			log.Warn(
				"[GoogleReader] No session found with the given Google Reader credentials",
				slog.Bool("authentication_failed", true),
				slog.String("username", user.Username))
			sendUnauthorizedResponse(w)
			return
		}

		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r.WithContext(contextWithSessionKeys(ctx, user, sess)))
			return
		}

		if err := checkCSRF(r, sess.ID); err != nil {
			log.Warn("[GoogleReader] invalid or missing CSRF token",
				slog.Any("error", err))
			sendUnauthorizedResponse(w)
			return
		}
		next.ServeHTTP(w, r.WithContext(contextWithSessionKeys(ctx, user, sess)))
	})
}

func checkCSRF(r *http.Request, expected string) error {
	if token := r.PostFormValue("T"); token == "" {
		return errors.New("token from T is empty")
	} else if token != expected {
		return errors.New("unexpected token")
	}
	return nil
}

func contextWithSessionKeys(ctx context.Context, user *model.User,
	sess *model.Session,
) context.Context {
	ctx = context.WithValue(ctx, request.UserIDContextKey, user.ID)
	ctx = context.WithValue(ctx, request.UserNameContextKey, user.Username)
	ctx = context.WithValue(ctx, request.UserTimezoneContextKey, user.Timezone)
	ctx = context.WithValue(ctx, request.IsAdminUserContextKey, user.IsAdmin)
	ctx = context.WithValue(ctx, request.IsAuthenticatedContextKey, true)
	ctx = context.WithValue(ctx, request.GoogleReaderTokenKey, sess.ID)
	return ctx
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

	log := logging.FromContext(ctx).With(
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))

	token, err := authToken(r)
	if err != nil {
		log.Warn("[GoogleReader] authentication failed",
			slog.Bool("authentication_failed", true),
			slog.Any("error", err))
		sendUnauthorizedResponse(w)
		return
	} else if token == "" {
		self.next.ServeHTTP(w, r)
		return
	}

	user, sess, err := self.store.UserSession(ctx, token)
	if err != nil {
		log.Warn(
			"[GoogleReader] No user found with the given Google Reader username",
			slog.Bool("authentication_failed", true),
			slog.Any("error", err),
		)
		sendUnauthorizedResponse(w)
		return
	}

	if sess == nil {
		log.Warn(
			"[GoogleReader] No session found with the given Google Reader credentials",
			slog.Bool("authentication_failed", true))
		sendUnauthorizedResponse(w)
		return
	}
	middleware.AccessLogUser(ctx, user)

	if !user.Integration().GoogleReaderEnabled {
		log.Warn(
			"[GoogleReader] No user found with the given Google Reader credentials",
			slog.Bool("authentication_failed", true))
		sendUnauthorizedResponse(w)
		return
	}

	self.next.ServeHTTP(w, r.WithContext(
		request.WithUserSession(ctx, user, sess)))

	if d := time.Since(sess.UpdatedAt); d > 5*time.Minute {
		err := self.store.RefreshAppSession(ctx, sess)
		if err != nil {
			log.Error("[GoogleReader] Unable update session updated timestamp",
				slog.String("id", sess.ID),
				slog.Duration("last_updated_ago", d),
				slog.Any("error", err))
		}
	}
}

func authToken(r *http.Request) (string, error) {
	authorization := r.Header.Get("Authorization")
	if authorization == "" {
		return "", nil
	}

	googleLogin, ok := strings.CutPrefix(authorization, "GoogleLogin ")
	if !ok {
		return "", errors.New(
			"authorization header doesn't begin with GoogleLogin")
	}

	authKey, token, ok := strings.Cut(strings.TrimSpace(googleLogin), "=")
	if !ok || authKey != "auth" {
		return "", errors.New(
			"authorization header doesn't have the expected GoogleLogin format auth=xxxxxx")
	}
	return token, nil
}
