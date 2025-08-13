package mux

import (
	"net/http"
	"path"
	"slices"
	"strings"
)

func New() *ServeMux {
	return &ServeMux{
		ServeMux:      http.NewServeMux(),
		namedPatterns: make(map[string]string),
	}
}

type ServeMux struct {
	*http.ServeMux

	middlewares   []MiddlewareFunc
	namedPatterns map[string]string
	pathPrefix    string
}

type MiddlewareFunc func(next http.Handler) http.Handler

var _ http.Handler = (*ServeMux)(nil)

func (self *ServeMux) Group(funcs ...func(m *ServeMux)) *ServeMux {
	g := *self
	g.middlewares = slices.Clone(self.middlewares)
	for _, fn := range funcs {
		fn(&g)
	}
	return &g
}

func (self *ServeMux) Handle(pattern string, handler http.Handler) *ServeMux {
	self.mux().Handle(pattern, self.wrapped(handler))
	return self
}

func (self *ServeMux) mux() *http.ServeMux { return self.ServeMux }

func (self *ServeMux) wrapped(handler http.Handler) http.Handler {
	for _, m := range slices.Backward(self.middlewares) {
		handler = m(handler)
	}
	return handler
}

func (self *ServeMux) HandleFunc(pattern string,
	handler func(http.ResponseWriter, *http.Request),
) *ServeMux {
	return self.Handle(pattern, http.HandlerFunc(handler))
}

func (self *ServeMux) NameHandle(pattern string, handler http.Handler,
	name string,
) *ServeMux {
	pathPattern := self.joinPathPrefix(pattern)
	self.namedPatterns[name] = pathPattern
	return self.Handle(pattern, handler)
}

func (self *ServeMux) joinPathPrefix(pattern string) string {
	pattern = strings.TrimSuffix(removeMethod(pattern), "{$}")
	if self.pathPrefix == "" {
		return pattern
	}

	prefixed := path.Join(self.pathPrefix, pattern)
	if strings.HasSuffix(pattern, "/") && !strings.HasSuffix(prefixed, "/") {
		return prefixed + "/"
	}
	return prefixed
}

func removeMethod(pattern string) string {
	before, after, found := strings.Cut(pattern, " ")
	if !found {
		return before
	}
	return strings.TrimLeft(after, " ")
}

func (self *ServeMux) NameHandleFunc(pattern string,
	handler func(http.ResponseWriter, *http.Request), name string,
) *ServeMux {
	return self.NameHandle(pattern, http.HandlerFunc(handler), name)
}

func (self *ServeMux) NamedPath(name string, pairs ...string) string {
	pattern, ok := self.namedPatterns[name]
	if !ok {
		return ""
	} else if len(pairs) < 2 {
		return pattern
	}

	for i := 0; i < len(pairs); i += 2 {
		k := "{" + pairs[i] + "}"
		pattern = strings.Replace(pattern, k, pairs[i+1], 1)
	}
	return pattern
}

func (self *ServeMux) PrefixGroup(prefix string, funcs ...func(m *ServeMux),
) *ServeMux {
	if prefix == "" {
		return self.Group(funcs...)
	}

	pattern := prefix
	if !strings.HasSuffix(pattern, "/") {
		pattern += "/"
	}
	mux := http.NewServeMux()
	self.Handle(pattern, http.StripPrefix(prefix, mux))

	g := *self
	g.ServeMux = mux
	g.middlewares = nil
	g.pathPrefix = path.Join(self.pathPrefix, prefix)

	for _, fn := range funcs {
		fn(&g)
	}
	return &g
}

func (self *ServeMux) Use(m ...MiddlewareFunc) *ServeMux {
	self.middlewares = append(self.middlewares, m...)
	return self
}
