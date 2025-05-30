package middleware

import (
	"net/http"
	"strings"

	"miniflux.app/v2/internal/http/request"
)

func WithPublicRoutes(m map[string]struct{}) MiddlewareFunc {
	fn := func(next http.Handler) http.Handler {
		return &PublicRoutes{m: m, next: next}
	}
	return fn
}

type PublicRoutes struct {
	m    map[string]struct{}
	next http.Handler
}

func (self *PublicRoutes) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if p == "/" {
		self.public(w, r)
		return
	}

	if _, ok := self.m[p]; ok {
		self.public(w, r)
		return
	}

	for s := range self.m {
		if strings.HasPrefix(p, s) {
			self.public(w, r)
			return
		}
	}
	self.next.ServeHTTP(w, r)
}

func (self *PublicRoutes) public(w http.ResponseWriter, r *http.Request) {
	ctx := request.WithPublic(r.Context())
	self.next.ServeHTTP(w, r.WithContext(ctx))
}
