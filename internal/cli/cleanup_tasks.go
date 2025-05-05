// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"context"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"miniflux.app/v2/internal/config"
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
	nbSessions := store.CleanOldSessions(ctx,
		config.Opts.CleanupRemoveSessionsDays())
	nbUserSessions := store.CleanOldUserSessions(ctx,
		config.Opts.CleanupRemoveSessionsDays())
	slog.Info("Sessions cleanup completed",
		slog.Int64("application_sessions_removed", nbSessions),
		slog.Int64("user_sessions_removed", nbUserSessions),
	)

	startTime := time.Now()
	rowsAffected, err := store.ArchiveEntries(ctx, model.EntryStatusRead,
		config.Opts.CleanupArchiveReadDays(),
		config.Opts.CleanupArchiveBatchSize())
	if err != nil {
		slog.Error("Unable to archive read entries", slog.Any("error", err))
	} else {
		slog.Info("Archiving read entries completed",
			slog.Int64("read_entries_archived", rowsAffected),
		)

		if config.Opts.HasMetricsCollector() {
			metric.ArchiveEntriesDuration.WithLabelValues(model.EntryStatusRead).Observe(time.Since(startTime).Seconds())
		}
	}

	startTime = time.Now()
	rowsAffected, err = store.ArchiveEntries(ctx, model.EntryStatusUnread,
		config.Opts.CleanupArchiveUnreadDays(),
		config.Opts.CleanupArchiveBatchSize())
	if err != nil {
		slog.Error("Unable to archive unread entries", slog.Any("error", err))
	} else {
		slog.Info("Archiving unread entries completed",
			slog.Int64("unread_entries_archived", rowsAffected),
		)

		if config.Opts.HasMetricsCollector() {
			metric.ArchiveEntriesDuration.WithLabelValues(model.EntryStatusUnread).Observe(time.Since(startTime).Seconds())
		}
	}
}
