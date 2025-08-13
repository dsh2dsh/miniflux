// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package view // import "miniflux.app/v2/internal/ui/view"

import (
	"net/http"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
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
func New(tpl *template.Engine, r *http.Request, sess *session.Session) *View {
	theme := request.UserTheme(r)
	v := &View{
		tpl: tpl,
		r:   r,
		params: map[string]any{
			"menu":            "",
			"theme":           theme,
			"language":        request.UserLanguage(r),
			"webAuthnEnabled": config.Opts.WebAuthn(),
		},
	}

	if sess != nil {
		v.session = sess
		v.Set("flashMessage", sess.FlashMessage(request.FlashMessage(r))).
			Set("flashErrorMessage",
				sess.FlashErrorMessage(request.FlashErrorMessage(r)))
	}
	return v
}

// Set adds a new template argument.
func (v *View) Set(param string, value any) *View {
	v.params[param] = value
	return v
}

// Render executes the template with arguments.
func (v *View) Render(template string) []byte {
	if v.session != nil {
		v.session.Commit(v.r.Context())
	}
	return v.tpl.Render(template+".html", v.params)
}
