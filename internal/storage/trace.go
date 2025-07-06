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
	if d != 0 {
		atomic.AddInt64((*int64)(&self.Elapsed), d.Nanoseconds())
	}
}

func (self *TraceStat) elapsedSince(t time.Time) {
	d := time.Since(t)
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
	batch     bool
	startTime time.Time
}

type queryTracer struct{}

var (
	_ pgx.CopyFromTracer = (*queryTracer)(nil)
	_ pgx.BatchTracer    = (*queryTracer)(nil)
	_ pgx.QueryTracer    = (*queryTracer)(nil)
)

func (self queryTracer) TraceCopyFromStart(ctx context.Context, conn *pgx.Conn,
	data pgx.TraceCopyFromStartData,
) context.Context {
	return context.WithValue(ctx, traceQueryDataKey, &traceQueryData{
		startTime: time.Now(),
	})
}

func (self queryTracer) TraceCopyFromEnd(ctx context.Context, conn *pgx.Conn,
	data pgx.TraceCopyFromEndData,
) {
	t := TraceStatFrom(ctx)
	if t == nil {
		return
	}

	queryData := ctx.Value(traceQueryDataKey).(*traceQueryData)
	t.incQuery(time.Since(queryData.startTime))
}

func (self queryTracer) TraceBatchStart(ctx context.Context, conn *pgx.Conn,
	data pgx.TraceBatchStartData,
) context.Context {
	return context.WithValue(ctx, traceQueryDataKey, &traceQueryData{
		batch:     true,
		startTime: time.Now(),
	})
}

func (self queryTracer) TraceBatchQuery(ctx context.Context, conn *pgx.Conn,
	data pgx.TraceBatchQueryData) {
}

func (self queryTracer) TraceBatchEnd(ctx context.Context, conn *pgx.Conn,
	data pgx.TraceBatchEndData,
) {
	t := TraceStatFrom(ctx)
	if t == nil {
		return
	}

	queryData := ctx.Value(traceQueryDataKey).(*traceQueryData)
	t.elapsedSince(queryData.startTime)
}

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
	t := TraceStatFrom(ctx)
	if t == nil {
		return
	}

	queryData := ctx.Value(traceQueryDataKey).(*traceQueryData)
	if queryData.batch {
		t.incQuery(0)
	} else {
		t.incQuery(time.Since(queryData.startTime))
	}
}
