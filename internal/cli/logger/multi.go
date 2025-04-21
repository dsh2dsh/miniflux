package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"
)

func NewMultiHandler(handlers []slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

type MultiHandler struct {
	closers  []io.Closer
	handlers []slog.Handler
}

var _ slog.Handler = (*MultiHandler)(nil)

func (self *MultiHandler) WithClosers(closers []io.Closer) *MultiHandler {
	self.closers = closers
	return self
}

func (self *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range self.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (self *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for i, h := range self.handlers {
		if !h.Enabled(ctx, r.Level) {
			continue
		}
		if err := h.Handle(ctx, r.Clone()); err != nil {
			err = fmt.Errorf("logger: handler #%v failed: %w", i, err)
			self.logInternalErr(ctx, i, err)
			return err
		}
	}
	return nil
}

func (self *MultiHandler) logInternalErr(ctx context.Context, i int, err error) {
	if i == 0 || !self.handlers[0].Enabled(ctx, slog.LevelError) {
		return
	}
	r := slog.NewRecord(time.Now(), slog.LevelError, "log handler failed",
		uintptr(0))
	r.AddAttrs(slog.Any("error", err))
	_ = self.handlers[0].Handle(ctx, r)
}

func (self *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(self.handlers))
	for i := range self.handlers {
		handlers[i] = self.handlers[i].WithAttrs(attrs)
	}
	return &MultiHandler{handlers: handlers}
}

func (self *MultiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(self.handlers))
	for i := range self.handlers {
		handlers[i] = self.handlers[i].WithGroup(name)
	}
	return &MultiHandler{handlers: handlers}
}

func (self *MultiHandler) Close() error {
	for _, closer := range self.closers {
		if closer != nil {
			_ = closer.Close()
		}
	}
	return nil
}
