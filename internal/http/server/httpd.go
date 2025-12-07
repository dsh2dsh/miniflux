// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server // import "miniflux.app/v2/internal/http/server"

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
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
	"miniflux.app/v2/internal/version"
	"miniflux.app/v2/internal/worker"
)

func Listener() (net.Listener, error) {
	if !config.HasHTTPService() {
		return nil, nil
	}

	var listener net.Listener
	listenAddr := config.ListenAddr()

	switch {
	case os.Getenv("LISTEN_PID") == strconv.Itoa(os.Getpid()):
		f := os.NewFile(3, "systemd socket")
		l, err := net.FileListener(f)
		if err != nil {
			return nil, fmt.Errorf(
				"http/server: create listener from systemd socket: %w", err)
		}
		listener = l
	case strings.HasPrefix(listenAddr, "/"):
		l, err := unixListener(listenAddr, 0o666)
		if err != nil {
			return nil, fmt.Errorf("create unix listener on %q: %w", listenAddr, err)
		}
		listener = l
	}
	return listener, nil
}

func unixListener(path string, mode uint32) (*net.UnixListener, error) {
	if err := unlinkStaleUnix(path); err != nil {
		return nil, err
	}

	laddr, err := net.ResolveUnixAddr("unix", path)
	if err != nil {
		return nil, fmt.Errorf("http/server: resolve unix address: %w", err)
	}

	l, err := net.ListenUnix("unix", laddr)
	if err != nil {
		return nil, fmt.Errorf("http/server: listen unix: %w", err)
	}

	l.SetUnlinkOnClose(true)
	if mode == 0 {
		return l, nil
	}

	if err := os.Chmod(path, os.FileMode(mode)); err != nil {
		return nil, fmt.Errorf(
			"http/server: change socket mode to %O: %w", mode, err)
	}
	return l, nil
}

func unlinkStaleUnix(path string) error {
	sockdir := filepath.Dir(path)
	stat, err := os.Stat(sockdir)
	switch {
	case err != nil && os.IsNotExist(err):
		if err := os.MkdirAll(sockdir, 0o755); err != nil {
			return fmt.Errorf("http/server: cannot mkdir %q: %w", sockdir, err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("http/server: cannot stat(2) %q: %w", sockdir, err)
	case !stat.IsDir():
		return fmt.Errorf("http/server: not a directory: %q", sockdir)
	}

	_, err = os.Stat(path)
	switch {
	case err == nil:
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("http/server: cannot remove stale socket: %w", err)
		}
	case !os.IsNotExist(err):
		return fmt.Errorf("http/server: cannot stat(2): %w", err)
	}
	return nil
}

func StartWebServer(store *storage.Storage, pool *worker.Pool,
	templates *template.Engine, g *errgroup.Group, listener net.Listener,
) *http.Server {
	certFile := config.CertFile()
	keyFile := config.CertKeyFile()
	certDomain := config.CertDomain()
	listenAddr := config.ListenAddr()
	server := &http.Server{
		ReadTimeout:  config.HTTPServerTimeout(),
		WriteTimeout: config.HTTPServerTimeout(),
		IdleTimeout:  config.HTTPServerTimeout(),
		Handler:      setupHandler(store, pool, templates),
	}

	switch {
	case os.Getenv("LISTEN_PID") == strconv.Itoa(os.Getpid()):
		startSystemdSocketServer(server, listener, g)
	case strings.HasPrefix(listenAddr, "/"):
		startUnixSocketServer(server, listenAddr, listener, g)
	case certDomain != "":
		config.EnableHTTPS()
		startAutoCertTLSServer(server, certDomain, store, g)
	case certFile != "" && keyFile != "":
		config.EnableHTTPS()
		server.Addr = listenAddr
		startTLSServer(server, certFile, keyFile, g)
	default:
		server.Addr = listenAddr
		startHTTPServer(server, g)
	}
	return server
}

func startSystemdSocketServer(server *http.Server, listener net.Listener,
	g *errgroup.Group,
) {
	g.Go(func() error {
		defer listener.Close()
		slog.Info(`Starting server using systemd socket`)
		if err := server.Serve(listener); err != http.ErrServerClosed {
			slog.Error("failed serve on systemd socket", slog.Any("error", err))
			return fmt.Errorf(
				"http/server: failed serve on systemd socket: %w", err)
		}
		return nil
	})
}

func startUnixSocketServer(server *http.Server, path string,
	listener net.Listener, g *errgroup.Group,
) {
	g.Go(func() error {
		defer listener.Close()
		slog.Info("Starting server using a Unix socket",
			slog.String("socket", path))
		if err := server.Serve(listener); err != http.ErrServerClosed {
			slog.Error("failed serve on unix socket",
				slog.String("socket", path), slog.Any("error", err))
			return fmt.Errorf(
				"http/server: failed serve on unix socket %q: %w", path, err)
		}
		return nil
	})
}

func startAutoCertTLSServer(server *http.Server, certDomain string,
	store *storage.Storage, g *errgroup.Group,
) {
	server.Addr = ":https"
	certManager := autocert.Manager{
		Cache:      store.NewCertificateCache(),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(certDomain),
	}
	server.TLSConfig.GetCertificate = certManager.GetCertificate
	server.TLSConfig.NextProtos = []string{"h2", "http/1.1", acme.ALPNProto}

	// Handle http-01 challenge.
	s := &http.Server{
		Handler: certManager.HTTPHandler(nil),
		Addr:    ":http",
	}

	g.Go(func() error {
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("failed serve http-01 challenge", slog.Any("error", err))
			return fmt.Errorf(
				"http/server: failed serve http-01 challenge: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		slog.Info("Starting TLS server using automatic certificate management",
			slog.String("listen_address", server.Addr),
			slog.String("domain", certDomain))
		if err := server.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
			slog.Error(
				"failed serve TLS server with automatic certificate management",
				slog.Any("error", err))
			return fmt.Errorf(
				"http/server: failed serve auto cert TLS server: %w", err)
		}
		return nil
	})
}

