// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/mux"
	"miniflux.app/v2/internal/http/server"
	"miniflux.app/v2/internal/metric"
	"miniflux.app/v2/internal/proxyrotator"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/systemd"
	"miniflux.app/v2/internal/template"
	"miniflux.app/v2/internal/ui/static"
	"miniflux.app/v2/internal/worker"
)

func NewDaemon() *Daemon { return &Daemon{} }

type Daemon struct {
	store      *storage.Storage
	g          *errgroup.Group
	httpServer *http.Server
	pool       *worker.Pool
	templates  *template.Engine
}

func (self *Daemon) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGTERM, os.Interrupt)
	defer cancel()

	slog.Info("Starting daemon...")
	defer self.close(ctx)

	if err := self.configure(ctx); err != nil {
		return err
	}

	if err := self.start(ctx); err != nil {
		return err
	}
	return self.wait(ctx)
}

func (self *Daemon) close(ctx context.Context) {
	if self.store != nil {
		self.store.Close(ctx)
	}
}

func (self *Daemon) configure(ctx context.Context) error {
	store, err := makeStorage(ctx)
	if err != nil {
		return err
	}
	self.store = store

	// Run migrations and start the daemon.
	if config.RunMigrations() {
		if err := self.store.Migrate(ctx); err != nil {
			return err
		}
	}

	if err := self.store.SchemaUpToDate(ctx); err != nil {
		return err
	}

	if config.CreateAdmin() {
		err := createAdminUserFromEnvironmentVariables(ctx, self.store)
		if err != nil {
			return err
		}
	}

	if config.HasHTTPClientProxiesConfigured() {
		slog.Info("Initializing proxy rotation",
			slog.Int("proxies_count", len(config.HTTPClientProxies())))
		rotatorInstance, err := proxyrotator.NewProxyRotator(
			config.HTTPClientProxies())
		if err != nil {
			return err
		}
		proxyrotator.ProxyRotatorInstance = rotatorInstance
	}

	templates, err := compileTemplates()
	if err != nil {
		return err
	}
	self.templates = templates
	return generateBundles(ctx)
}

func compileTemplates() (*template.Engine, error) {
	templates := template.NewEngine(mux.New())
	if err := templates.ParseTemplates(); err != nil {
		return nil, err
	}
	return templates, nil
}

func generateBundles(ctx context.Context) error {
	if err := static.CalculateBinaryFileChecksums(ctx); err != nil {
		return fmt.Errorf("failed calculate binary file hashes: %w", err)
	}

	if err := static.GenerateStylesheetsBundles(ctx); err != nil {
		return fmt.Errorf("failed generate css bundles: %w", err)
	}

	if err := static.GenerateJavascriptBundles(ctx); err != nil {
		return fmt.Errorf("failed generate js bundles: %w", err)
	}
	return nil
}

func (self *Daemon) start(ctx context.Context) error {
	listener, err := server.Listener()
	if err != nil {
		return err
	}

	self.g, ctx = errgroup.WithContext(ctx)
	self.pool = worker.NewPool(ctx, self.store, self.templates)
	self.g.Go(self.pool.Run)
	if config.HasSchedulerService() && !config.HasMaintenanceMode() {
		self.runScheduler(ctx)
	}

	if config.HasHTTPService() {
		self.httpServer = server.StartWebServer(self.store, self.pool,
			self.templates, self.g, listener)
	}

	if config.HasMetricsCollector() {
		metric.RegisterMetrics(self.store)
	}

	if systemd.HasNotifySocket() {
		if err := self.systemdReady(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (self *Daemon) systemdReady(ctx context.Context) error {
	slog.Debug("Sending readiness notification to Systemd")
	if err := systemd.SdNotify(systemd.SdNotifyReady); err != nil {
		return fmt.Errorf(
			"unable to send readiness notification to systemd: %w", err)
	}

	if !config.HasWatchdog() || !systemd.HasSystemdWatchdog() {
		return nil
	}

	slog.Debug("Activating Systemd watchdog")
	interval, err := systemd.WatchdogInterval()
	if err != nil {
		return fmt.Errorf("unable to get watchdog interval from systemd: %w", err)
	}

	self.g.Go(func() error {
		ticker := time.NewTicker(interval / 3)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				if err := self.store.Ping(ctx); err != nil {
					slog.Error("Unable to ping database", slog.Any("error", err))
					continue
				}
				if err := systemd.SdNotify(systemd.SdNotifyWatchdog); err != nil {
					slog.Error("Unable notify systemd watchdog", slog.Any("error", err))
				}
			}
		}
	})
	return nil
}

func (self *Daemon) wait(ctx context.Context) error {
	<-ctx.Done()
	if self.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		slog.Info("Shutting down the process gracefully...")
		if err := self.httpServer.Shutdown(ctx); err != nil {
			slog.Error("failed shutdown http server", slog.Any("error", err))
		}
	}

	if err := self.g.Wait(); err != nil {
		slog.Error("process stopped with error", slog.Any("error", err))
		return fmt.Errorf("process stopped with error: %w", err)
	}
	slog.Info("Process gracefully stopped")
	return nil
}
