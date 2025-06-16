// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/http/cookie"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/storage"
)

const (
	csrfForm   = "csrf"
	csrfHeader = "X-Csrf-Token"
)

var publicSession = model.Session{
	Data: &model.SessionData{
		Language: "en_US",
		Theme:    "light_sans_serif",
	},
}

type middleware struct {
	router *mux.Router
	store  *storage.Storage
}

func newMiddleware(router *mux.Router, store *storage.Storage) *middleware {
	return &middleware{router, store}
}

func (m *middleware) handleUserSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if request.Public(r) {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		log := logging.FromContext(ctx).With(
			slog.String("url", r.URL.String()))

		user := request.User(r)
		sess := request.Session(r)

		if user == nil || sess == nil {
			log.Debug(
				"Redirecting to login page because no user session has been found")
			html.Redirect(w, r, route.Path(m.router, "login"))
			return
		}

		log.Debug("User session found",
			slog.Group("user",
				slog.Int64("id", user.ID),
				slog.String("name", user.Username)),
			slog.Group("session", slog.String("id", sess.ID)))

		ctx = context.WithValue(ctx, request.UserIDContextKey, user.ID)
		ctx = context.WithValue(ctx, request.IsAuthenticatedContextKey, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *middleware) handleAppSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := logging.FromContext(ctx).With(
			slog.String("url", r.URL.String()))
		s := request.Session(r)

		if s == nil {
			if request.Public(r) {
				ctx = contextWithSessionKeys(ctx, &publicSession)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			err := errors.New("no app session has been found")
			log.Warn(err.Error())
			html.BadRequest(w, r, err)
			return
		}

		ctx = contextWithSessionKeys(ctx, s)
		r = r.WithContext(ctx)
		if r.Method == http.MethodPost && !checkCSRF(w, r, s.Data.CSRF) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func contextWithSessionKeys(ctx context.Context, sess *model.Session,
) context.Context {
	ctx = context.WithValue(ctx, request.SessionIDContextKey, sess.ID)
	ctx = request.WithCSRF(ctx, sess.Data.CSRF)
	ctx = context.WithValue(ctx, request.OAuth2StateContextKey,
		sess.Data.OAuth2State)
	ctx = context.WithValue(ctx, request.OAuth2CodeVerifierContextKey,
		sess.Data.OAuth2CodeVerifier)
	ctx = context.WithValue(ctx, request.FlashMessageContextKey,
		sess.Data.FlashMessage)
	ctx = context.WithValue(ctx, request.FlashErrorMessageContextKey,
		sess.Data.FlashErrorMessage)
	ctx = context.WithValue(ctx, request.UserLanguageContextKey,
		sess.Data.Language)
	ctx = context.WithValue(ctx, request.UserThemeContextKey,
		sess.Data.Theme)
	ctx = context.WithValue(ctx, request.LastForceRefreshContextKey,
		sess.Data.LastForceRefresh)
	ctx = context.WithValue(ctx, request.WebAuthnDataContextKey,
		sess.Data.WebAuthnSessionData)
	return ctx
}

func checkCSRF(w http.ResponseWriter, r *http.Request, csrf string) bool {
	formToken := r.FormValue(csrfForm)
	headerToken := r.Header.Get(csrfHeader)

	ok := crypto.ConstantTimeCmp(csrf, formToken) ||
		crypto.ConstantTimeCmp(csrf, headerToken)
	if csrf != "" && ok {
		return true
	}

	err := errors.New("invalid or missing CSRF token")
	logging.FromContext(r.Context()).Warn(err.Error(),
		slog.String("csrf", csrf),
		slog.String("form", formToken),
		slog.String("header", headerToken))
	html.BadRequest(w, r, err)
	return false
}

func (m *middleware) handleAuthProxy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if request.IsAuthenticated(r) || config.Opts.AuthProxyHeader() == "" {
			next.ServeHTTP(w, r)
			return
		}

		username := r.Header.Get(config.Opts.AuthProxyHeader())
		if username == "" {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		clientIP := request.ClientIP(r)
		log := logging.FromContext(ctx).With(
			slog.String("client_ip", clientIP),
			slog.String("user_agent", r.UserAgent()),
			slog.String("username", username))
		log.Debug("[AuthProxy] Received authenticated request")

		user, err := m.store.UserByUsername(ctx, username)
		if err != nil {
			html.ServerError(w, r, err)
			return
		}

		if user == nil {
			if !config.Opts.IsAuthProxyUserCreationAllowed() {
				log.Debug(
					"[AuthProxy] User doesn't exist and user creation is not allowed")
				html.Forbidden(w, r)
				return
			}

			user, err = m.store.CreateUser(ctx, &model.UserCreationRequest{
				Username: username,
			})
			if err != nil {
				html.ServerError(w, r, err)
				return
			}
		}

		sess, err := m.store.CreateAppSessionForUser(r.Context(), user,
			r.UserAgent(), clientIP)
		if err != nil {
			html.ServerError(w, r, err)
			return
		}

		log.Info("[AuthProxy] User authenticated successfully",
			slog.Int64("user_id", user.ID),
			slog.String("session_id", sess.ID))

		if err := m.store.SetLastLogin(r.Context(), user.ID); err != nil {
			html.ServerError(w, r, err)
			return
		}

		http.SetCookie(w, cookie.NewSession(sess.ID))
		html.Redirect(w, r, route.Path(m.router, user.DefaultHomePage))
	})
}

func (m *middleware) PublicCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !request.Public(r) {
			next.ServeHTTP(w, r)
			return
		}

		if r.Method != http.MethodPost {
			csrf := crypto.GenerateRandomString(64)
			http.SetCookie(w, cookie.NewCSRF(csrf))
			ctx := request.WithCSRF(r.Context(), csrf)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		csrf := request.CookieValue(r, cookie.CookieCSRF)
		if !checkCSRF(w, r, csrf) {
			return
		}
		http.SetCookie(w, cookie.ExpiredCSRF())
		next.ServeHTTP(w, r)
	})
}
