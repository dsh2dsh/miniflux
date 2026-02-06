// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package request // import "miniflux.app/v2/internal/http/request"

import (
	"context"
	"net/http"

	"miniflux.app/v2/internal/model"
)

type (
	// ContextKey represents a context key.
	ContextKey int

	ctxPublic  struct{}
	ctxSession struct{}
	ctxUser    struct{}
)

// List of context keys.
const (
	UserIDContextKey ContextKey = iota
	UserNameContextKey
	UserTimezoneContextKey
	IsAdminUserContextKey
	IsAuthenticatedContextKey
	ClientIPContextKey
	GoogleReaderTokenKey
)

var (
	publicKey  ctxPublic  = struct{}{}
	sessionKey ctxSession = struct{}{}
	userKey    ctxUser    = struct{}{}
)

// WebAuthnSessionData returns WebAuthn session data from the request context, or nil if absent.
func WebAuthnSessionData(r *http.Request) *model.WebAuthnSession {
	if s := Session(r); s != nil {
		return &s.Data.WebAuthnSessionData
	}
	return nil
}

// GoogleReaderToken returns the Google Reader token from the request context, if present.
func GoogleReaderToken(r *http.Request) string {
	return getContextStringValue(r, GoogleReaderTokenKey)
}

// IsAdminUser reports whether the logged-in user is an administrator.
func IsAdminUser(r *http.Request) bool {
	return getContextBoolValue(r, IsAdminUserContextKey)
}

// IsAuthenticated reports whether the user is authenticated.
func IsAuthenticated(r *http.Request) bool {
	return getContextBoolValue(r, IsAuthenticatedContextKey)
}

// UserID returns the logged-in user's ID from the request context.
func UserID(r *http.Request) int64 {
	return getContextInt64Value(r, UserIDContextKey)
}

// UserName returns the logged-in user's username, or "unknown" when unset.
func UserName(r *http.Request) string {
	value := getContextStringValue(r, UserNameContextKey)
	if value == "" {
		value = "unknown"
	}
	return value
}

// UserTimezone returns the user's timezone, defaulting to "UTC" when unset.
func UserTimezone(r *http.Request) string {
	value := getContextStringValue(r, UserTimezoneContextKey)
	if value == "" {
		value = "UTC"
	}
	return value
}

// UserLanguage returns the user's locale, defaulting to "en_US" when unset.
func UserLanguage(r *http.Request) string {
	if s := Session(r); s != nil {
		return s.Data.Language
	}
	return "en_US"
}

// UserTheme returns the user's theme, defaulting to "system_serif" when unset.
func UserTheme(r *http.Request) string {
	if s := Session(r); s != nil {
		return s.Data.Theme
	}
	return "system_serif"
}

// SessionID returns the current session ID from the request context.
func SessionID(r *http.Request) string {
	if s := Session(r); s != nil {
		return s.ID
	}
	return ""
}

// OAuth2State returns the OAuth2 state value from the request context.
func OAuth2State(r *http.Request) string {
	if s := Session(r); s != nil {
		return s.Data.OAuth2State
	}
	return ""
}

// OAuth2CodeVerifier returns the OAuth2 PKCE code verifier from the request context.
func OAuth2CodeVerifier(r *http.Request) string {
	if s := Session(r); s != nil {
		return s.Data.OAuth2CodeVerifier
	}
	return ""
}

// FlashMessage returns the flash message from the request context, if any.
func FlashMessage(r *http.Request) string {
	if s := Session(r); s != nil {
		return s.Data.FlashMessage
	}
	return ""
}

// FlashErrorMessage returns the flash error message from the request context, if any.
func FlashErrorMessage(r *http.Request) string {
	if s := Session(r); s != nil {
		return s.Data.FlashErrorMessage
	}
	return ""
}

// LastForceRefresh returns the last force refresh timestamp from the request context.
func LastForceRefresh(r *http.Request) int64 {
	if s := Session(r); s != nil {
		return s.Data.LastForceRefresh
	}
	return 0
}

func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, ClientIPContextKey, ip)
}

// ClientIP returns the client IP address stored in the request context.
func ClientIP(r *http.Request) string {
	return getContextValue[string](r, ClientIPContextKey)
}

func getContextValue[T any](r *http.Request, key ContextKey) (zero T) {
	if v := r.Context().Value(key); v != nil {
		if value, ok := v.(T); ok {
			return value
		}
	}
	return zero
}

func getContextStringValue(r *http.Request, key ContextKey) string {
	return getContextValue[string](r, key)
}

func getContextBoolValue(r *http.Request, key ContextKey) bool {
	return getContextValue[bool](r, key)
}

func getContextInt64Value(r *http.Request, key ContextKey) int64 {
	return getContextValue[int64](r, key)
}

func WithUserSession(ctx context.Context, user *model.User, s *model.Session,
) context.Context {
	return WithSession(WithUser(ctx, user), s)
}

func WithUser(ctx context.Context, user *model.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

func User(r *http.Request) *model.User {
	if u, ok := r.Context().Value(userKey).(*model.User); ok {
		return u
	}
	return nil
}

func WithSession(ctx context.Context, s *model.Session) context.Context {
	return context.WithValue(ctx, sessionKey, s)
}

func Session(r *http.Request) *model.Session {
	if s, ok := r.Context().Value(sessionKey).(*model.Session); ok {
		return s
	}
	return nil
}

func WithPublic(ctx context.Context) context.Context {
	return context.WithValue(ctx, publicKey, struct{}{})
}

func Public(r *http.Request) bool {
	return r.Context().Value(publicKey) != nil
}
