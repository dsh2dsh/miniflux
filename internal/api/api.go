// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	"net/http"
	"runtime"

	"miniflux.app/v2/internal/http/mux"
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/template"
	"miniflux.app/v2/internal/version"
	"miniflux.app/v2/internal/worker"
)

const PathPrefix = "/v1"

type handler struct {
	store     *storage.Storage
	pool      *worker.Pool
	router    *mux.ServeMux
	templates *template.Engine
}

// Serve declares API routes for the application.
func Serve(m *mux.ServeMux, store *storage.Storage, pool *worker.Pool,
	t *template.Engine,
) {
	m = m.PrefixGroup(PathPrefix)
	m.Use(WithKeyAuth(store), WithBasicAuth(store), CORS, requestUser)

	handler := &handler{
		store:     store,
		pool:      pool,
		router:    m,
		templates: t,
	}

	m.HandleFunc("POST /users", handler.createUser).
		HandleFunc("GET /users", handler.users).
		HandleFunc("GET /users/{userID}", handler.userByID).
		HandleFunc("PUT /users/{userID}", handler.updateUser).
		HandleFunc("DELETE /users/{userID}", handler.removeUser).
		HandleFunc("/users/{userID}/mark-all-as-read", handler.markUserAsRead).
		HandleFunc("/me", handler.currentUser).
		HandleFunc("POST /categories", handler.createCategory).
		HandleFunc("GET /categories", handler.getCategories).
		HandleFunc("PUT /categories/{categoryID}", handler.updateCategory).
		HandleFunc("DELETE /categories/{categoryID}", handler.removeCategory).
		HandleFunc("/categories/{categoryID}/mark-all-as-read",
			handler.markCategoryAsRead).
		HandleFunc("/categories/{categoryID}/feeds", handler.getCategoryFeeds).
		HandleFunc("/categories/{categoryID}/refresh", handler.refreshCategory).
		HandleFunc("/categories/{categoryID}/entries", handler.getCategoryEntries).
		HandleFunc("/categories/{categoryID}/entries/{entryID}",
			handler.getCategoryEntry).
		HandleFunc("/discover", handler.discoverSubscriptions).
		HandleFunc("POST /feeds", handler.createFeed).
		HandleFunc("GET /feeds", handler.getFeeds).
		HandleFunc("GET /feeds/counters", handler.fetchCounters).
		HandleFunc("PUT /feeds/refresh", handler.refreshAllFeeds).
		HandleFunc("/feeds/{feedID}/refresh", handler.refreshFeed).
		HandleFunc("GET /feeds/{feedID}", handler.getFeed).
		HandleFunc("PUT /feeds/{feedID}", handler.updateFeed).
		HandleFunc("DELETE /feeds/{feedID}", handler.removeFeed).
		HandleFunc("/feeds/{feedID}/icon", handler.getIconByFeedID).
		HandleFunc("/feeds/{feedID}/mark-all-as-read", handler.markFeedAsRead).
		HandleFunc("/export", handler.exportFeeds).
		HandleFunc("/import", handler.importFeeds).
		HandleFunc("POST /import/entries", handler.importEntries).
		HandleFunc("/feeds/{feedID}/entries", handler.getFeedEntries).
		HandleFunc("/feeds/{feedID}/entries/{entryID}", handler.getFeedEntry).
		HandleFunc("GET /entries", handler.getEntries).
		HandleFunc("PUT /entries", handler.setEntryStatus).
		HandleFunc("GET /entries/{entryID}", handler.getEntry).
		HandleFunc("PUT /entries/{entryID}", handler.updateEntry).
		HandleFunc("/entries/{entryID}/bookmark", handler.toggleBookmark).
		HandleFunc("/entries/{entryID}/save", handler.saveEntry).
		HandleFunc("/entries/{entryID}/fetch-content", handler.fetchContent).
		HandleFunc("PUT /entries/{entryID}/enclosure/{at}",
			handler.updateEnclosureAt).
		HandleFunc("/flush-history", handler.flushHistory).
		HandleFunc("/icons/{iconID}", handler.getIconByIconID).
		HandleFunc("/integrations/status", handler.getIntegrationsStatus).
		HandleFunc("/version", handler.versionHandler).
		HandleFunc("POST /api-keys", handler.createAPIKey).
		HandleFunc("GET /api-keys", handler.getAPIKeys).
		HandleFunc("/api-keys/{apiKeyID}", handler.deleteAPIKey)
}

func (h *handler) versionHandler(w http.ResponseWriter, r *http.Request) {
	json.OK(w, r, &VersionResponse{
		Version:   version.Version,
		Commit:    version.Commit,
		BuildDate: version.BuildDate,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Arch:      runtime.GOARCH,
		OS:        runtime.GOOS,
	})
}
