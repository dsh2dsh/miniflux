package server

import (
	"fmt"
	"net/http"

	"miniflux.app/v2/internal/config"
)

func (self *Server) makeReadinessProbe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := self.store.Ping(r.Context()); err != nil {
			http.Error(w, fmt.Sprintf("Database Connection Error: %s", err),
				http.StatusServiceUnavailable)
			return
		}

		if err := self.pool.Err(); err != nil {
			http.Error(w,
				fmt.Sprintf("refresh of feeds completed with error: %s", err),
				http.StatusServiceUnavailable)
		}

		schedulerFreq := config.PollingFrequency()
		if d := self.pool.SinceSchedulerCompleted(); d > schedulerFreq*2 {
			http.Error(w, fmt.Sprintf("slow scheduler: %s", d),
				http.StatusServiceUnavailable)
		}
		livenessProbe(w, r)
	}
}

func livenessProbe(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
