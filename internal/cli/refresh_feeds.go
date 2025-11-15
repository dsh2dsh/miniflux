// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/worker"
)

var refreshFeedsCmd = cobra.Command{
	Use:   "refresh-feeds",
	Short: "Refresh a batch of feeds and exit",
	Args:  cobra.ExactArgs(0),

	RunE: func(cmd *cobra.Command, args []string) error {
		return withStorage(runRefreshFeeds)
	},
}

func runRefreshFeeds(ctx context.Context, store *storage.Storage) error {
	if err := store.SchemaUpToDate(ctx); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pool := worker.NewPool(ctx, store, config.Opts.WorkerPoolSize())
	g, ctx := errgroup.WithContext(ctx)
	g.Go(pool.Run)

	refreshFeeds(ctx, store, pool)

	cancel()
	if err := g.Wait(); err != nil {
		return fmt.Errorf("waiting for worker pool: %w", err)
	}
	return nil
}

func refreshFeeds(ctx context.Context, store *storage.Storage,
	pool *worker.Pool,
) bool {
	// Generate a batch of feeds for any user that has feeds to refresh.
	batch := store.NewBatchBuilder().
		WithBatchSize(config.Opts.BatchSize()).
		WithNextCheckExpired().
		WithoutDisabledFeeds()

	if d := config.Opts.PollingErrorRetry(); d == 0 {
		batch.WithErrorLimit(config.Opts.PollingErrorLimit())
	}

	if jobs, err := batch.FetchJobs(ctx); err != nil {
		slog.Error("Unable to fetch jobs from database", slog.Any("error", err))
	} else if len(jobs) > 0 {
		pool.Push(ctx, jobs)
		return true
	}
	return false
}
