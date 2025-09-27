// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/cookie"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/securecookie"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/ui/view"
)

func (h *handler) checkLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clientIP := request.ClientIP(r)
	log := logging.FromContext(ctx).With(
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()))
	v := view.New(h.tpl, r, nil)

	if config.Opts.DisableLocalAuth() {
		log.Warn("blocking local auth login attempt, local auth is disabled")
		html.OK(w, r, v.Render("login"))
		return
	}

	f := form.NewAuthForm(r)
	v.Set("errorMessage",
		locale.NewLocalizedError("error.bad_credentials").
			Translate(request.UserLanguage(r)))
	v.Set("form", f)
	log = log.With(slog.String("username", f.Username))

	if lerr := f.Validate(); lerr != nil {
		translatedErrorMessage := lerr.Translate(request.UserLanguage(r))
		log.Warn("Validation error during login check",
			slog.Any("error", translatedErrorMessage))
		html.OK(w, r, v.Render("login"))
		return
	}

	err := h.store.CheckPassword(ctx, f.Username, f.Password)
	if err != nil {
		log.Warn("Incorrect username or password", slog.Any("error", err))
		html.OK(w, r, v.Render("login"))
		return
	}

	user, err := h.store.UserByUsername(ctx, f.Username)
	if err != nil {
		html.ServerError(w, r, err)
		return
	} else if user == nil {
		log.Warn("User not found")
		html.OK(w, r, v.Render("login"))
		return
	}
	log.Info("User authenticated successfully with username/password",
		slog.Int64("user_id", user.ID))

	sess, err := h.store.CreateAppSessionForUser(ctx, user, r.UserAgent(),
		clientIP)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	if err := h.store.SetLastLogin(ctx, user.ID); err != nil {
		html.ServerError(w, r, err)
		return
	}

	http.SetCookie(w, cookie.NewSession(sess.ID))
	h.redirectHome(w, r, user)
}

func (h *handler) redirectHome(w http.ResponseWriter, r *http.Request,
	user *model.User,
) {
	log := logging.FromContext(r.Context())

	redirect, err := redirectFromCookie(w, r, h.secureCookie)
	if err != nil {
		log.Error("Unable redirect back to original page", slog.Any("error", err))
		h.redirect(w, r, user.DefaultHomePage)
		return
	}

	if redirect != "" {
		log.Debug("Redirect back to original page", slog.String("uri", redirect))
		html.Redirect(w, r, redirect)
		return
	}

	log.Debug("Redirect to user default home page")
	h.redirect(w, r, user.DefaultHomePage)
}

type redirectCookie struct {
	Redirect string `json:"redirect"`
}

func setLoginRedirect(w http.ResponseWriter,
	secureCookie *securecookie.SecureCookie, redirect string,
) error {
	data := redirectCookie{Redirect: redirect}
	b, err := json.Marshal(&data)
	if err != nil {
		return fmt.Errorf("ui: marshal login redirect to cookie: %w", err)
	}

	encrypted, err := secureCookie.EncryptCookie(b)
	if err != nil {
		return fmt.Errorf("ui: encrypt login redirect cookie: %w", err)
	}

	http.SetCookie(w, cookie.NewSessionData(encrypted))
	return nil
}

func redirectFromCookie(w http.ResponseWriter, r *http.Request,
	secureCookie *securecookie.SecureCookie,
) (string, error) {
	plaintext := request.CookieValue(r, cookie.CookieSessionData)
	if plaintext == "" {
		return "", nil
	}
	http.SetCookie(w, cookie.ExpiredSessionData())

	b, err := secureCookie.DecryptCookie(plaintext)
	if err != nil {
		return "", fmt.Errorf("ui: decrypt login redirect cookie: %w", err)
	}

	var data redirectCookie
	if err := json.Unmarshal(b, &data); err != nil {
		return "", fmt.Errorf("ui: unmarshal login redirect cookie: %w", err)
	}

	http.SetCookie(w, cookie.ExpiredSessionData())
	return data.Redirect, nil
}
