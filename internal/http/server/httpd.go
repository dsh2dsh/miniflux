// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server // import "miniflux.app/v2/internal/http/server"

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"

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
	serverType int

	store     *storage.Storage
	pool      *worker.Pool
	templates *template.Engine
	g         *errgroup.Group
	listener  net.Listener

	certFile string
	keyFile  string
	cert     *tls.Certificate
	mu       sync.RWMutex

	httpServer *http.Server
}

func New() (*Server, error) {
	self := &Server{
		serverType: plainServer,
		certFile:   config.CertFile(),
		keyFile:    config.CertKeyFile(),

		httpServer: &http.Server{
			ReadTimeout:  config.HTTPServerTimeout(),
			WriteTimeout: config.HTTPServerTimeout(),
			IdleTimeout:  config.HTTPServerTimeout(),
		},
	}
	self.detectServerType()

	l, err := newListener(self.serverType)
	if err != nil {
		return nil, err
	}
	self.listener = l

	if err := self.loadCert(); err != nil {
		return nil, err
	}
	return self, nil
}

func (self *Server) loadCert() error {
	if self.serverType != tlsServer || !self.tlsConfigured() {
		return nil
	}

	slog.Info("load TLS certificate",
		slog.String("cert", self.certFile),
		slog.String("key", self.keyFile))

	cert, err := tls.LoadX509KeyPair(self.certFile, self.keyFile)
	if err != nil {
		slog.Error("unable load TLS certificate", slog.Any("error", err))
		return fmt.Errorf("load TLS cert from cert=%q, key=%q: %w",
			self.certFile, self.keyFile, err)
	}

	self.mu.Lock()
	self.cert = &cert
	self.mu.Unlock()
	return nil
}

func (self *Server) Start(
	store *storage.Storage,
	pool *worker.Pool,
	templates *template.Engine,
	g *errgroup.Group,
) *Server {
	self.store = store
	self.pool = pool
	self.templates = templates
	self.g = g

	self.httpServer.Handler = self.httpHandler()

	switch self.serverType {
	case systemdServer:
		self.startSystemdServer()
	case unixServer:
		self.startUnixServer(config.ListenAddr())
	case autoCertServer:
		config.EnableHTTPS()
		self.startAutoCertServer(config.CertDomain())
	case tlsServer:
		config.EnableHTTPS()
		self.tlsStart(config.ListenAddr())
	default:
		self.startPlainServer(config.ListenAddr())
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
	certManager := &autocert.Manager{
		Cache:      self.store.NewCertificateCache(),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(certDomain),
	}
	self.httpServer.TLSConfig = &tls.Config{
		GetCertificate: certManager.GetCertificate,
		NextProtos:     []string{"h2", "http/1.1", acme.ALPNProto},
	}

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

func (self *Server) tlsStart(addr string) {
	self.httpServer.Addr = addr
	self.httpServer.TLSConfig = &tls.Config{GetCertificate: self.tlsCertificate}

	self.g.Go(func() error {
		slog.Info("Starting TLS server using a certificate",
			slog.String("listen_address", self.httpServer.Addr),
			slog.String("cert_file", self.certFile),
			slog.String("key_file", self.keyFile))
		err := self.httpServer.ListenAndServeTLS("", "")
		if err != http.ErrServerClosed {
			slog.Error("failed serve TLS server", slog.Any("error", err))
			return fmt.Errorf("http/server: failed serve TLS server: %w", err)
		}
		return nil
	})
}

func (self *Server) tlsCertificate(*tls.ClientHelloInfo) (*tls.Certificate,
	error,
) {
	self.mu.RLock()
	cert := self.cert
	self.mu.RUnlock()
	return cert, nil
}

func (self *Server) startPlainServer(addr string) {
	self.httpServer.Addr = addr
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

func (self *Server) Reload() {
	if self.cert != nil {
		_ = self.loadCert()
	}
}

func (self *Server) Shutdown(ctx context.Context) error {
	if err := self.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("http/server: shutdown http server: %w", err)
	}
	return nil
}
