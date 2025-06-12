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

func (self *TraceStat) incQuery(d time.Duration) {
	atomic.AddInt64(&self.Queries, 1)
	atomic.AddInt64((*int64)(&self.Elapsed), d.Nanoseconds())
}

func (self *TraceStat) Add(t *TraceStat) {
	self.Queries += t.Queries
	self.Elapsed += t.Elapsed
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
	if t := TraceStatFrom(ctx); t != nil {
		queryData := ctx.Value(traceQueryDataKey).(*traceQueryData)
		t.incQuery(time.Since(queryData.startTime))
	}
}
