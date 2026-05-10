package template

import (
	"net/http"

	"miniflux.app/v2/internal/locale"
)

type Option func(*Template)

func WithLanguage(language string) Option {
	return func(t *Template) { t.printer = locale.NewPrinter(language) }
}

func WithRequest(r *http.Request) Option {
	return func(t *Template) { t.r = r }
}
