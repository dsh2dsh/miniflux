// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/database"
	httpd "miniflux.app/v2/internal/http/server"
	"miniflux.app/v2/internal/metric"
	"miniflux.app/v2/internal/proxyrotator"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/systemd"
	"miniflux.app/v2/internal/ui/static"
	"miniflux.app/v2/internal/worker"
)

func RunDaemon() error {
	return withStorage(func(db *sql.DB, store *storage.Storage) error {
		if err := configureDaemon(db, store); err != nil {
			return err
		}
		startDaemon(store)
		return nil
	})
}

func configureDaemon(db *sql.DB, store *storage.Storage) error {
	// Run migrations and start the daemon.
	if config.Opts.RunMigrations() {
		if err := database.Migrate(db); err != nil {
			return err
		}
	}

	if err := database.IsSchemaUpToDate(db); err != nil {
		return err
	}

	if config.Opts.CreateAdmin() {
		if err := createAdminUserFromEnvironmentVariables(store); err != nil {
			return err
		}
	}

	if config.Opts.HasHTTPClientProxiesConfigured() {
		slog.Info("Initializing proxy rotation",
			slog.Int("proxies_count", len(config.Opts.HTTPClientProxies())))
		rotatorInstance, err := proxyrotator.NewProxyRotator(
			config.Opts.HTTPClientProxies())
		if err != nil {
			return err
		}
		proxyrotator.ProxyRotatorInstance = rotatorInstance
	}
	return generateBundles()
}

func generateBundles() error {
	if err := static.CalculateBinaryFileChecksums(); err != nil {
		return fmt.Errorf("unable to calculate binary file checksums: %w", err)
	}

	if err := static.GenerateStylesheetsBundles(); err != nil {
		return fmt.Errorf("unable to generate stylesheets bundles: %w", err)
	}

	if err := static.GenerateJavascriptBundles(); err != nil {
		return fmt.Errorf("unable to generate javascript bundles: %w", err)
	}
	return nil
}

func startDaemon(store *storage.Storage) {
	slog.Info("Starting daemon...")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	signal.Notify(stop, syscall.SIGTERM)

	pool := worker.NewPool(store, config.Opts.WorkerPoolSize())

	if config.Opts.HasSchedulerService() && !config.Opts.HasMaintenanceMode() {
		runScheduler(store, pool)
	}

	var httpServer *http.Server
	if config.Opts.HasHTTPService() {
		httpServer = httpd.StartWebServer(store, pool)
	}

	if config.Opts.HasMetricsCollector() {
		collector := metric.NewCollector(store, config.Opts.MetricsRefreshInterval())
		go collector.GatherStorageMetrics()
	}

	if systemd.HasNotifySocket() {
		slog.Debug("Sending readiness notification to Systemd")

		if err := systemd.SdNotify(systemd.SdNotifyReady); err != nil {
			slog.Error("Unable to send readiness notification to systemd", slog.Any("error", err))
		}

		if config.Opts.HasWatchdog() && systemd.HasSystemdWatchdog() {
			slog.Debug("Activating Systemd watchdog")

			go func() {
				interval, err := systemd.WatchdogInterval()
				if err != nil {
					slog.Error("Unable to get watchdog interval from systemd", slog.Any("error", err))
					return
				}

				for {
					if err := store.Ping(); err != nil {
						slog.Error("Unable to ping database", slog.Any("error", err))
					} else {
						if err := systemd.SdNotify(systemd.SdNotifyWatchdog); err != nil {
							slog.Error("cli: failed notify systemd watchdog",
								slog.Any("error", err))
						}
					}

					time.Sleep(interval / 3)
				}
			}()
		}
	}

	<-stop
	slog.Info("Shutting down the process")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if httpServer != nil {
		if err := httpServer.Shutdown(ctx); err != nil {
			slog.Error("cli: failed shutdown http server", slog.Any("error", err))
		}
	}
	slog.Info("Process gracefully stopped")
}
