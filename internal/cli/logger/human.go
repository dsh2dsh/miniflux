package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"sync"
)

func NewHumanTextHandler(w io.Writer, opts *slog.HandlerOptions,
	logTime bool,
) *HumanTextHandler {
	b := NewBytesBuffer()
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	self := &HumanTextHandler{
		logTime: logTime,
		b:       b,
		w:       w,
		opts:    *opts,
		mu:      new(sync.Mutex),
	}
	return self.init()
}

type HumanTextHandler struct {
	logTime bool
	b       *BytesBuffer
	stdLog  *log.Logger
	w       io.Writer

	h    slog.Handler
	opts slog.HandlerOptions

	mu *sync.Mutex
}

var _ slog.Handler = (*HumanTextHandler)(nil)

func (self *HumanTextHandler) init() *HumanTextHandler {
	if self.logTime {
		self.stdLog = log.New(self.b, "", log.LstdFlags)
	}
	opts := self.opts
	opts.ReplaceAttr = self.replace
	self.h = slog.NewTextHandler(self.b, &opts)
	return self
}

func (self *HumanTextHandler) replace(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 {
		switch a.Key {
		case slog.TimeKey, slog.LevelKey, slog.MessageKey:
			return slog.Attr{}
		}
	}
	if self.opts.ReplaceAttr != nil {
		return self.opts.ReplaceAttr(groups, a)
	}
	return a
}

func (self *HumanTextHandler) Enabled(ctx context.Context, level slog.Level,
) bool {
	return self.h.Enabled(ctx, level)
}

func (self *HumanTextHandler) Handle(ctx context.Context, r slog.Record) error {
	self.lock()
	defer self.unlock()

	if err := self.formatStd(r); err != nil {
		return err
	}

	if err := self.h.Handle(ctx, r); err != nil {
		return fmt.Errorf("logger: failed slog handler: %w", err)
	}

	// Discard trailing '\n', added by slog.TextHandler, and trailing ' ' added by
	// formatStd.
	b := bytes.TrimSpace(self.b.Bytes())
	self.b.Truncate(len(b))

	self.b.WriteByte('\n')
	if _, err := self.b.WriteTo(self.w); err != nil {
		return fmt.Errorf("logger: failed write formatted entry: %w", err)
	}
	return nil
}

func (self *HumanTextHandler) lock() {
	self.mu.Lock()
	self.b.Alloc()
}

func (self *HumanTextHandler) unlock() {
	self.b.Free()
	self.mu.Unlock()
}

func (self *HumanTextHandler) formatStd(r slog.Record) error {
	if self.logTime {
		// output log.LstdFlags
		if err := self.stdLog.Output(2, ""); err != nil {
			return fmt.Errorf("logger: write prefix to log.Output: %w", err)
		}
		// Discard last byte (\n), added by log.Output.
		self.b.Truncate(self.b.Len() - 1)
	}

	self.b.WriteString(r.Level.String())
	self.b.WriteByte(' ')
	self.b.WriteString(r.Message)
	self.b.WriteByte(' ')
	return nil
}

func (self *HumanTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h := *self
	h.h = self.h.WithAttrs(attrs)
	return &h
}

func (self *HumanTextHandler) WithGroup(name string) slog.Handler {
	h := *self
	h.h = self.h.WithGroup(name)
	return &h
}
