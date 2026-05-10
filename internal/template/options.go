package template

import "miniflux.app/v2/internal/locale"

type Option func(*Template)

func WithLanguage(language string) Option {
	return func(t *Template) {
		t.printer = locale.NewPrinter(language)
	}
}
