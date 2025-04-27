package logging

import (
	"context"
	"log/slog"
)

type ctxKey struct{}

var ctxKeyLogger ctxKey = struct{}{}

func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKeyLogger).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

func With(ctx context.Context, args ...any) context.Context {
	return WithLogger(ctx, FromContext(ctx).With(args...))
}

func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKeyLogger, l)
}
