// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"context"
	"log/slog"
	"time"

	"miniflux.app/v2/internal/config"
)

func (self *Daemon) runScheduler(ctx context.Context) {
	slog.Info(`Starting background scheduler...`)

	self.g.Go(func() error {
		self.feedScheduler(ctx,
			config.Opts.PollingFrequency(),
			config.Opts.BatchSize(),
			config.Opts.PollingParsingErrorLimit())
		return nil
	})

	self.g.Go(func() error {
		self.cleanupScheduler(ctx, config.Opts.CleanupFrequencyHours())
		return nil
	})
}

func (self *Daemon) feedScheduler(ctx context.Context,
	freq, batchSize, errorLimit int,
) {
	d := time.Duration(freq) * time.Minute
	slog.Info("feed scheduler started", slog.Duration("freq", d))

	ticker := time.NewTicker(d)
	defer ticker.Stop()

forLoop:
	for {
		select {
		case <-ctx.Done():
			break forLoop
		case <-ticker.C:
			slog.Info("feed scheduler got tick")
			self.refreshFeeds(ctx, batchSize, errorLimit)
		case <-self.pool.WakeupSignal():
			slog.Info("feed scheduler got wakeup signal")
			self.refreshFeeds(ctx, batchSize, errorLimit)
		}
	}
	slog.Info("feed scheduler stopped",
		slog.Any("reason", context.Cause(ctx)))
}

func (self *Daemon) refreshFeeds(ctx context.Context,
	batchSize, errorLimit int,
) {
	// Generate a batch of feeds for any user that has feeds to refresh.
	batch := self.store.NewBatchBuilder().
		WithBatchSize(batchSize).
		WithErrorLimit(errorLimit).
		WithoutDisabledFeeds().
		WithNextCheckExpired()

	self.pool.WithWakeup()
	if jobs, err := batch.FetchJobs(ctx); err != nil {
		slog.Error("Unable to fetch jobs from database", slog.Any("error", err))
	} else if len(jobs) > 0 {
		self.pool.Push(ctx, jobs)
	}
}

func (self *Daemon) cleanupScheduler(ctx context.Context, freq int) {
	d := time.Duration(freq) * time.Hour
	slog.Info("cleanup scheduler started", slog.Duration("freq", d))

	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("cleanup scheduler stopped")
			return
		case <-ticker.C:
			runCleanupTasks(ctx, self.store)
		}
	}
}
