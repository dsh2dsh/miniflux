package ui

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/ui/session"
)

func (h *handler) switchDark(w http.ResponseWriter, r *http.Request) {
	h.switchTheme(w, r, true)
}

func (h *handler) switchLight(w http.ResponseWriter, r *http.Request) {
	h.switchTheme(w, r, false)
}

func (h *handler) switchTheme(w http.ResponseWriter, r *http.Request,
	toDark bool,
) {
	s := session.New(h.store, r)
	user := request.User(r)
	if toDark {
		s.SetTheme(user.DarkTheme())
	} else {
		s.SetTheme(user.LightTheme())
	}
	s.Commit(r.Context())
	html.NoContent(w, r)
}
