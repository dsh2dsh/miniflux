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
	UserLanguageContextKey
	UserThemeContextKey
	SessionIDContextKey
	CSRFContextKey
	OAuth2StateContextKey
	OAuth2CodeVerifierContextKey
	FlashMessageContextKey
	FlashErrorMessageContextKey
	PocketRequestTokenContextKey
	LastForceRefreshContextKey
	ClientIPContextKey
	GoogleReaderToken
	WebAuthnDataContextKey
)

var (
	publicKey  ctxPublic  = struct{}{}
	sessionKey ctxSession = struct{}{}
	userKey    ctxUser    = struct{}{}
)

func WebAuthnSessionData(r *http.Request) *model.WebAuthnSession {
	return getContextValue[*model.WebAuthnSession](r, WebAuthnDataContextKey)
}

// GoolgeReaderToken returns the google reader token if it exists.
func GoolgeReaderToken(r *http.Request) string {
	return getContextStringValue(r, GoogleReaderToken)
}

// IsAdminUser checks if the logged user is administrator.
func IsAdminUser(r *http.Request) bool {
	return getContextBoolValue(r, IsAdminUserContextKey)
}

// IsAuthenticated returns a boolean if the user is authenticated.
func IsAuthenticated(r *http.Request) bool {
	return getContextBoolValue(r, IsAuthenticatedContextKey)
}

// UserID returns the UserID of the logged user.
func UserID(r *http.Request) int64 {
	return getContextInt64Value(r, UserIDContextKey)
}

// UserName returns the username of the logged user.
func UserName(r *http.Request) string {
	value := getContextStringValue(r, UserNameContextKey)
	if value == "" {
		value = "unknown"
	}
	return value
}

// UserTimezone returns the timezone used by the logged user.
func UserTimezone(r *http.Request) string {
	value := getContextStringValue(r, UserTimezoneContextKey)
	if value == "" {
		value = "UTC"
	}
	return value
}

// UserLanguage get the locale used by the current logged user.
func UserLanguage(r *http.Request) string {
	language := getContextStringValue(r, UserLanguageContextKey)
	if language == "" {
		language = "en_US"
	}
	return language
}

// UserTheme get the theme used by the current logged user.
func UserTheme(r *http.Request) string {
	theme := getContextStringValue(r, UserThemeContextKey)
	if theme == "" {
		theme = "system_serif"
	}
	return theme
}

func WithCSRF(ctx context.Context, csrf string) context.Context {
	return context.WithValue(ctx, CSRFContextKey, csrf)
}

// CSRF returns the current CSRF token.
func CSRF(r *http.Request) string {
	return getContextStringValue(r, CSRFContextKey)
}

// SessionID returns the current session ID.
func SessionID(r *http.Request) string {
	return getContextStringValue(r, SessionIDContextKey)
}

// OAuth2State returns the current OAuth2 state.
func OAuth2State(r *http.Request) string {
	return getContextStringValue(r, OAuth2StateContextKey)
}

func OAuth2CodeVerifier(r *http.Request) string {
	return getContextStringValue(r, OAuth2CodeVerifierContextKey)
}

// FlashMessage returns the message message if any.
func FlashMessage(r *http.Request) string {
	return getContextStringValue(r, FlashMessageContextKey)
}

// FlashErrorMessage returns the message error message if any.
func FlashErrorMessage(r *http.Request) string {
	return getContextStringValue(r, FlashErrorMessageContextKey)
}

// PocketRequestToken returns the Pocket Request Token if any.
func PocketRequestToken(r *http.Request) string {
	return getContextStringValue(r, PocketRequestTokenContextKey)
}

// LastForceRefresh returns the last force refresh timestamp.
func LastForceRefresh(r *http.Request) int64 {
	return getContextInt64Value(r, LastForceRefreshContextKey)
}

func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, ClientIPContextKey, ip)
}

// ClientIP returns the client IP address stored in the context.
func ClientIP(r *http.Request) string {
	return getContextValue[string](r, ClientIPContextKey)
}

func getContextValue[T any](r *http.Request, key ContextKey) (zero T) {
	if v := r.Context().Value(key); v != nil {
		if value, ok := v.(T); ok {
			return value
		}
	}
	return
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
