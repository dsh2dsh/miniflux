package server

import (
	"fmt"
	"net/http"

	"miniflux.app/v2/internal/storage"
)

func makeReadinessProbe(store *storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := store.Ping(r.Context()); err != nil {
			http.Error(w, fmt.Sprintf("Database Connection Error: %q", err),
				http.StatusServiceUnavailable)
			return
		}
		livenessProbe(w, r)
	}
}

func livenessProbe(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
