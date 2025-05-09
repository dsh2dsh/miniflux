// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"context"
	"log/slog"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/worker"
)

func (self *Daemon) runScheduler(ctx context.Context, pool *worker.Pool) {
	slog.Info(`Starting background scheduler...`)

	self.g.Go(func() error {
		self.feedScheduler(ctx, pool,
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

func (self *Daemon) feedScheduler(ctx context.Context, pool *worker.Pool,
	freq, batchSize, errorLimit int,
) {
	d := time.Duration(freq) * time.Minute
	slog.Info("feed scheduler started", slog.Duration("freq", d))

	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("feed scheduler stopped")
			return
		case <-ticker.C:
			// Generate a batch of feeds for any user that has feeds to refresh.
			batchBuilder := self.store.NewBatchBuilder()
			batchBuilder.WithBatchSize(batchSize)
			batchBuilder.WithErrorLimit(errorLimit)
			batchBuilder.WithoutDisabledFeeds()
			batchBuilder.WithNextCheckExpired()

			if jobs, err := batchBuilder.FetchJobs(ctx); err != nil {
				slog.Error("Unable to fetch jobs from database",
					slog.Any("error", err))
			} else if len(jobs) > 0 {
				slog.Info("Created a batch of feeds", slog.Int("nb_jobs", len(jobs)))
				pool.Push(ctx, jobs)
			}
		}
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
