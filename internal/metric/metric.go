// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package metric // import "miniflux.app/v2/internal/metric"

import (
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/storage"
)

// Prometheus Metrics.
var (
	BackgroundFeedRefreshDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "miniflux",
			Name:      "background_feed_refresh_duration",
			Help:      "Processing time to refresh feeds from the background workers",
			Buckets:   prometheus.LinearBuckets(1, 2, 15),
		},
		[]string{"status"},
	)

	ScraperRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "miniflux",
			Name:      "scraper_request_duration",
			Help:      "Web scraper request duration",
			Buckets:   prometheus.LinearBuckets(1, 2, 25),
		},
		[]string{"status"},
	)

	ArchiveEntriesDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "miniflux",
			Name:      "archive_entries_duration",
			Help:      "Archive entries duration",
			Buckets:   prometheus.LinearBuckets(1, 2, 30),
		},
		[]string{"status"},
	)
)

func RegisterMetrics(store *storage.Storage) {
	prometheus.MustRegister(BackgroundFeedRefreshDuration)
	prometheus.MustRegister(ScraperRequestDuration)
	prometheus.MustRegister(ArchiveEntriesDuration)
	store.RegisterMetricts()
}

func Handler(store *storage.Storage) http.Handler {
	promHandler := promhttp.Handler()
	var lastStorageMetricsAt time.Time

	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := logging.FromContext(ctx)
		if !isAllowedToAccessMetricsEndpoint(r) {
			log.Warn("Authentication failed while accessing the metrics endpoint",
				slog.String("client_ip", request.ClientIP(r)),
				slog.String("client_user_agent", r.UserAgent()),
				slog.String("client_remote_addr", r.RemoteAddr),
			)
			http.NotFound(w, r)
			return
		}

		d := time.Since(lastStorageMetricsAt)
		fromDB := d >= config.Opts.MetricsRefreshInterval()
		log.Debug("Collecting storage metrics",
			slog.Duration("elapsed", d), slog.Bool("from_db", fromDB))
		if fromDB {
			lastStorageMetricsAt = time.Now()
		}
		if err := store.Metrics(ctx, fromDB); err != nil {
			log.Error("unable collect storage metrics", slog.Any("error", err))
			html.ServerError(w, r, err)
			return
		}
		promHandler.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func isAllowedToAccessMetricsEndpoint(r *http.Request) bool {
	log := logging.FromContext(r.Context()).With(
		slog.Bool("authentication_failed", true),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("client_user_agent", r.UserAgent()),
		slog.String("client_remote_addr", r.RemoteAddr))

	needAuth := config.Opts.MetricsUsername() != "" &&
		config.Opts.MetricsPassword() != ""
	if needAuth {
		username, password, authOK := r.BasicAuth()
		switch {
		case !authOK:
			log.Warn("Metrics endpoint accessed without authentication header")
			return false
		case username == "" || password == "":
			log.Warn("Metrics endpoint accessed with empty username or password")
			return false
		case username != config.Opts.MetricsUsername() || password != config.Opts.MetricsPassword():
			log.Warn("Metrics endpoint accessed with invalid username or password")
			return false
		}
	}

	remoteIP := request.FindRemoteIP(r)
	if remoteIP == "@" {
		// This indicates a request sent via a Unix socket, always consider these
		// trusted.
		return true
	}

	for _, cidr := range config.Opts.MetricsAllowedNetworks() {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Error("Metrics endpoint accessed with invalid CIDR",
				slog.String("cidr", cidr))
			return false
		}

		// We use r.RemoteAddr in this case because HTTP headers like
		// X-Forwarded-For can be easily spoofed. The recommendation is to use HTTP
		// Basic authentication.
		if network.Contains(net.ParseIP(remoteIP)) {
			return true
		}
	}
	return false
}
