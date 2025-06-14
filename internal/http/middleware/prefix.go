package middleware

import (
	"net/http"
	"slices"
	"strings"
)

func NewPathPrefix() *PathPrefix { return new(PathPrefix) }

type PathPrefix struct {
	prefixes []prefixHandler
}

type prefixHandler struct {
	prefix      string
	middlewares []MiddlewareFunc
}

func (self *prefixHandler) Match(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, self.prefix)
}

func (self *prefixHandler) Middleware(next http.Handler) http.Handler {
	for _, m := range slices.Backward(self.middlewares) {
		next = m(next)
	}
	return next
}

func (self *PathPrefix) WithPrefix(prefix string, m ...MiddlewareFunc,
) *PathPrefix {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	self.prefixes = append(self.prefixes, prefixHandler{
		prefix:      prefix,
		middlewares: m,
	})
	return self
}

func (self *PathPrefix) WithDefault(m ...MiddlewareFunc) *PathPrefix {
	self.prefixes = append(self.prefixes, prefixHandler{middlewares: m})
	return self
}

func (self *PathPrefix) Middleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		for i := range self.prefixes {
			prefix := &self.prefixes[i]
			if prefix.Match(r) {
				prefix.Middleware(next).ServeHTTP(w, r)
				return
			}
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
