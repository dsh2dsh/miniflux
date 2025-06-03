package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"miniflux.app/v2/internal/http/cookie"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/storage"
)

func WithUserSession(s *storage.Storage, m map[string]struct{}) MiddlewareFunc {
	fn := func(next http.Handler) http.Handler {
		return &UserSession{store: s, next: next, publicRoutes: m}
	}
	return fn
}

type UserSession struct {
	store *storage.Storage
	next  http.Handler

	publicRoutes map[string]struct{}
}

func (self *UserSession) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := request.CookieValue(r, cookie.CookieAppSessionID)
	if id == "" || self.skipPublic(r) {
		self.next.ServeHTTP(w, r)
		return
	}

	ctx := r.Context()
	user, sess, err := self.store.UserSession(ctx, id)
	if err != nil {
		html.ServerError(w, r, err)
	}

	if sess == nil {
		logging.FromContext(ctx).Debug("lost session detected",
			slog.String("id", id))
		self.next.ServeHTTP(w, r)
		return
	}

	ctx = request.WithUserSession(ctx, user, sess)
	self.next.ServeHTTP(w, r.WithContext(ctx))
}

func (self *UserSession) skipPublic(r *http.Request) bool {
	if len(self.publicRoutes) == 0 || !request.Public(r) {
		return false
	}

	p := r.URL.Path
	if p == "/" {
		return false
	}

	for s := range self.publicRoutes {
		if strings.HasPrefix(p, s) {
			return false
		}
	}
	return true
}
