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
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/version"
)

var (
	flagConfigFile string
	flagConfigYAML string
	flagDebugMode  bool

	logCloser io.Closer
)

var Cmd = cobra.Command{
	Use:     "miniflux",
	Short:   "Miniflux is a minimalist and opinionated feed reader.",
	Version: version.Version,

	PersistentPreRunE: persistentPreRunE,

	RunE: func(cmd *cobra.Command, args []string) error {
		if err := NewDaemon().Run(); err != nil {
			slog.Error("daemon exited with error", slog.Any("error", err))
			return err
		}
		return nil
	},

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
		return withStorage(
			func(ctx context.Context, store *storage.Storage) error {
				return store.Migrate(ctx)
			})
	},
}

var resetFeedErrorsCmd = cobra.Command{
	Use:   "reset-feed-errors",
	Short: "Clear all feed errors for all users",
	Args:  cobra.ExactArgs(0),

	RunE: func(cmd *cobra.Command, args []string) error {
		return withStorage(
			func(ctx context.Context, store *storage.Storage) error {
				return store.ResetFeedErrors(ctx)
			})
	},
}

var resetFeedNextCmd = cobra.Command{
	Use:   "reset-feed-next-check-at",
	Short: "Reset the next check time for all feeds",
	Args:  cobra.ExactArgs(0),

	RunE: func(cmd *cobra.Command, args []string) error {
		return withStorage(
			func(ctx context.Context, store *storage.Storage) error {
				return store.ResetNextCheckAt(ctx)
			})
	},
}

func init() {
	Cmd.PersistentFlags().StringVarP(&flagConfigFile, "config-file", "c", "",
		"Path to .env configuration file")
	Cmd.PersistentFlags().StringVarP(&flagConfigYAML, "config-yaml", "", "",
		"Path to YAML configuration file")
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

	if err := config.LoadYAML(flagConfigYAML, flagConfigFile); err != nil {
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

func withStorage(fn func(ctx context.Context, store *storage.Storage) error,
) error {
	ctx := context.Background()
	store, err := makeStorage(ctx)
	if err != nil {
		return err
	}
	defer store.Close(ctx)
	return fn(ctx, store)
}

func makeStorage(ctx context.Context) (*storage.Storage, error) {
	if config.Opts.IsDefaultDatabaseURL() {
		logging.FromContext(ctx).Info("The default value for DATABASE_URL is used")
	}

	store, err := storage.New(ctx,
		config.Opts.DatabaseURL(),
		config.Opts.DatabaseMaxConns(),
		config.Opts.DatabaseMinConns(),
		config.Opts.DatabaseConnectionLifetime())
	if err != nil {
		return nil, err
	}

	if err := store.Ping(ctx); err != nil {
		store.Close(ctx)
		return nil, err
	}
	return store, nil
}

func Execute() {
	if err := Cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
