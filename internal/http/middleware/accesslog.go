package middleware

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/storage"
)

type ctxAccessLog struct{}

var accessLogKey ctxAccessLog = struct{}{}

type accessLogRequest struct {
	User *model.User
}

func AccessLogUser(ctx context.Context, u *model.User) {
	if r, ok := ctx.Value(accessLogKey).(*accessLogRequest); ok {
		r.User = u
	}
}

func WithAccessLog(prefixes ...string) MiddlewareFunc {
	m := make(map[string]struct{})
	for _, prefix := range prefixes {
		m[prefix] = struct{}{}
	}

	fn := func(next http.Handler) http.Handler {
		return &AccessLog{m: m, next: next}
	}
	return fn
}

type AccessLog struct {
	m    map[string]struct{}
	next http.Handler
}

func (self *AccessLog) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logRequest := accessLogRequest{User: request.User(r)}
	ctx := context.WithValue(r.Context(), accessLogKey, &logRequest)
	ctx, traceStat := storage.WithTraceStat(ctx)

	sw := newStatusResponseWriter(w)
	startTime := time.Now()
	self.next.ServeHTTP(sw, r.WithContext(ctx))

	log := logging.FromContext(ctx).With(
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("proto", r.Proto))

	if u := logRequest.User; u != nil {
		log = log.With(slog.GroupAttrs("user",
			slog.Int64("id", u.ID),
			slog.String("name", u.Username)))
	}

	if traceStat.Queries > 0 {
		log = log.With(slog.GroupAttrs("storage",
			slog.Int64("queries", traceStat.Queries),
			slog.Duration("elapsed", traceStat.Elapsed)))
	}

	methodURL := r.Method + " " + r.URL.RequestURI()
	log.LogAttrs(ctx, self.level(r), methodURL,
		slog.Int("status_code", sw.StatusCode()),
		slog.Int("size", sw.Size()),
		slog.Duration("request_time", time.Since(startTime)))
}

func (self *AccessLog) level(r *http.Request) slog.Level {
	p := r.URL.Path
	if _, ok := self.m[p]; ok {
		return slog.LevelDebug
	}

	for s := range self.m {
		if strings.HasPrefix(p, s) {
			return slog.LevelDebug
		}
	}
	return slog.LevelInfo
}

func newStatusResponseWriter(w http.ResponseWriter) *statusResponseWriter {
	return &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

type statusResponseWriter struct {
	http.ResponseWriter

	statusCode    int
	headerWritten bool
	size          int
}

var (
	_ io.ReaderFrom       = (*statusResponseWriter)(nil)
	_ http.ResponseWriter = (*statusResponseWriter)(nil)
)

func (self *statusResponseWriter) StatusCode() int { return self.statusCode }
func (self *statusResponseWriter) Size() int       { return self.size }

func (self *statusResponseWriter) WriteHeader(statusCode int) {
	self.ResponseWriter.WriteHeader(statusCode)
	if !self.headerWritten {
		self.statusCode = statusCode
		self.headerWritten = true
	}
}

func (self *statusResponseWriter) Write(b []byte) (n int, err error) {
	self.headerWritten = true
	n, err = self.ResponseWriter.Write(b)
	self.size += n
	return n, err //nolint:wrapcheck // return as is
}

func (self *statusResponseWriter) Unwrap() http.ResponseWriter {
	return self.ResponseWriter
}

func (self *statusResponseWriter) ReadFrom(r io.Reader) (n int64, err error) {
	self.headerWritten = true
	switch v := self.ResponseWriter.(type) {
	case io.ReaderFrom:
		n, err = v.ReadFrom(r)
	default:
		n, err = io.Copy(self.ResponseWriter, r)
	}
	self.size += int(n)
	return n, err //nolint:wrapcheck // return as is
}
