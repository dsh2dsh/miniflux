// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"log/slog"
	"net/http"

	"miniflux.app/v2/internal/config"
	hmw "miniflux.app/v2/internal/http/middleware"
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
	templateEngine := template.NewEngine(m)
	if err := templateEngine.ParseTemplates(); err != nil {
		panic(err)
	}

	h := &handler{
		router: m,
		store:  store,
		tpl:    templateEngine,
		pool:   pool,

		secureCookie: securecookie.New(),
	}

	m = m.Group().Use(hmw.CrossOriginProtection())
	mw := newMiddleware(m, store)

	// public endpoints
	m.Group(func(m *mux.ServeMux) {
		// Static assets.
		m.NameHandleFunc("/data/{filename}", h.showBinaryFile, "binaryFile")
		m.NameHandleFunc("/css/{name}", h.showStylesheet, "stylesheet")
		m.HandleFunc("/favicon.ico", h.showFavicon)
		m.NameHandleFunc("/feed-icon/{externalIconID}", h.showFeedIcon, "feedIcon")
		m.NameHandleFunc("/js/{name}", h.showJavascript, "javascript")
		m.NameHandleFunc("/manifest.json", h.showWebManifest, "webManifest")
		m.NameHandleFunc("/offline", h.showOfflinePage, "offline")
		m.HandleFunc("/robots.txt", robotsTxt)

		// Authentication pages.
		m.Group().Use(mw.handleAppSession).
			NameHandleFunc("/login", h.checkLogin, "checkLogin")

		// WebAuthn flow
		if config.Opts.WebAuthn() {
			m.NameHandleFunc("/webauthn/login/begin", h.beginLogin,
				"webauthnLoginBegin")
			m.NameHandleFunc("/webauthn/login/finish", h.finishLogin,
				"webauthnLoginFinish")
		}

		m.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}", h.mediaProxy,
			"proxy")
		m.NameHandleFunc("/share/{shareCode}", h.sharedEntry, "sharedEntry")
	})

	m = m.Group().Use(hmw.WithUserSession(store))

	// OAuth2 flow.
	if config.Opts.OAuth2Provider() != "" {
		m.NameHandleFunc("/oauth2/callback/{provider}", h.oauth2Callback,
			"oauth2Callback")
		m.NameHandleFunc("/oauth2/redirect/{provider}", h.oauth2Redirect,
			"oauth2Redirect")
	}

	// Authentication pages.
	m.Group(func(m *mux.ServeMux) {
		m.Use(mw.handleAuthProxy, mw.handleAppSession)
		m.NameHandleFunc("/{$}", h.showLoginPage, "login")
	})

	m.Group(func(m *mux.ServeMux) {
		m.Use(mw.userNoRedirect(), mw.handleAppSession)
		m.NameHandleFunc("/mark-all-as-read", h.markAllAsRead, "markAllAsRead")
		m.NameHandleFunc("/history/flush", h.flushHistory, "flushHistory")

		// Entry pages.
		m.NameHandleFunc("/entry/save-progression/{entryID}/{at}",
			h.saveEnclosureProgression, "saveEnclosureProgression")
		m.NameHandleFunc("/entry/save/{entryID}", h.saveEntry, "saveEntry")
		m.NameHandleFunc("/entry/status", h.updateEntriesStatus,
			"updateEntriesStatus")
		m.NameHandleFunc("/entry/status/count", h.updateEntriesStatusCount,
			"updateEntriesStatusCount")
		m.NameHandleFunc("/entry/bookmark/{entryID}", h.toggleBookmark,
			"toggleBookmark")

		if config.Opts.WebAuthn() {
			// WebAuthn flow
			m.NameHandleFunc("/webauthn/deleteall", h.deleteAllCredentials,
				"webauthnDeleteAll")
			m.NameHandleFunc("/webauthn/{credentialHandle}/delete", h.deleteCredential,
				"webauthnDelete")
		}
	})

	m = m.Group().Use(mw.userWithRedirect(), mw.handleAppSession)
	m.NameHandleFunc("/logout", h.logout, "logout")

	// New subscription pages.
	m.NameHandleFunc("GET /subscribe", h.showAddSubscriptionPage,
		"addSubscription")
	m.NameHandleFunc("POST /subscribe", h.submitSubscription,
		"submitSubscription")
	m.NameHandleFunc("/subscriptions", h.showChooseSubscriptionPage,
		"chooseSubscription")
	m.NameHandleFunc("/bookmarklet", h.bookmarklet, "bookmarklet")

	// Unread page.
	m.NameHandleFunc("/unread", h.showUnreadPage, "unread")
	m.NameHandleFunc("/unread/entry/{entryID}", h.showUnreadEntryPage,
		"unreadEntry")

	// History pages.
	m.NameHandleFunc("/history", h.showHistoryPage, "history")
	m.NameHandleFunc("/history/entry/{entryID}", h.showReadEntryPage, "readEntry")

	// Bookmark pages.
	m.NameHandleFunc("/starred", h.showStarredPage, "starred")
	m.NameHandleFunc("/starred/entry/{entryID}", h.showStarredEntryPage,
		"starredEntry")

	// Search pages.
	m.NameHandleFunc("/search", h.showSearchPage, "search")
	m.NameHandleFunc("/search/entry/{entryID}", h.showSearchEntryPage,
		"searchEntry")

	// Feed listing pages.
	m.NameHandleFunc("/feeds", h.showFeedsPage, "feeds")
	m.NameHandleFunc("/feeds/refresh", h.refreshAllFeeds, "refreshAllFeeds")

	// Individual feed pages.
	m.NameHandleFunc("/feed/{feedID}/refresh", h.refreshFeed, "refreshFeed")
	m.NameHandleFunc("/feed/{feedID}/edit", h.showEditFeedPage, "editFeed")
	m.NameHandleFunc("/feed/{feedID}/remove", h.removeFeed, "removeFeed")
	m.NameHandleFunc("/feed/{feedID}/update", h.updateFeed, "updateFeed")
	m.NameHandleFunc("/feed/{feedID}/entries", h.showFeedEntriesPage,
		"feedEntries")
	m.NameHandleFunc("/feed/{feedID}/entries/all", h.showFeedEntriesAllPage,
		"feedEntriesAll")
	m.NameHandleFunc("/feed/{feedID}/entry/{entryID}", h.showFeedEntryPage,
		"feedEntry")
	m.NameHandleFunc("/unread/feed/{feedID}/entry/{entryID}",
		h.showUnreadFeedEntryPage, "unreadFeedEntry")
	m.NameHandleFunc("/feed/{feedID}/mark-all-as-read", h.markFeedAsRead,
		"markFeedAsRead")

	// Category pages.
	m.NameHandleFunc("/category/{categoryID}/entry/{entryID}",
		h.showCategoryEntryPage, "categoryEntry")
	m.NameHandleFunc("/unread/category/{categoryID}/entry/{entryID}",
		h.showUnreadCategoryEntryPage, "unreadCategoryEntry")
	m.NameHandleFunc("/categories", h.showCategoryListPage, "categories")
	m.NameHandleFunc("/category/create", h.showCreateCategoryPage,
		"createCategory")
	m.NameHandleFunc("/category/save", h.saveCategory, "saveCategory")
	m.NameHandleFunc("/category/{categoryID}/feeds", h.showCategoryFeedsPage,
		"categoryFeeds")
	m.NameHandleFunc("/category/{categoryID}/feed/{feedID}/remove",
		h.removeCategoryFeed, "removeCategoryFeed")
	m.NameHandleFunc("/category/{categoryID}/feeds/refresh",
		h.refreshCategoryFeedsPage, "refreshCategoryFeedsPage")
	m.NameHandleFunc("/category/{categoryID}/entries", h.showCategoryEntriesPage,
		"categoryEntries")
	m.NameHandleFunc("/category/{categoryID}/entries/refresh",
		h.refreshCategoryEntriesPage, "refreshCategoryEntriesPage")
	m.NameHandleFunc("/category/{categoryID}/entries/all",
		h.showCategoryEntriesAllPage, "categoryEntriesAll")
	m.NameHandleFunc("/category/{categoryID}/entries/starred",
		h.showCategoryEntriesStarredPage, "categoryEntriesStarred")
	m.NameHandleFunc("/category/{categoryID}/edit", h.showEditCategoryPage,
		"editCategory")
	m.NameHandleFunc("/category/{categoryID}/update", h.updateCategory,
		"updateCategory")
	m.NameHandleFunc("/category/{categoryID}/remove", h.removeCategory,
		"removeCategory")
	m.NameHandleFunc("/category/{categoryID}/mark-all-as-read",
		h.markCategoryAsRead, "markCategoryAsRead")

	// Tag pages.
	m.NameHandleFunc("/tags/{tagName}/entries/all", h.showTagEntriesAllPage,
		"tagEntriesAll")
	m.NameHandleFunc("/tags/{tagName}/entry/{entryID}", h.showTagEntryPage,
		"tagEntry")

	// Entry pages.
	m.NameHandleFunc("/entry/download/{entryID}", h.fetchContent, "fetchContent")

	// Share pages.
	m.NameHandleFunc("/entry/share/{entryID}", h.createSharedEntry, "shareEntry")
	m.NameHandleFunc("/entry/unshare/{entryID}", h.unshareEntry, "unshareEntry")
	m.NameHandleFunc("/shares", h.sharedEntries, "sharedEntries")

	// User pages.
	m.NameHandleFunc("/users", h.showUsersPage, "users")
	m.NameHandleFunc("/user/create", h.showCreateUserPage, "createUser")
	m.NameHandleFunc("/user/save", h.saveUser, "saveUser")
	m.NameHandleFunc("/users/{userID}/edit", h.showEditUserPage, "editUser")
	m.NameHandleFunc("/users/{userID}/update", h.updateUser, "updateUser")
	m.NameHandleFunc("/users/{userID}/remove", h.removeUser, "removeUser")

	// Settings pages.
	m.NameHandleFunc("GET /settings", h.showSettingsPage, "settings")
	m.NameHandleFunc("POST /settings", h.updateSettings, "updateSettings")
	m.NameHandleFunc("GET /integrations", h.showIntegrationPage, "integrations")
	m.NameHandleFunc("POST /integration", h.updateIntegration,
		"updateIntegration")
	m.NameHandleFunc("/about", h.showAboutPage, "about")

	// Session pages.
	m.NameHandleFunc("/sessions", h.showSessionsPage, "sessions")
	m.NameHandleFunc("/sessions/{sessionID}/remove", h.removeSession,
		"removeSession")

	// API Keys pages.
	m.NameHandleFunc("/keys", h.showAPIKeysPage, "apiKeys")
	m.NameHandleFunc("/keys/{keyID}/delete", h.deleteAPIKey, "deleteAPIKey")
	m.NameHandleFunc("/keys/create", h.showCreateAPIKeyPage, "createAPIKey")
	m.NameHandleFunc("/keys/save", h.saveAPIKey, "saveAPIKey")

	// OPML pages.
	m.NameHandleFunc("/export", h.exportFeeds, "export")
	m.NameHandleFunc("/import", h.showImportPage, "import")
	m.NameHandleFunc("/upload", h.uploadOPML, "uploadOPML")
	m.NameHandleFunc("/fetch", h.fetchOPML, "fetchOPML")

	// OAuth2 flow.
	if config.Opts.OAuth2Provider() != "" {
		m.NameHandleFunc("/oauth2/unlink/{provider}", h.oauth2Unlink, "oauth2Unlink")
	}

	if config.Opts.WebAuthn() {
		// WebAuthn flow
		m.NameHandleFunc("/webauthn/register/begin", h.beginRegistration,
			"webauthnRegisterBegin")
		m.NameHandleFunc("/webauthn/register/finish", h.finishRegistration,
			"webauthnRegisterFinish")
		m.NameHandleFunc("/webauthn/{credentialHandle}/rename", h.renameCredential,
			"webauthnRename")
		m.NameHandleFunc("/webauthn/{credentialHandle}/save", h.saveCredential,
			"webauthnSave")
	}
}

func robotsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, err := w.Write([]byte("User-agent: *\nDisallow: /"))
	if err != nil {
		logging.FromContext(r.Context()).
			Error(http.StatusText(http.StatusInternalServerError),
				slog.Any("error", err),
				slog.String("client_ip", request.ClientIP(r)),
				slog.GroupAttrs("request",
					slog.String("method", r.Method),
					slog.String("uri", r.RequestURI),
					slog.String("user_agent", r.UserAgent())),
				slog.GroupAttrs("response",
					slog.Int("status_code", http.StatusInternalServerError)))
	}
}
