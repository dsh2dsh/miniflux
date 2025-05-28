// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package response // import "miniflux.app/v2/internal/http/response"

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/klauspost/compress/gzhttp"
)

const (
	compressionThreshold = 1024
	longCacheControl     = "public, max-age=31536000, immutable"
)

// Builder generates HTTP responses.
type Builder struct {
	w          http.ResponseWriter
	r          *http.Request
	statusCode int
	headers    map[string]string
	body       any
}

// WithStatus uses the given status code to build the response.
func (b *Builder) WithStatus(statusCode int) *Builder {
	b.statusCode = statusCode
	return b
}

// WithHeader adds the given HTTP header to the response.
func (b *Builder) WithHeader(key, value string) *Builder {
	b.headers[key] = value
	return b
}

// WithBody uses the given body to build the response.
func (b *Builder) WithBody(body any) *Builder {
	b.body = body
	return b
}

// WithAttachment forces the document to be downloaded by the web browser.
func (b *Builder) WithAttachment(filename string) *Builder {
	b.headers["Content-Disposition"] = "attachment; filename=" + filename
	return b
}

// WithoutCompression disables HTTP compression.
func (b *Builder) WithoutCompression() *Builder {
	b.headers[gzhttp.HeaderNoCompression] = "yes"
	return b
}

// WithCaching adds caching headers to the response.
func (b *Builder) WithCaching(etag string, duration time.Duration, callback func(*Builder)) {
	b.headers["ETag"] = etag
	b.headers["Cache-Control"] = "public"
	b.headers["Expires"] = time.Now().Add(duration).UTC().Format(http.TimeFormat)

	if etag == b.r.Header.Get("If-None-Match") {
		b.statusCode = http.StatusNotModified
		b.body = nil
		b.Write()
	} else {
		callback(b)
	}
}

func (b *Builder) WithLongCaching() *Builder {
	b.headers["Cache-Control"] = longCacheControl
	return b
}

// Write generates the HTTP response.
func (b *Builder) Write() {
	if b.body == nil {
		b.writeHeaders()
		return
	}

	switch v := b.body.(type) {
	case []byte:
		b.write(v)
	case string:
		b.write([]byte(v))
	case error:
		b.write([]byte(v.Error()))
	case io.Reader:
		// Compression not implemented in this case
		b.writeHeaders()
		_, err := io.Copy(b.w, v)
		if err != nil {
			slog.Error("Unable to write response body", slog.Any("error", err))
		}
	}
}

func (b *Builder) writeHeaders() {
	b.headers["X-Content-Type-Options"] = "nosniff"
	b.headers["X-Frame-Options"] = "DENY"
	b.headers["Referrer-Policy"] = "no-referrer"

	for key, value := range b.headers {
		b.w.Header().Set(key, value)
	}

	b.w.WriteHeader(b.statusCode)
}

func (b *Builder) write(data []byte) {
	b.writeHeaders()
	if _, err := b.w.Write(data); err != nil {
		slog.Error("http/response: unable to write response",
			slog.Any("error", err))
	}
}

// New creates a new response builder.
func New(w http.ResponseWriter, r *http.Request) *Builder {
	return &Builder{
		w:          w,
		r:          r,
		statusCode: http.StatusOK,
		headers:    make(map[string]string),
	}
}
