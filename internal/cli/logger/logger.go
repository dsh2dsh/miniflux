// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"miniflux.app/v2/internal/config"
)

func InitializeDefaultLogger() (io.Closer, error) {
	logs := config.Opts.Logging()
	closers := make([]io.Closer, len(logs))
	handlers := make([]slog.Handler, len(logs))

	for i := range logs {
		h, closer, err := handlerFromConfig(&logs[i])
		if err != nil {
			return nil, err
		}
		closers[i] = closer
		handlers[i] = h
	}

	if len(logs) == 1 {
		slog.SetDefault(slog.New(handlers[0]))
		return closers[0], nil
	}

	h := NewMultiHandler(handlers).WithClosers(closers)
	slog.SetDefault(slog.New(h))
	return h, nil
}

func handlerFromConfig(c *config.Log) (slog.Handler, io.Closer, error) {
	w, closer, err := parseLogFile(c.LogFile)
	if err != nil {
		return nil, nil, err
	}
	h := parseFormat(w, c.LogFormat, c.LogLevel, c.LogDateTime)
	return h, closer, nil
}

func parseLogFile(logFile string) (io.Writer, io.Closer, error) {
	switch logFile {
	case "stdout":
		return os.Stdout, nil, nil
	case "stderr":
		return os.Stderr, nil, nil
	}

	f, err := NewLogFile(logFile)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"unable to open log file %q: %w", logFile, err)
	}
	return f, f, nil
}

func parseLogLevel(s string) slog.Level {
	var level slog.Level
	switch s {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	return level
}

func hideTime(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		return slog.Attr{}
	}
	return a
}

func parseFormat(w io.Writer, format, level string, logTime bool) slog.Handler {
	opts := &slog.HandlerOptions{Level: parseLogLevel(level)}
	if !logTime {
		opts.ReplaceAttr = hideTime
	}

	switch format {
	case "human":
		return NewHumanTextHandler(w, opts, logTime)
	case "json":
		return slog.NewJSONHandler(w, opts)
	}
	return slog.NewTextHandler(w, opts)
}
