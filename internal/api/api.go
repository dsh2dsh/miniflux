// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	"net/http"
	"runtime"

	"miniflux.app/v2/internal/http/mux"
	"miniflux.app/v2/internal/http/response"
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

	m.HandleFunc("POST /users", response.CreatedJSON(handler.createUser)).
		HandleFunc("GET /users", response.JSON(handler.users)).
		HandleFunc("GET /users/{userID}", response.JSON(handler.userByID)).
		HandleFunc("PUT /users/{userID}",
			response.CreatedJSON(handler.updateUser)).
		HandleFunc("DELETE /users/{userID}",
			response.NoContentJSON(handler.removeUser)).
		HandleFunc("/users/{userID}/mark-all-as-read",
			response.NoContentJSON(handler.markUserAsRead)).
		HandleFunc("/me", response.JSON(handler.currentUser)).
		HandleFunc("POST /categories",
			response.CreatedJSON(handler.createCategory)).
		HandleFunc("GET /categories", response.JSON(handler.getCategories)).
		HandleFunc("PUT /categories/{categoryID}",
			response.CreatedJSON(handler.updateCategory)).
		HandleFunc("DELETE /categories/{categoryID}",
			response.NoContentJSON(handler.removeCategory)).
		HandleFunc("/categories/{categoryID}/mark-all-as-read",
			response.NoContentJSON(handler.markCategoryAsRead)).
		HandleFunc("/categories/{categoryID}/feeds",
			response.JSON(handler.getCategoryFeeds)).
		HandleFunc("/categories/{categoryID}/refresh",
			response.NoContentJSON(handler.refreshCategory)).
		HandleFunc("/categories/{categoryID}/entries",
			response.JSON(handler.getCategoryEntries)).
		HandleFunc("/categories/{categoryID}/entries/{entryID}",
			response.JSON(handler.getCategoryEntry)).
		HandleFunc("/discover", response.JSON(handler.discoverSubscriptions)).
		HandleFunc("POST /feeds", response.CreatedJSON(handler.createFeed)).
		HandleFunc("GET /feeds", response.JSON(handler.getFeeds)).
		HandleFunc("GET /feeds/counters", response.JSON(handler.fetchCounters)).
		HandleFunc("PUT /feeds/refresh",
			response.NoContentJSON(handler.refreshAllFeeds)).
		HandleFunc("/feeds/{feedID}/refresh",
			response.NoContentJSON(handler.refreshFeed)).
		HandleFunc("GET /feeds/{feedID}", response.JSON(handler.getFeed)).
		HandleFunc("PUT /feeds/{feedID}",
			response.CreatedJSON(handler.updateFeed)).
		HandleFunc("DELETE /feeds/{feedID}",
			response.NoContentJSON(handler.removeFeed)).
		HandleFunc("/feeds/{feedID}/icon",
			response.JSON(handler.getIconByFeedID)).
		HandleFunc("/feeds/{feedID}/mark-all-as-read",
			response.NoContentJSON(handler.markFeedAsRead)).
		HandleFunc("/export", handler.exportFeeds).
		HandleFunc("/import", response.CreatedJSON(handler.importFeeds)).
		HandleFunc("POST /import/entries",
			response.CreatedJSON(handler.importEntries)).
		HandleFunc("/feeds/{feedID}/entries",
			response.JSON(handler.getFeedEntries)).
		HandleFunc("/feeds/{feedID}/entries/{entryID}",
			response.JSON(handler.getFeedEntry)).
		HandleFunc("GET /entries", response.JSON(handler.getEntries)).
		HandleFunc("PUT /entries", response.NoContentJSON(handler.setEntryStatus)).
		HandleFunc("GET /entries/{entryID}", response.JSON(handler.getEntry)).
		HandleFunc("PUT /entries/{entryID}",
			response.CreatedJSON(handler.updateEntry)).
		HandleFunc("/entries/{entryID}/bookmark",
			response.NoContentJSON(handler.toggleBookmark)).
		HandleFunc("/entries/{entryID}/save",
			response.AcceptedJSON(handler.saveEntry)).
		HandleFunc("/entries/{entryID}/fetch-content",
			response.JSON(handler.fetchContent)).
		HandleFunc("PUT /entries/{entryID}/enclosure/{at}",
			response.NoContentJSON(handler.updateEnclosureAt)).
		HandleFunc("/flush-history", response.AcceptedJSON(handler.flushHistory)).
		HandleFunc("/icons/{iconID}", response.JSON(handler.getIconByIconID)).
		HandleFunc("/integrations/status",
			response.JSON(handler.getIntegrationsStatus)).
		HandleFunc("/version", response.JSON(handler.versionHandler)).
		HandleFunc("POST /api-keys", response.CreatedJSON(handler.createAPIKey)).
		HandleFunc("GET /api-keys", response.JSON(handler.getAPIKeys)).
		HandleFunc("/api-keys/{apiKeyID}",
			response.NoContentJSON(handler.deleteAPIKey))
}

func (h *handler) versionHandler(w http.ResponseWriter, r *http.Request,
) (*VersionResponse, error) {
	return &VersionResponse{
		Version:   version.Version,
		Commit:    version.Commit,
		BuildDate: version.BuildDate,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Arch:      runtime.GOARCH,
		OS:        runtime.GOOS,
	}, nil
}
