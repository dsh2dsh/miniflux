// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"log/slog"
	"net/http"

	"miniflux.app/v2/internal/http/mux"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/securecookie"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/template"
	"miniflux.app/v2/internal/worker"
)

type handler struct {
	router *mux.ServeMux
	store  *storage.Storage
	tpl    *template.Engine
	pool   *worker.Pool

	secureCookie *securecookie.SecureCookie
}

// Serve declares all routes for the user interface.
func Serve(m *mux.ServeMux, store *storage.Storage, pool *worker.Pool) {
	m.HandleFunc("/robots.txt", robotsTXT)

	templateEngine := template.NewEngine(m)
	if err := templateEngine.ParseTemplates(); err != nil {
		panic(err)
	}

	handler := &handler{
		router: m,
		store:  store,
		tpl:    templateEngine,
		pool:   pool,

		secureCookie: securecookie.New(),
	}

	middleware := newMiddleware(m, store)
	m = m.Group().Use(middleware.handleUserSession, middleware.handleAppSession)

	// Authentication pages.
	m.Group().Use(middleware.handleAuthProxy, middleware.PublicCSRF).
		NameHandleFunc("/", handler.showLoginPage, "login")
	m.Group().Use(middleware.PublicCSRF).
		NameHandleFunc("/login", handler.checkLogin, "checkLogin")
	m.NameHandleFunc("/logout", handler.logout, "logout")

	// Static assets.
	m.NameHandleFunc("/css/{name}", handler.showStylesheet, "stylesheet").
		NameHandleFunc("/js/{name}", handler.showJavascript, "javascript").
		HandleFunc("/favicon.ico", handler.showFavicon).
		NameHandleFunc("/icon/{filename}", handler.showAppIcon, "appIcon").
		NameHandleFunc("/manifest.json", handler.showWebManifest, "webManifest")

	// New subscription pages.
	m.NameHandleFunc("GET /subscribe", handler.showAddSubscriptionPage,
		"addSubscription").
		NameHandleFunc("POST /subscribe", handler.submitSubscription,
			"submitSubscription").
		NameHandleFunc("/subscriptions", handler.showChooseSubscriptionPage,
			"chooseSubscription").
		NameHandleFunc("/bookmarklet", handler.bookmarklet, "bookmarklet")

	// Unread page.
	m.NameHandleFunc("/mark-all-as-read", handler.markAllAsRead,
		"markAllAsRead").
		NameHandleFunc("/unread", handler.showUnreadPage, "unread").
		NameHandleFunc("/unread/entry/{entryID}", handler.showUnreadEntryPage,
			"unreadEntry")

	// History pages.
	m.NameHandleFunc("/history", handler.showHistoryPage, "history").
		NameHandleFunc("/history/entry/{entryID}", handler.showReadEntryPage,
			"readEntry").
		NameHandleFunc("/history/flush", handler.flushHistory, "flushHistory")

	// Bookmark pages.
	m.NameHandleFunc("/starred", handler.showStarredPage, "starred").
		NameHandleFunc("/starred/entry/{entryID}", handler.showStarredEntryPage,
			"starredEntry")

	// Search pages.
	m.NameHandleFunc("/search", handler.showSearchPage, "search").
		NameHandleFunc("/search/entry/{entryID}", handler.showSearchEntryPage,
			"searchEntry")

	// Feed listing pages.
	m.NameHandleFunc("/feeds", handler.showFeedsPage, "feeds").
		NameHandleFunc("/feeds/refresh", handler.refreshAllFeeds,
			"refreshAllFeeds")

	// Individual feed pages.
	m.NameHandleFunc("/feed/{feedID}/refresh", handler.refreshFeed,
		"refreshFeed").
		NameHandleFunc("/feed/{feedID}/edit", handler.showEditFeedPage,
			"editFeed").
		NameHandleFunc("/feed/{feedID}/remove", handler.removeFeed, "removeFeed").
		NameHandleFunc("/feed/{feedID}/update", handler.updateFeed, "updateFeed").
		NameHandleFunc("/feed/{feedID}/entries", handler.showFeedEntriesPage,
			"feedEntries").
		NameHandleFunc("/feed/{feedID}/entries/all",
			handler.showFeedEntriesAllPage, "feedEntriesAll").
		NameHandleFunc("/feed/{feedID}/entry/{entryID}", handler.showFeedEntryPage,
			"feedEntry").
		NameHandleFunc("/unread/feed/{feedID}/entry/{entryID}",
			handler.showUnreadFeedEntryPage, "unreadFeedEntry").
		NameHandleFunc("/feed-icon/{externalIconID}", handler.showFeedIcon,
			"feedIcon").
		NameHandleFunc("/feed/{feedID}/mark-all-as-read", handler.markFeedAsRead,
			"markFeedAsRead")

	// Category pages.
	m.NameHandleFunc("/category/{categoryID}/entry/{entryID}",
		handler.showCategoryEntryPage, "categoryEntry").
		NameHandleFunc("/unread/category/{categoryID}/entry/{entryID}",
			handler.showUnreadCategoryEntryPage, "unreadCategoryEntry").
		NameHandleFunc("/categories", handler.showCategoryListPage, "categories").
		NameHandleFunc("/category/create", handler.showCreateCategoryPage,
			"createCategory").
		NameHandleFunc("/category/save", handler.saveCategory, "saveCategory").
		NameHandleFunc("/category/{categoryID}/feeds",
			handler.showCategoryFeedsPage, "categoryFeeds").
		NameHandleFunc("/category/{categoryID}/feed/{feedID}/remove",
			handler.removeCategoryFeed, "removeCategoryFeed").
		NameHandleFunc("/category/{categoryID}/feeds/refresh",
			handler.refreshCategoryFeedsPage, "refreshCategoryFeedsPage").
		NameHandleFunc("/category/{categoryID}/entries",
			handler.showCategoryEntriesPage, "categoryEntries").
		NameHandleFunc("/category/{categoryID}/entries/refresh",
			handler.refreshCategoryEntriesPage, "refreshCategoryEntriesPage").
		NameHandleFunc("/category/{categoryID}/entries/all",
			handler.showCategoryEntriesAllPage, "categoryEntriesAll").
		NameHandleFunc("/category/{categoryID}/entries/starred",
			handler.showCategoryEntriesStarredPage, "categoryEntriesStarred").
		NameHandleFunc("/category/{categoryID}/edit", handler.showEditCategoryPage,
			"editCategory").
		NameHandleFunc("/category/{categoryID}/update", handler.updateCategory,
			"updateCategory").
		NameHandleFunc("/category/{categoryID}/remove", handler.removeCategory,
			"removeCategory").
		NameHandleFunc("/category/{categoryID}/mark-all-as-read",
			handler.markCategoryAsRead, "markCategoryAsRead")

	// Tag pages.
	m.NameHandleFunc("/tags/{tagName}/entries/all",
		handler.showTagEntriesAllPage, "tagEntriesAll").
		NameHandleFunc("/tags/{tagName}/entry/{entryID}", handler.showTagEntryPage,
			"tagEntry")

	// Entry pages.
	m.NameHandleFunc("/entry/status", handler.updateEntriesStatus,
		"updateEntriesStatus").
		NameHandleFunc("/entry/save/{entryID}", handler.saveEntry, "saveEntry").
		NameHandleFunc("/entry/enclosure/{enclosureID}/save-progression",
			handler.saveEnclosureProgression, "saveEnclosureProgression").
		NameHandleFunc("/entry/download/{entryID}",
			handler.fetchContent, "fetchContent").
		NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}", handler.mediaProxy,
			"proxy").
		NameHandleFunc("/entry/bookmark/{entryID}", handler.toggleBookmark,
			"toggleBookmark")

	// Share pages.
	m.NameHandleFunc("/entry/share/{entryID}", handler.createSharedEntry,
		"shareEntry").
		NameHandleFunc("/entry/unshare/{entryID}", handler.unshareEntry,
			"unshareEntry").
		NameHandleFunc("/share/{shareCode}", handler.sharedEntry, "sharedEntry").
		NameHandleFunc("/shares", handler.sharedEntries, "sharedEntries")

	// User pages.
	m.NameHandleFunc("/users", handler.showUsersPage, "users").
		NameHandleFunc("/user/create", handler.showCreateUserPage, "createUser").
		NameHandleFunc("/user/save", handler.saveUser, "saveUser").
		NameHandleFunc("/users/{userID}/edit", handler.showEditUserPage,
			"editUser").
		NameHandleFunc("/users/{userID}/update", handler.updateUser, "updateUser").
		NameHandleFunc("/users/{userID}/remove", handler.removeUser, "removeUser")

	// Settings pages.
	m.NameHandleFunc("GET /settings", handler.showSettingsPage, "settings").
		NameHandleFunc("POST /settings", handler.updateSettings, "updateSettings").
		NameHandleFunc("GET /integrations", handler.showIntegrationPage,
			"integrations").
		NameHandleFunc("POST /integration", handler.updateIntegration,
			"updateIntegration").
		NameHandleFunc("/about", handler.showAboutPage, "about")

	// Session pages.
	m.NameHandleFunc("/sessions", handler.showSessionsPage, "sessions").
		NameHandleFunc("/sessions/{sessionID}/remove", handler.removeSession,
			"removeSession")

	// API Keys pages.
	m.NameHandleFunc("/keys", handler.showAPIKeysPage, "apiKeys").
		NameHandleFunc("/keys/{keyID}/delete", handler.deleteAPIKey,
			"deleteAPIKey").
		NameHandleFunc("/keys/create", handler.showCreateAPIKeyPage,
			"createAPIKey").
		NameHandleFunc("/keys/save", handler.saveAPIKey, "saveAPIKey")

	// OPML pages.
	m.NameHandleFunc("/export", handler.exportFeeds, "export").
		NameHandleFunc("/import", handler.showImportPage, "import").
		NameHandleFunc("/upload", handler.uploadOPML, "uploadOPML").
		NameHandleFunc("/fetch", handler.fetchOPML, "fetchOPML")

	// OAuth2 flow.
	m.NameHandleFunc("/oauth2/unlink/{provider}", handler.oauth2Unlink,
		"oauth2Unlink").
		NameHandleFunc("/oauth2/redirect/{provider}", handler.oauth2Redirect,
			"oauth2Redirect").
		NameHandleFunc("/oauth2/callback/{provider}", handler.oauth2Callback,
			"oauth2Callback")

	// Offline page
	m.NameHandleFunc("/offline", handler.showOfflinePage, "offline")

	// WebAuthn flow
	m.NameHandleFunc("/webauthn/register/begin", handler.beginRegistration,
		"webauthnRegisterBegin").
		NameHandleFunc("/webauthn/register/finish", handler.finishRegistration,
			"webauthnRegisterFinish").
		NameHandleFunc("/webauthn/login/begin", handler.beginLogin,
			"webauthnLoginBegin").
		NameHandleFunc("/webauthn/login/finish", handler.finishLogin,
			"webauthnLoginFinish").
		NameHandleFunc("/webauthn/deleteall", handler.deleteAllCredentials,
			"webauthnDeleteAll").
		NameHandleFunc("/webauthn/{credentialHandle}/delete",
			handler.deleteCredential, "webauthnDelete").
		NameHandleFunc("/webauthn/{credentialHandle}/rename",
			handler.renameCredential, "webauthnRename").
		NameHandleFunc("/webauthn/{credentialHandle}/save", handler.saveCredential,
			"webauthnSave")
}

func robotsTXT(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, err := w.Write([]byte("User-agent: *\nDisallow: /"))
	if err != nil {
		logging.FromContext(r.Context()).
			Error(http.StatusText(http.StatusInternalServerError),
				slog.Any("error", err),
				slog.String("client_ip", request.ClientIP(r)),
				slog.Group("request",
					slog.String("method", r.Method),
					slog.String("uri", r.RequestURI),
					slog.String("user_agent", r.UserAgent())),
				slog.Group("response",
					slog.Int("status_code", http.StatusInternalServerError)))
	}
}
