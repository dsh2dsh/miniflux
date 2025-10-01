package handler

import "miniflux.app/v2/internal/template"

type Option func(r *Refresh)

func WithForceUpdate(v bool) Option {
	return func(self *Refresh) { self.force = v }
}

func WithTemplates(templates *template.Engine) Option {
	return func(self *Refresh) { self.templates = templates }
}
