package middleware

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"miniflux.app/v2/internal/logging"
)

func WithPanic(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				//nolint:errorlint // we are checking exactly ErrAbortHandler
				if err == http.ErrAbortHandler {
					// we don't recover http.ErrAbortHandler so the response
					// to the client is aborted, this should not be logged
					panic(err)
				}
				logPanic(r, err)
				if r.Header.Get("Connection") != "Upgrade" {
					http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
				}
			}
		}()
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func logPanic(r *http.Request, err any) {
	log := logging.FromContext(r.Context())
	log.Error("request aborted with panic", slog.Any("reason", err))

	for line := range bytes.Lines(debug.Stack()) {
		line = bytes.Replace(line, []byte("\t"), []byte("  "), 1)
		line = bytes.TrimRight(line, "\n")
		log.Error("panic: " + string(line))
	}
}
