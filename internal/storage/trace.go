package storage

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
)

type ctxTraceStat struct{}

var TraceStatKey ctxTraceStat = struct{}{}

type TraceStat struct {
	Elapsed time.Duration
	Queries int64
}

func WithTraceStat(ctx context.Context) context.Context {
	return context.WithValue(ctx, TraceStatKey, &TraceStat{})
}

func TraceStatFrom(ctx context.Context) *TraceStat {
	if s, ok := ctx.Value(TraceStatKey).(*TraceStat); ok {
		return s
	}
	return nil
}

type ctxTraceQueryData struct{}

var traceQueryDataKey ctxTraceQueryData = struct{}{}

type traceQueryData struct {
	startTime time.Time
}

type queryTracer struct{}

var _ pgx.QueryTracer = (*queryTracer)(nil)

func (self queryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	return context.WithValue(ctx, traceQueryDataKey, &traceQueryData{
		startTime: time.Now(),
	})
}

func (self queryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn,
	data pgx.TraceQueryEndData,
) {
	if s := TraceStatFrom(ctx); s != nil {
		queryData := ctx.Value(traceQueryDataKey).(*traceQueryData)
		atomic.AddInt64((*int64)(&s.Elapsed),
			time.Since(queryData.startTime).Nanoseconds())
		atomic.AddInt64(&s.Queries, 1)
	}
}
