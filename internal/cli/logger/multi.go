package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"
)

func NewMultiHandler(handlers []slog.Handler) *MultiHandler {
	self := &MultiHandler{
		MultiHandler: slog.NewMultiHandler(handlers...),
		h0:           handlers[0],
	}
	return self
}

type MultiHandler struct {
	*slog.MultiHandler

	h0      slog.Handler
	closers []io.Closer
}

var _ slog.Handler = (*MultiHandler)(nil)

func (self *MultiHandler) WithClosers(closers []io.Closer) *MultiHandler {
	self.closers = closers
	return self
}

func (self *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	err := self.MultiHandler.Handle(ctx, r)
	if err == nil {
		return nil
	}

	err = fmt.Errorf("logger: one of handlers failed: %w", err)
	self.logInternalErr(ctx, err)
	return err
}

func (self *MultiHandler) logInternalErr(ctx context.Context, err error) {
	if !self.h0.Enabled(ctx, slog.LevelError) {
		return
	}

	r := slog.NewRecord(time.Now(), slog.LevelError, "unable log message",
		uintptr(0))
	r.AddAttrs(slog.Any("error", err))
	_ = self.h0.Handle(ctx, r)
}

func (self *MultiHandler) Close() error {
	for _, closer := range self.closers {
		if closer != nil {
			_ = closer.Close()
		}
	}
	return nil
}
