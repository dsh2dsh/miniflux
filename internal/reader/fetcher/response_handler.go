// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package fetcher // import "miniflux.app/v2/internal/reader/fetcher"

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/reader/encoding"
	"miniflux.app/v2/internal/reader/sanitizer"
)

type ResponseHandler struct {
	httpResponse *http.Response
	clientErr    error
	maxBodySize  int64

	content []byte
}

func NewResponseHandler(resp *http.Response, err error) *ResponseHandler {
	r := &ResponseHandler{
		httpResponse: resp,
		clientErr:    err,
		maxBodySize:  config.HTTPClientMaxBodySize(),
	}
	return r.withResponseContent()
}

func (self *ResponseHandler) withResponseContent() *ResponseHandler {
	skip := self.Err() != nil || self.goodStatusCode() ||
		self.httpResponse.ContentLength == 0
	if skip {
		return self
	}

	body, err := encoding.NewCharsetReader(self.Body(), self.ContentType())
	if err != nil {
		return self
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, body); err != nil {
		return self
	}
	self.content = b.Bytes()

	self.logResponseContent()
	return self
}

func (self *ResponseHandler) goodStatusCode() bool {
	return self.StatusCode() < 400
}

func (self *ResponseHandler) logResponseContent() {
	if len(self.content) == 0 {
		return
	}

	content := sanitizer.StripTags(string(self.content))
	if content == "" {
		return
	}

	r := strings.NewReader(content)
	s := bufio.NewScanner(r)
	limit := 1024
	log := logging.FromContext(self.Context())

	for s.Scan() {
		line := bytes.TrimSpace(s.Bytes())
		switch n := len(line); {
		case n == 0:
			continue
		case n > limit:
			log.Warn("limit reached, left of response content skipped",
				slog.Int("length", len(s.Bytes())+r.Len()),
				slog.Int("limit", limit),
				slog.Int("line_length", n))
			return
		default:
			limit -= n
		}
		log.Warn(strconv.Quote(html.UnescapeString(string(line))))
	}
}

func (self *ResponseHandler) Context() context.Context {
	return self.httpResponse.Request.Context()
}

func (self *ResponseHandler) Status() string {
	return self.httpResponse.Status
}

func (self *ResponseHandler) StatusCode() int {
	return self.httpResponse.StatusCode
}

func (self *ResponseHandler) Header(key string) string {
	return self.httpResponse.Header.Get(key)
}

func (self *ResponseHandler) Err() error { return self.clientErr }

func (self *ResponseHandler) URL() *url.URL { return self.httpResponse.Request.URL }

func (self *ResponseHandler) EffectiveURL() string { return self.URL().String() }

func (self *ResponseHandler) ContentType() string {
	return self.httpResponse.Header.Get("Content-Type")
}

func (self *ResponseHandler) LastModified() string {
	// Ignore caching headers for feeds that do not want any cache.
	if self.httpResponse.Header.Get("Expires") == "0" {
		return ""
	}
	return self.httpResponse.Header.Get("Last-Modified")
}

func (self *ResponseHandler) ETag() string {
	// Ignore caching headers for feeds that do not want any cache.
	if self.httpResponse.Header.Get("Expires") == "0" {
		return ""
	}
	return self.httpResponse.Header.Get("ETag")
}

func (self *ResponseHandler) ExpiresInMinutes() int {
	expiresHeaderValue := self.httpResponse.Header.Get("Expires")
	if expiresHeaderValue != "" {
		t, err := time.Parse(time.RFC1123, expiresHeaderValue)
		if err == nil {
			return int(math.Ceil(time.Until(t).Minutes()))
		}
	}
	return 0
}

func (self *ResponseHandler) CacheControlMaxAgeInMinutes() int {
	cacheControlHeaderValue := self.httpResponse.Header.Get("Cache-Control")
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

func (self *ResponseHandler) parseRetryDelay() time.Duration {
	retryAfter := self.Header("Retry-After")
	if retryAfter == "" {
		return 0
	}

	// First, try to parse as an integer (number of seconds)
	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return time.Duration(max(0, seconds)) * time.Second
	}

	// If not an integer, try to parse as an HTTP-date
	t, err := time.Parse(time.RFC1123, retryAfter)
	if err != nil || t.Before(time.Now()) {
		return 0
	}
	return time.Until(t)
}

func (self *ResponseHandler) rateLimited() bool {
	return self.httpResponse != nil &&
		self.httpResponse.StatusCode == http.StatusTooManyRequests
}

func (self *ResponseHandler) IsModified(lastEtagValue, lastModifiedValue string) bool {
	if self.httpResponse.StatusCode == http.StatusNotModified {
		return false
	}

	if self.ETag() != "" {
		return self.ETag() != lastEtagValue
	}

	if self.LastModified() != "" {
		return self.LastModified() != lastModifiedValue
	}

	return true
}

func (self *ResponseHandler) IsRedirect() bool {
	return self.httpResponse != nil &&
		(self.httpResponse.StatusCode == http.StatusMovedPermanently ||
			self.httpResponse.StatusCode == http.StatusFound ||
			self.httpResponse.StatusCode == http.StatusSeeOther ||
			self.httpResponse.StatusCode == http.StatusTemporaryRedirect ||
			self.httpResponse.StatusCode == http.StatusPermanentRedirect)
}

func (self *ResponseHandler) Close() {
	if self.Err() != nil {
		return
	}
	BodyClose(self.httpResponse.Body)
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

func (self *ResponseHandler) getReader(maxBodySize int64) io.ReadCloser {
	logging.FromContext(self.httpResponse.Request.Context()).Debug(
		"Request response",
		slog.String("effective_url", self.EffectiveURL()),
		slog.String("content_length", self.httpResponse.Header.Get("Content-Length")),
		slog.String("content_encoding",
			self.httpResponse.Header.Get("Content-Encoding")),
		slog.String("content_type", self.httpResponse.Header.Get("Content-Type")))
	return http.MaxBytesReader(nil, self.httpResponse.Body, maxBodySize)
}

func (self *ResponseHandler) Body() io.ReadCloser {
	return self.getReader(self.maxBodySize)
}

func (self *ResponseHandler) ReadBody() ([]byte, *locale.LocalizedErrorWrapper) {
	var buffer bytes.Buffer
	if err := self.WriteBodyTo(&buffer); err != nil {
		return nil, err
	}

	if buffer.Len() == 0 {
		return nil, locale.NewLocalizedErrorWrapper(
			errors.New("fetcher: empty response body"),
			"error.http_empty_response_body")
	}
	return buffer.Bytes(), nil
}

func (self *ResponseHandler) WriteBodyTo(w io.Writer,
) *locale.LocalizedErrorWrapper {
	_, err := io.Copy(w, self.Body())
	if err == nil || errors.Is(err, io.EOF) {
		return nil
	}

	if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return locale.NewLocalizedErrorWrapper(
			fmt.Errorf("fetcher: response body too large: %d bytes",
				maxBytesErr.Limit),
			"error.http_response_too_large")
	}

	return locale.NewLocalizedErrorWrapper(
		fmt.Errorf("fetcher: unable to read response body: %w", err),
		"error.http_body_read", err)
}
