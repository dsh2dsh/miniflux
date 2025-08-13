// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package html // import "miniflux.app/v2/internal/http/response/html"

import (
	"context"
	"errors"
	"html"
	"log/slog"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/logging"
)

const (
	cacheControl = "Cache-Control"
	cacheNoCache = "no-cache, max-age=0, must-revalidate, no-store"

	contentType = "Content-Type"
	textHTML    = "text/html; charset=utf-8"
	textPlain   = "text/plain; charset=utf-8"

	contentSecPol = "Content-Security-Policy"
)

// OK creates a new HTML response with a 200 status code.
func OK(w http.ResponseWriter, r *http.Request, body any) {
	response.New(w, r).
		WithHeader(contentType, textHTML).
		WithHeader(cacheControl, cacheNoCache).
		WithBody(body).
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

	response.New(w, r).
		WithStatus(statusCode).
		WithHeader(contentSecPol,
			response.ContentSecurityPolicyForUntrustedContent).
		WithHeader(contentType, textPlain).
		WithHeader(cacheControl, cacheNoCache).
		WithBody(html.EscapeString(err.Error())).
		Write()
}

// BadRequest sends a bad request error to the client.
func BadRequest(w http.ResponseWriter, r *http.Request, err error) {
	statusCode := http.StatusBadRequest
	logStatusCode(r, statusCode, err)
	response.New(w, r).
		WithStatus(statusCode).
		WithHeader(contentSecPol,
			response.ContentSecurityPolicyForUntrustedContent).
		WithHeader(contentType, textPlain).
		WithHeader(cacheControl, cacheNoCache).
		WithBody(html.EscapeString(err.Error())).
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
	response.New(w, r).
		WithStatus(statusCode).
		WithHeader(contentType, textHTML).
		WithHeader(cacheControl, cacheNoCache).
		WithBody("Access Forbidden").
		Write()
}

// NotFound sends a page not found error to the client.
func NotFound(w http.ResponseWriter, r *http.Request) {
	statusCode := http.StatusNotFound
	logStatusCode(r, statusCode, nil)
	response.New(w, r).
		WithStatus(statusCode).
		WithHeader(contentType, textHTML).
		WithHeader(cacheControl, cacheNoCache).
		WithBody("Page Not Found").
		Write()
}

// Redirect redirects the user to another location.
func Redirect(w http.ResponseWriter, r *http.Request, uri string) {
	http.Redirect(w, r, uri, http.StatusFound)
}

// RequestedRangeNotSatisfiable sends a range not satisfiable error to the client.
func RequestedRangeNotSatisfiable(w http.ResponseWriter, r *http.Request,
	contentRange string,
) {
	statusCode := http.StatusRequestedRangeNotSatisfiable
	logStatusCode(r, statusCode, nil)
	response.New(w, r).
		WithStatus(http.StatusRequestedRangeNotSatisfiable).
		WithHeader(contentType, textHTML).
		WithHeader(cacheControl, cacheNoCache).
		WithHeader("Content-Range", contentRange).
		WithBody("Range Not Satisfiable").
		Write()
}
