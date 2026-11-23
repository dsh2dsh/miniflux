package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/logging"
)

func (self *Server) makeReadinessProbe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logging.FromContext(r.Context())

		if err := self.store.Ping(r.Context()); err != nil {
			l.Error("Readiness probe failed: unable to reach the database",
				slog.Any("error", err))
			http.Error(w, "Database Connection Error", http.StatusServiceUnavailable)
			return
		}

		if err := self.pool.Err(); err != nil {
			l.Error("Readiness probe failed: refresh of feeds completed with error",
				slog.Any("error", err))
			http.Error(w, "Refresh of feeds completed with error",
				http.StatusServiceUnavailable)
			return
		}

		schedulerFreq := config.PollingFrequency()
		if d := self.pool.SinceSchedulerCompleted(); d > schedulerFreq*2 {
			http.Error(w, fmt.Sprintf("Slow scheduler: %s", d),
				http.StatusServiceUnavailable)
			return
		}
		livenessProbe(w, r)
	}
}

func livenessProbe(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("OK"))
}
