// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cookie // import "miniflux.app/v2/internal/http/cookie"

import (
	"net/http"
	"time"

	"miniflux.app/v2/internal/config"
)

// Cookie names.
const (
	CookieAppSessionID = "MinifluxAppSessionID"
	CookieCSRF         = "MinifluxCSRF"
	CookieSessionData  = "MinifluxSession"
)

func NewCSRF(v string) *http.Cookie { return makeSessionCookie(CookieCSRF, v) }

func NewSession(id string) *http.Cookie { return New(CookieAppSessionID, id) }

func NewSessionData(v string) *http.Cookie {
	return makeSessionCookie(CookieSessionData, v)
}

// New creates a new cookie.
func New(name, value string) *http.Cookie {
	c := makeSessionCookie(name, value)
	c.Expires = time.Now().Add(
		time.Duration(config.Opts.CleanupRemoveSessionsDays()) * 24 * time.Hour)
	return c
}

func makeSessionCookie(name, value string) *http.Cookie {
	path := config.Opts.BasePath()
	if path == "" {
		path = "/"
	}

	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Secure:   config.Opts.HTTPS(),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func ExpiredCSRF() *http.Cookie        { return Expired(CookieCSRF) }
func ExpiredSession() *http.Cookie     { return Expired(CookieAppSessionID) }
func ExpiredSessionData() *http.Cookie { return Expired(CookieSessionData) }

// Expired returns an expired cookie.
func Expired(name string) *http.Cookie {
	c := makeSessionCookie(name, "")
	c.MaxAge = -1
	c.Expires = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	return c
}
