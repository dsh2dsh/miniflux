// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package fetcher // import "miniflux.app/v2/internal/reader/fetcher"

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"miniflux.app/v2/internal/locale"
)

type ResponseHandler struct {
	httpResponse *http.Response
	clientErr    error

	maxBodySize int64
}

func (r *ResponseHandler) Status() string  { return r.httpResponse.Status }
func (r *ResponseHandler) StatusCode() int { return r.httpResponse.StatusCode }

func (r *ResponseHandler) Header(key string) string {
	return r.httpResponse.Header.Get(key)
}

func (r *ResponseHandler) Err() error { return r.clientErr }

func (r *ResponseHandler) URL() *url.URL { return r.httpResponse.Request.URL }

func (r *ResponseHandler) EffectiveURL() string { return r.URL().String() }

func (r *ResponseHandler) ContentType() string {
	return r.httpResponse.Header.Get("Content-Type")
}

func (r *ResponseHandler) LastModified() string {
	// Ignore caching headers for feeds that do not want any cache.
	if r.httpResponse.Header.Get("Expires") == "0" {
		return ""
	}
	return r.httpResponse.Header.Get("Last-Modified")
}

func (r *ResponseHandler) ETag() string {
	// Ignore caching headers for feeds that do not want any cache.
	if r.httpResponse.Header.Get("Expires") == "0" {
		return ""
	}
	return r.httpResponse.Header.Get("ETag")
}

func (r *ResponseHandler) ExpiresInMinutes() int {
	expiresHeaderValue := r.httpResponse.Header.Get("Expires")
	if expiresHeaderValue != "" {
		t, err := time.Parse(time.RFC1123, expiresHeaderValue)
		if err == nil {
			return int(math.Ceil(time.Until(t).Minutes()))
		}
	}
	return 0
}

func (r *ResponseHandler) CacheControlMaxAgeInMinutes() int {
	cacheControlHeaderValue := r.httpResponse.Header.Get("Cache-Control")
	if cacheControlHeaderValue != "" {
		for directive := range strings.SplitSeq(cacheControlHeaderValue, ",") {
			directive = strings.TrimSpace(directive)
			if s, ok := strings.CutPrefix(directive, "max-age="); ok {
				maxAge, err := strconv.Atoi(s)
				if err == nil {
					return int(math.Ceil(float64(maxAge) / 60))
				}
			}
		}
	}
	return 0
}

func (r *ResponseHandler) parseRetryDelay() time.Duration {
	retryAfterHeaderValue := r.httpResponse.Header.Get("Retry-After")
	if retryAfterHeaderValue == "" {
		return 0
	}

	// First, try to parse as an integer (number of seconds)
	if seconds, err := strconv.Atoi(retryAfterHeaderValue); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// If not an integer, try to parse as an HTTP-date
	if t, err := time.Parse(time.RFC1123, retryAfterHeaderValue); err == nil {
		return time.Until(t)
	}
	return 0
}

func (r *ResponseHandler) rateLimited() bool {
	return r.httpResponse != nil &&
		r.httpResponse.StatusCode == http.StatusTooManyRequests
}

func (r *ResponseHandler) IsModified(lastEtagValue, lastModifiedValue string) bool {
	if r.httpResponse.StatusCode == http.StatusNotModified {
		return false
	}

	if r.ETag() != "" {
		return r.ETag() != lastEtagValue
	}

	if r.LastModified() != "" {
		return r.LastModified() != lastModifiedValue
	}

	return true
}

func (r *ResponseHandler) IsRedirect() bool {
	return r.httpResponse != nil &&
		(r.httpResponse.StatusCode == http.StatusMovedPermanently ||
			r.httpResponse.StatusCode == http.StatusFound ||
			r.httpResponse.StatusCode == http.StatusSeeOther ||
			r.httpResponse.StatusCode == http.StatusTemporaryRedirect ||
			r.httpResponse.StatusCode == http.StatusPermanentRedirect)
}

func (r *ResponseHandler) Close() {
	if r.Err() != nil {
		return
	}
	BodyClose(r.httpResponse.Body)
}

// maxPostHandlerReadBytes is the max number of Request.Body bytes not
// consumed by a handler that the server will read from the client
// in order to keep a connection alive. If there are more bytes
// than this, the server, to be paranoid, instead sends a
// "Connection close" response.
//
// This number is approximately what a typical machine's TCP buffer
// size is anyway.  (if we have the bytes on the machine, we might as
// well read them)
//
// See: net/http/server.go
const maxPostHandlerReadBytes = 256 << 10

// https://github.com/golang/go/issues/60240
func BodyClose(r io.ReadCloser) {
	_, _ = io.CopyN(io.Discard, r, maxPostHandlerReadBytes+1)
	r.Close()
}

func (r *ResponseHandler) getReader(maxBodySize int64) io.ReadCloser {
	slog.Debug("Request response",
		slog.String("effective_url", r.EffectiveURL()),
		slog.String("content_length", r.httpResponse.Header.Get("Content-Length")),
		slog.String("content_encoding",
			r.httpResponse.Header.Get("Content-Encoding")),
		slog.String("content_type", r.httpResponse.Header.Get("Content-Type")))
	return http.MaxBytesReader(nil, r.httpResponse.Body, maxBodySize)
}

func (r *ResponseHandler) Body() io.ReadCloser {
	return r.getReader(r.maxBodySize)
}

func (r *ResponseHandler) ReadBody() ([]byte, *locale.LocalizedErrorWrapper) {
	var buffer bytes.Buffer
	_, err := io.Copy(&buffer, r.Body())
	if err != nil && !errors.Is(err, io.EOF) {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return nil, locale.NewLocalizedErrorWrapper(
				fmt.Errorf("fetcher: response body too large: %d bytes",
					maxBytesErr.Limit),
				"error.http_response_too_large")
		}
		return nil, locale.NewLocalizedErrorWrapper(
			fmt.Errorf("fetcher: unable to read response body: %w", err),
			"error.http_body_read", err)
	}

	if buffer.Len() == 0 {
		return nil, locale.NewLocalizedErrorWrapper(
			errors.New("fetcher: empty response body"),
			"error.http_empty_response_body")
	}
	return buffer.Bytes(), nil
}
