// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/model"
	feedHandler "miniflux.app/v2/internal/reader/handler"
	"miniflux.app/v2/internal/storage"
)

var refreshFeedsCmd = cobra.Command{
	Use:   "refresh-feeds",
	Short: "Refresh a batch of feeds and exit",
	Args:  cobra.ExactArgs(0),

	RunE: func(cmd *cobra.Command, args []string) error {
		return withStorage(
			func(ctx context.Context, store *storage.Storage) error {
				refreshFeeds(ctx, store)
				return nil
			})
	},
}

func refreshFeeds(ctx context.Context, store *storage.Storage) {
	var wg sync.WaitGroup

	startTime := time.Now()

	// Generate a batch of feeds for any user that has feeds to refresh.
	batchBuilder := store.NewBatchBuilder()
	batchBuilder.WithBatchSize(config.Opts.BatchSize())
	batchBuilder.WithErrorLimit(config.Opts.PollingParsingErrorLimit())
	batchBuilder.WithoutDisabledFeeds()
	batchBuilder.WithNextCheckExpired()

	jobs, err := batchBuilder.FetchJobs(ctx)
	if err != nil {
		slog.Error("Unable to fetch jobs from database", slog.Any("error", err))
		return
	}

	nbJobs := len(jobs)

	slog.Info("Created a batch of feeds",
		slog.Int("nb_jobs", nbJobs),
		slog.Int("batch_size", config.Opts.BatchSize()),
	)

	jobQueue := make(chan model.Job, nbJobs)

	slog.Info("Starting a pool of workers",
		slog.Int("nb_workers", config.Opts.WorkerPoolSize()),
	)

	for i := range config.Opts.WorkerPoolSize() {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobQueue {
				slog.Info("Refreshing feed",
					slog.Int64("feed_id", job.FeedID),
					slog.Int64("user_id", job.UserID),
					slog.Int("worker_id", workerID),
				)

				localizedError := feedHandler.RefreshFeed(ctx, store, job.UserID,
					job.FeedID, false)
				if localizedError != nil {
					slog.Warn("Unable to refresh feed",
						slog.Int64("feed_id", job.FeedID),
						slog.Int64("user_id", job.UserID),
						slog.Any("error", localizedError.Error()),
					)
				}
			}
		}(i)
	}

	for _, job := range jobs {
		jobQueue <- job
	}
	close(jobQueue)

	wg.Wait()

	slog.Info("Refreshed a batch of feeds",
		slog.Int("nb_feeds", nbJobs),
		slog.String("duration", time.Since(startTime).String()),
	)
}
