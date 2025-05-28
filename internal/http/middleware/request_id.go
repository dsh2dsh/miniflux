package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"sync/atomic"

	"miniflux.app/v2/internal/logging"
)

type ctxRequestId struct{}

var requestIdKey ctxRequestId = struct{}{}

func genRequestId() uint64 {
	return atomic.AddUint64(&nextRequestId, 1)
}

var nextRequestId uint64

func RequestId(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		id := genRequestId()
		ctx := context.WithValue(r.Context(), requestIdKey,
			strconv.FormatUint(id, 10))
		ctx = logging.WithLogger(ctx,
			logging.FromContext(ctx).With(slog.Uint64("rid", id)))
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

func RequestIdFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(requestIdKey).(string); ok {
		return id
	}
	return ""
}
