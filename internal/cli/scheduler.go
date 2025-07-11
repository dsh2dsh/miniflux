// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"context"
	"log/slog"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/reader/fetcher"
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

	timer := time.NewTimer(d)
	defer timer.Stop()

forLoop:
	for {
		select {
		case <-ctx.Done():
			break forLoop
		case <-timer.C:
			slog.Info("feed scheduler got tick")
			for {
				hasJobs := refreshFeeds(ctx, self.store, self.pool, batchSize,
					errorLimit)
				if !hasJobs {
					slog.Info("scheduler: no jobs is a good news")
					break
				}
				self.pool.SchedulerCompleted()
				slog.Info("scheduler: check for more jobs")
			}
		case <-self.pool.WakeupSignal():
			slog.Info("feed scheduler got wakeup signal")
			refreshFeeds(ctx, self.store, self.pool, batchSize, errorLimit)
		}
		fetcher.ExpireHostLimits(d)
		timer.Reset(d)
		self.pool.SchedulerCompleted()
	}
	slog.Info("feed scheduler stopped",
		slog.Any("reason", context.Cause(ctx)))
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
