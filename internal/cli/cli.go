// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"miniflux.app/v2/internal/cli/logger"
	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/version"
)

var (
	flagConfigFile string
	flagDebugMode  bool

	logCloser io.Closer
)

var Cmd = cobra.Command{
	Use:     "miniflux",
	Short:   "Miniflux is a minimalist and opinionated feed reader.",
	Version: version.Version,

	PersistentPreRunE: persistentPreRunE,

	RunE: func(cmd *cobra.Command, args []string) error { return RunDaemon() },

	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if logCloser != nil {
			logCloser.Close()
		}
	},
}

var configDumpCmd = cobra.Command{
	Use:   "config-dump",
	Short: "Print parsed configuration values",
	Args:  cobra.ExactArgs(0),
	Run:   func(cmd *cobra.Command, args []string) { fmt.Print(config.Opts) },
}

var migrateCmd = cobra.Command{
	Use:   "migrate",
	Short: "Run SQL migrations",
	Args:  cobra.ExactArgs(0),

	RunE: func(cmd *cobra.Command, args []string) error {
		return withStorage(func(store *storage.Storage) error {
			return store.Migrate(context.Background())
		})
	},
}

var resetFeedErrorsCmd = cobra.Command{
	Use:   "reset-feed-errors",
	Short: "Clear all feed errors for all users",
	Args:  cobra.ExactArgs(0),

	RunE: func(cmd *cobra.Command, args []string) error {
		return withStorage(func(store *storage.Storage) error {
			return store.ResetFeedErrors(context.Background())
		})
	},
}

var resetFeedNextCmd = cobra.Command{
	Use:   "reset-feed-next-check-at",
	Short: "Reset the next check time for all feeds",
	Args:  cobra.ExactArgs(0),

	RunE: func(cmd *cobra.Command, args []string) error {
		return withStorage(func(store *storage.Storage) error {
			return store.ResetNextCheckAt(context.Background())
		})
	},
}

func init() {
	Cmd.PersistentFlags().StringVarP(&flagConfigFile, "config-file", "c", "",
		"Path to configuration file")
	Cmd.PersistentFlags().BoolVarP(&flagDebugMode, "debug", "d", false,
		"Show debug logs")

	Cmd.AddCommand(&cleanupTasksCmd)
	Cmd.AddCommand(&configDumpCmd)
	Cmd.AddCommand(&createAdminCmd)
	Cmd.AddCommand(&exportUserFeedsCmd)
	Cmd.AddCommand(&flushSessionsCmd)
	Cmd.AddCommand(&healthCmd)
	Cmd.AddCommand(&infoCmd)
	Cmd.AddCommand(&migrateCmd)
	Cmd.AddCommand(&refreshFeedsCmd)
	Cmd.AddCommand(&resetFeedErrorsCmd)
	Cmd.AddCommand(&resetFeedNextCmd)
	Cmd.AddCommand(&resetPassCmd)
}

func persistentPreRunE(cmd *cobra.Command, args []string) error {
	// Don't show usage on app errors.
	// https://github.com/spf13/cobra/issues/340#issuecomment-378726225
	cmd.SilenceUsage = true

	if err := config.Load(flagConfigFile); err != nil {
		return err
	} else if flagDebugMode {
		config.Opts.SetLogLevel("debug")
	}

	closer, err := logger.InitializeDefaultLogger()
	if err != nil {
		return err
	}
	logCloser = closer
	return nil
}

func printErrorAndExit(err error) {
	fmt.Fprintf(os.Stderr, "%v\n", err)
	os.Exit(1)
}

func withStorage(fn func(store *storage.Storage) error) error {
	if config.Opts.IsDefaultDatabaseURL() {
		slog.Info("The default value for DATABASE_URL is used")
	}

	ctx := context.Background()
	store, err := storage.New(ctx,
		config.Opts.DatabaseURL(),
		config.Opts.DatabaseMaxConns(),
		config.Opts.DatabaseMinConns(),
		config.Opts.DatabaseConnectionLifetime())
	if err != nil {
		return err
	}
	defer store.Close(ctx)

	if err := store.Ping(ctx); err != nil {
		return err
	}
	return fn(store)
}

func Execute() {
	if err := Cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
