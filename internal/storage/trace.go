package storage

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type ctxTraceKey struct{}

var ctxTraceStats ctxTraceKey = struct{}{}

type traceStats struct {
	queries int
}

func withTraceStats(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTraceStats, &traceStats{})
}

func traceStatsFrom(ctx context.Context) *traceStats {
	if s, ok := ctx.Value(ctxTraceStats).(*traceStats); ok {
		return s
	}
	return nil
}

type queryTracer struct{}

var _ pgx.QueryTracer = (*queryTracer)(nil)

func (self queryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	if s := traceStatsFrom(ctx); s != nil {
		s.queries++
	}
	return ctx
}

func (self queryTracer) TraceQueryEnd(_ context.Context, conn *pgx.Conn,
	data pgx.TraceQueryEndData,
) {
}
