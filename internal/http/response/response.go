// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package response // import "miniflux.app/v2/internal/http/response"

import (
	"context"
	"errors"
	"html"
	"log/slog"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/logging"
)

const (
	// ContentSecurityPolicyForUntrustedContent is the default CSP for untrusted content.
	// default-src 'none' disables all content sources
	// form-action 'none' disables all form submissions
	// sandbox enables a sandbox for the requested resource
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy/form-action
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy/sandbox
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy/default-src
	ContentSecurityPolicyForUntrustedContent = `default-src 'none'; form-action 'none'; sandbox;`

	cacheControl  = "Cache-Control"
	cacheNoCache  = "no-cache, max-age=0, must-revalidate, no-store"
	contentSecPol = "Content-Security-Policy"
	contentType   = "Content-Type"
	textPlain     = "text/plain; charset=utf-8"
)

func Accepted(w http.ResponseWriter, r *http.Request) {
	New(w, r).WithStatus(http.StatusAccepted).Write()
}

// BadRequest sends a bad request error to the client.
func BadRequest(w http.ResponseWriter, r *http.Request, err error) {
	statusCode := http.StatusBadRequest
	logStatusCode(r, statusCode, err)
	New(w, r).
		WithStatus(statusCode).
		WithHeader(contentSecPol, ContentSecurityPolicyForUntrustedContent).
		WithHeader(contentType, textPlain).
		WithHeader(cacheControl, cacheNoCache).
		WithBodyAsString(html.EscapeString(err.Error())).
		Write()
}

func logStatusCode(r *http.Request, statusCode int, err error) {
	log := logging.FromContext(r.Context())
	if err != nil {
		log = log.With(slog.Any("error", err))
	}
	log.Warn(http.StatusText(statusCode),
		slog.String("client_ip", request.ClientIP(r)),
		slog.GroupAttrs("request",
			slog.String("method", r.Method),
			slog.String("uri", r.RequestURI),
			slog.String("user_agent", r.UserAgent())),
		slog.GroupAttrs("response",
			slog.Int("status_code", statusCode)))
}

// Forbidden sends a forbidden error to the client.
func Forbidden(w http.ResponseWriter, r *http.Request) {
	statusCode := http.StatusForbidden
	logStatusCode(r, statusCode, nil)
	New(w, r).
		WithStatus(statusCode).
		WithHeader(contentType, textPlain).
		WithHeader(cacheControl, cacheNoCache).
		WithBodyAsString("Access Forbidden").
		Write()
}

func NoContent(w http.ResponseWriter, r *http.Request) {
	New(w, r).WithStatus(http.StatusNoContent).Write()
}

// NotFound sends a page not found error to the client.
func NotFound(w http.ResponseWriter, r *http.Request) {
	statusCode := http.StatusNotFound
	logStatusCode(r, statusCode, nil)
	New(w, r).
		WithStatus(statusCode).
		WithHeader(contentType, textPlain).
		WithHeader(cacheControl, cacheNoCache).
		WithBodyAsString("Page Not Found").
		Write()
}

// Redirect redirects the user to another location.
func Redirect(w http.ResponseWriter, r *http.Request, uri string) {
	http.Redirect(w, r, uri, http.StatusFound)
}

// RequestedRangeNotSatisfiable sends a range not satisfiable error to the
// client.
func RequestedRangeNotSatisfiable(w http.ResponseWriter, r *http.Request,
	contentRange string,
) {
	statusCode := http.StatusRequestedRangeNotSatisfiable
	logStatusCode(r, statusCode, nil)
	New(w, r).
		WithStatus(http.StatusRequestedRangeNotSatisfiable).
		WithHeader(contentType, textPlain).
		WithHeader(cacheControl, cacheNoCache).
		WithHeader("Content-Range", contentRange).
		WithBodyAsString("Range Not Satisfiable").
		Write()
}

// ServerError sends an internal error to the client.
func ServerError(w http.ResponseWriter, r *http.Request, err error) {
	log := logging.FromContext(r.Context()).With(
		slog.Any("error", err),
		slog.String("client_ip", request.ClientIP(r)),
		slog.GroupAttrs("request",
			slog.String("method", r.Method),
			slog.String("uri", r.RequestURI),
			slog.String("user_agent", r.UserAgent())))

	clientClosed := errors.Is(err, context.Canceled) &&
		errors.Is(r.Context().Err(), context.Canceled)
	if clientClosed {
		statusCode := 499
		log.Debug("client closed request",
			slog.GroupAttrs("response", slog.Int("status_code", statusCode)))
		http.Error(w, err.Error(), statusCode)
		return
	}

	statusCode := http.StatusInternalServerError
	log.Error(http.StatusText(statusCode),
		slog.GroupAttrs("response",
			slog.Int("status_code", http.StatusInternalServerError)))

	New(w, r).
		WithStatus(statusCode).
		WithHeader(contentSecPol, ContentSecurityPolicyForUntrustedContent).
		WithHeader(contentType, textPlain).
		WithHeader(cacheControl, cacheNoCache).
		WithBodyAsString(html.EscapeString(err.Error())).
		Write()
}
