// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"context"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/metric"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/storage"
)

var cleanupTasksCmd = cobra.Command{
	Use:   "run-cleanup-tasks",
	Short: "Run cleanup tasks (delete old sessions and archives old entries)",
	Args:  cobra.ExactArgs(0),

	RunE: func(cmd *cobra.Command, args []string) error {
		return withStorage(
			func(ctx context.Context, store *storage.Storage) error {
				runCleanupTasks(ctx, store)
				return nil
			})
	},
}

func runCleanupTasks(ctx context.Context, store *storage.Storage) {
	removed := store.CleanOldSessions(ctx,
		config.Opts.CleanupRemoveSessionsDays(),
		config.Opts.CleanupInactiveSessionsDays())

	log := logging.FromContext(ctx)
	log.Info("Sessions cleanup completed", slog.Int64("removed", removed))

	startTime := time.Now()
	rows, err := store.ArchiveEntries(ctx, model.EntryStatusRead,
		config.Opts.CleanupArchiveReadDays(),
		config.Opts.CleanupArchiveBatchSize())
	if err != nil {
		log.Error("Unable to archive read entries", slog.Any("error", err))
	} else {
		log.Info("Archiving read entries completed",
			slog.Int64("read_entries_archived", rows),
			slog.Duration("elapsed", time.Since(startTime)))

		if config.Opts.HasMetricsCollector() {
			metric.ArchiveEntriesDuration.
				WithLabelValues(model.EntryStatusRead).
				Observe(time.Since(startTime).Seconds())
		}
	}

	startTime = time.Now()
	rows, err = store.ArchiveEntries(ctx, model.EntryStatusUnread,
		config.Opts.CleanupArchiveUnreadDays(),
		config.Opts.CleanupArchiveBatchSize())
	if err != nil {
		log.Error("Unable to archive unread entries", slog.Any("error", err))
	} else {
		log.Info("Archiving unread entries completed",
			slog.Int64("unread_entries_archived", rows),
			slog.Duration("elapsed", time.Since(startTime)))

		if config.Opts.HasMetricsCollector() {
			metric.ArchiveEntriesDuration.
				WithLabelValues(model.EntryStatusUnread).
				Observe(time.Since(startTime).Seconds())
		}
	}
}
