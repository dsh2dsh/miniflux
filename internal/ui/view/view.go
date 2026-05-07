// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package view // import "miniflux.app/v2/internal/ui/view"

import (
	"net/http"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/template"
	"miniflux.app/v2/internal/ui/session"
)

// View wraps template argument building.
type View struct {
	tpl    *template.Engine
	r      *http.Request
	params map[string]any

	session *session.Session
}

// New returns a new view with default parameters.
func New(tpl *template.Engine, r *http.Request) *View {
	theme := request.UserTheme(r)
	v := &View{
		tpl: tpl,
		r:   r,
		params: map[string]any{
			"menu":            "",
			"theme":           theme,
			"language":        request.UserLanguage(r),
			"requestURI":      request.RequestURI(r),
			"webAuthnEnabled": config.WebAuthn(),
		},
	}

	if sess := session.FromContext(r.Context()); sess != nil {
		v.session = sess
		v.Set("flashMessage", sess.FlashMessage(request.FlashMessage(r))).
			Set("flashErrorMessage",
				sess.FlashErrorMessage(request.FlashErrorMessage(r)))
	}
	return v
}

func (self *View) WithEntries(entries model.Entries) *View {
	self.params["entries"] = template.Entries(entries)
	self.params["numOfEntries"] = len(entries)
	if n := len(entries); n != 0 {
		self.params["lastEntry"] = entries[n-1]
	}
	return self
}

func (self *View) WithEntry(entry *model.Entry) *View {
	self.params["entry"] = template.NewEntry(entry)
	return self
}

// Set adds a new template argument.
func (self *View) Set(param string, value any) *View {
	self.params[param] = value
	return self
}

// Render executes the template with arguments.
func (self *View) Render(template string) []byte {
	return self.tpl.Render(template+".html", self.params)
}
