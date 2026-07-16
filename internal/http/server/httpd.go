// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server // import "miniflux.app/v2/internal/http/server"

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/api"
	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/fever"
	"miniflux.app/v2/internal/googlereader"
	"miniflux.app/v2/internal/http/middleware"
	"miniflux.app/v2/internal/metric"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/template"
	"miniflux.app/v2/internal/ui"
	"miniflux.app/v2/internal/worker"
)

type Server struct {
	store     *storage.Storage
	pool      *worker.Pool
	templates *template.Engine
	g         *errgroup.Group
	listener  net.Listener

	httpServer *http.Server
}

func New(
	store *storage.Storage,
	pool *worker.Pool,
	templates *template.Engine,
	g *errgroup.Group,
	listener net.Listener,
) *Server {
	self := &Server{
		store:     store,
		pool:      pool,
		templates: templates,
		g:         g,
		listener:  listener,
	}
	return self.init()
}

func (self *Server) init() *Server {
	self.httpServer = &http.Server{
		ReadTimeout:  config.HTTPServerTimeout(),
		WriteTimeout: config.HTTPServerTimeout(),
		IdleTimeout:  config.HTTPServerTimeout(),
		Handler:      self.httpHandler(),
	}
	return self.start()
}

func (self *Server) Shutdown(ctx context.Context) error {
	if err := self.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("http/server: shutdown http server: %w", err)
	}
	return nil
}

func (self *Server) start() *Server {
	certFile := config.CertFile()
	keyFile := config.CertKeyFile()
	certDomain := config.CertDomain()
	listenAddr := config.ListenAddr()

	switch {
	case os.Getenv("LISTEN_PID") == strconv.Itoa(os.Getpid()):
		self.startSystemdServer()
	case strings.HasPrefix(listenAddr, "/"):
		self.startUnixServer(listenAddr)
	case certDomain != "":
		config.EnableHTTPS()
		self.startAutoCertServer(certDomain)
	case certFile != "" && keyFile != "":
		config.EnableHTTPS()
		self.httpServer.Addr = listenAddr
		self.startTLSServer(certFile, keyFile)
	default:
		self.httpServer.Addr = listenAddr
		self.startPlainServer()
	}
	return self
}

func (self *Server) startSystemdServer() {
	self.g.Go(func() error {
		defer self.listener.Close()
		slog.Info(`Starting server using systemd socket`)
		if err := self.httpServer.Serve(self.listener); err != http.ErrServerClosed {
			slog.Error("failed serve on systemd socket", slog.Any("error", err))
			return fmt.Errorf(
				"http/server: failed serve on systemd socket: %w", err)
		}
		return nil
	})
}

func (self *Server) startUnixServer(path string) {
	self.g.Go(func() error {
		defer self.listener.Close()
		slog.Info("Starting server using a Unix socket",
			slog.String("socket", path))
		if err := self.httpServer.Serve(self.listener); err != http.ErrServerClosed {
			slog.Error("failed serve on unix socket",
				slog.String("socket", path), slog.Any("error", err))
			return fmt.Errorf(
				"http/server: failed serve on unix socket %q: %w", path, err)
		}
		return nil
	})
}

func (self *Server) startAutoCertServer(certDomain string) {
	self.httpServer.Addr = ":https"
	certManager := autocert.Manager{
		Cache:      self.store.NewCertificateCache(),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(certDomain),
	}
	self.httpServer.TLSConfig.GetCertificate = certManager.GetCertificate
	self.httpServer.TLSConfig.NextProtos = []string{"h2", "http/1.1", acme.ALPNProto}

	// Handle http-01 challenge.
	s := &http.Server{
		Handler: certManager.HTTPHandler(nil),
		Addr:    ":http",
	}

	self.g.Go(func() error {
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("failed serve http-01 challenge", slog.Any("error", err))
			return fmt.Errorf(
				"http/server: failed serve http-01 challenge: %w", err)
		}
		return nil
	})

	self.g.Go(func() error {
		slog.Info("Starting TLS server using automatic certificate management",
			slog.String("listen_address", self.httpServer.Addr),
			slog.String("domain", certDomain))
		if err := self.httpServer.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
			slog.Error(
				"failed serve TLS server with automatic certificate management",
				slog.Any("error", err))
			return fmt.Errorf(
				"http/server: failed serve auto cert TLS server: %w", err)
		}
		return nil
	})
}

func (self *Server) startTLSServer(certFile, keyFile string) {
	self.g.Go(func() error {
		slog.Info("Starting TLS server using a certificate",
			slog.String("listen_address", self.httpServer.Addr),
			slog.String("cert_file", certFile),
			slog.String("key_file", keyFile))
		err := self.httpServer.ListenAndServeTLS(certFile, keyFile)
		if err != http.ErrServerClosed {
			slog.Error("failed serve TLS server", slog.Any("error", err))
			return fmt.Errorf("http/server: failed serve TLS server: %w", err)
		}
		return nil
	})
}

func (self *Server) startPlainServer() {
	self.g.Go(func() error {
		slog.Info("Starting HTTP server",
			slog.String("listen_address", self.httpServer.Addr))
		if err := self.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("failed serve plain HTTP server", slog.Any("error", err))
			return fmt.Errorf("http/server: failed serve plain HTTP server: %w", err)
		}
		return nil
	})
}

func (self *Server) httpHandler() http.Handler {
	serveMux := self.templates.Router()

	// These routes do not take the base path into consideration and are always
	// available at the root of the server.
	readinessProbe := self.makeReadinessProbe()
	serveMux.HandleFunc("/liveness", livenessProbe).
		HandleFunc("/healthz", livenessProbe).
		HandleFunc("/readiness", readinessProbe).
		HandleFunc("/readyz", readinessProbe)

	m := serveMux
	if config.BasePath() != "" {
		m = serveMux.PrefixGroup(config.BasePath())
	}
	m.HandleFunc("/healthcheck", readinessProbe)

	m.Use(middleware.Gzip, middleware.RequestId, middleware.ClientIP)

	if config.HasMetricsCollector() {
		m.Handle("/metrics", metric.Handler(self.store))
	}

	if config.HasMaintenanceMode() {
		m.Use(func(http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(config.MaintenanceMessage()))
			})
		})
	}

	m.Use(middleware.WithAccessLog(), middleware.WithPanic)

	fever.Serve(m, self.store)
	googlereader.Serve(m, self.store, self.templates)
	if config.HasAPI() {
		api.Serve(m, self.store, self.pool, self.templates)
	}
	ui.Serve(m, self.store, self.pool, self.templates)
	return serveMux
}
