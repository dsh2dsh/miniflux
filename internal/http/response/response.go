// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package response // import "miniflux.app/v2/internal/http/response"

import (
	"net/http"
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
	New(w, r, WithStatusAccepted()).Write()
}

// BadRequest sends a bad request error to the client.
func BadRequest(w http.ResponseWriter, r *http.Request, err error) {
	WrapBadRequest(err).Serve(w, r)
}

// Forbidden sends a forbidden error to the client.
func Forbidden(w http.ResponseWriter, r *http.Request) {
	ErrForbidden.Serve(w, r)
}

func NoContent(w http.ResponseWriter, r *http.Request) {
	New(w, r, WithStatusNoContent()).Write()
}

// NotFound sends a page not found error to the client.
func NotFound(w http.ResponseWriter, r *http.Request) {
	ErrNotFound.Serve(w, r)
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
	NewError(http.StatusRequestedRangeNotSatisfiable).Serve(w, r,
		WithHeader("Content-Range", contentRange))
}

// ServerError sends an internal error to the client.
func ServerError(w http.ResponseWriter, r *http.Request, err error) {
	WrapServerError(err).Serve(w, r)
}

// Text writes a standard text response with a status 200 OK.
func Text(w http.ResponseWriter, r *http.Request, body string) {
	New(w, r).
		WithHeader(contentType, textPlain).
		WithBodyAsString(body).
		Write()
}
