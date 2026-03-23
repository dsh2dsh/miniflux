// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package response // import "miniflux.app/v2/internal/http/response"

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/klauspost/compress/gzhttp"

	"miniflux.app/v2/internal/logging"
)

const longCacheControl = "public, max-age=31536000, immutable"

// Builder generates HTTP responses.
type Builder struct {
	w          http.ResponseWriter
	r          *http.Request
	statusCode int
	headers    map[string]string
	body       any
}

// New creates a new response builder.
func New(w http.ResponseWriter, r *http.Request, opts ...Option) *Builder {
	b := &Builder{
		w:          w,
		r:          r,
		statusCode: http.StatusOK,
		headers:    make(map[string]string),
	}

	for _, fn := range opts {
		fn(b)
	}
	return b
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

// WithBodyAsBytes uses the given bytes to build the response.
func (b *Builder) WithBodyAsBytes(body []byte) *Builder {
	b.body = body
	return b
}

// WithBodyAsBytes uses the given error to build the response.
func (b *Builder) WithBodyAsError(body error) *Builder {
	b.body = body
	return b
}

// WithBodyAsString uses the given string to build the response.
func (b *Builder) WithBodyAsString(body string) *Builder {
	b.body = body
	return b
}

// WithBodyAsReader uses the given reader to build the response.
func (b *Builder) WithBodyAsReader(body io.Reader) *Builder {
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
func (b *Builder) WithCaching(contentHash string, duration time.Duration,
	callback func(*Builder),
) {
	etag := `"` + contentHash + `"`
	b.headers["ETag"] = etag
	b.headers["Cache-Control"] = "public"
	b.headers["Expires"] = time.Now().Add(duration).UTC().Format(http.TimeFormat)

	ifNoneMatch := strings.TrimSpace(b.r.Header.Get("If-None-Match"))
	if ifNoneMatch != etag {
		callback(b)
		return
	}

	b.statusCode = http.StatusNotModified
	b.body = nil
	b.Write()
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
			b.logger().Error("Unable to write response body", slog.Any("error", err))
		}
	}
}

func (b *Builder) logger() *slog.Logger {
	return logging.FromContext(b.r.Context())
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
		b.logger().Error("http/response: unable to write response",
			slog.Any("error", err))
	}
}
