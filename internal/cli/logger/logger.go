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
	w, closer, err := parseLogFile(config.Opts.LogFile())
	if err != nil {
		return nil, err
	}
	h := parseFormat(w)
	slog.SetDefault(slog.New(h))
	return closer, nil
}

func parseLogFile(logFile string) (io.Writer, io.Closer, error) {
	switch logFile {
	case "stdout":
		return os.Stdout, nil, nil
	case "stderr":
		return os.Stderr, nil, nil
	}

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
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

func parseFormat(w io.Writer) slog.Handler {
	opts := &slog.HandlerOptions{Level: parseLogLevel(config.Opts.LogLevel())}
	if !config.Opts.LogDateTime() {
		opts.ReplaceAttr = hideTime
	}

	switch config.Opts.LogFormat() {
	case "human":
		return NewHumanTextHandler(w, opts, config.Opts.LogDateTime())
	case "json":
		return slog.NewJSONHandler(w, opts)
	}
	return slog.NewTextHandler(w, opts)
}