func startTLSServer(server *http.Server, certFile, keyFile string,
	g *errgroup.Group,
) {
	g.Go(func() error {
		slog.Info("Starting TLS server using a certificate",
			slog.String("listen_address", server.Addr),
			slog.String("cert_file", certFile),
			slog.String("key_file", keyFile))
		err := server.ListenAndServeTLS(certFile, keyFile)
		if err != http.ErrServerClosed {
			slog.Error("failed serve TLS server", slog.Any("error", err))
			return fmt.Errorf("http/server: failed serve TLS server: %w", err)
		}
		return nil
	})
}

func startHTTPServer(server *http.Server, g *errgroup.Group) {
	g.Go(func() error {
		slog.Info("Starting HTTP server",
			slog.String("listen_address", server.Addr))
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("failed serve plain HTTP server", slog.Any("error", err))
			return fmt.Errorf("http/server: failed serve plain HTTP server: %w", err)
		}
		return nil
	})
}

func setupHandler(store *storage.Storage, pool *worker.Pool,
	templates *template.Engine,
) http.Handler {
	serveMux := templates.Router()

	// These routes do not take the base path into consideration and are always
	// available at the root of the server.
	readinessProbe := makeReadinessProbe(store, pool)
	serveMux.HandleFunc("/liveness", livenessProbe).
		HandleFunc("/healthz", livenessProbe).
		HandleFunc("/readiness", readinessProbe).
		HandleFunc("/readyz", readinessProbe)

	m := serveMux
	if config.BasePath() != "" {
		m = serveMux.PrefixGroup(config.BasePath())
	}
	m.HandleFunc("/healthcheck", readinessProbe)
	m.HandleFunc("/version", handleVersion)

	m.Use(middleware.Gzip, middleware.RequestId, middleware.ClientIP)

	if config.HasMetricsCollector() {
		m.Handle("/metrics", metric.Handler(store))
	}

	if config.HasMaintenanceMode() {
		m.Use(func(http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(config.MaintenanceMessage()))
			})
		})
	}

	m.Use(middleware.WithAccessLog(), middleware.WithPanic)

	fever.Serve(m, store)
	googlereader.Serve(m, store)
	if config.HasAPI() {
		api.Serve(m, store, pool)
	}
	ui.Serve(m, store, pool)
	return serveMux
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(version.Version))
}
