package ui

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/session"
)

func (h *handler) switchDark(w http.ResponseWriter, r *http.Request) {
	h.switchTheme(w, r, func(u *model.User) string { return u.DarkTheme() })
}

func (h *handler) switchLight(w http.ResponseWriter, r *http.Request) {
	h.switchTheme(w, r, func(u *model.User) string { return u.LightTheme() })
}

func (h *handler) switchLightDark(w http.ResponseWriter, r *http.Request) {
	h.switchTheme(w, r, func(u *model.User) string { return u.LightDarkTheme() })
}

func (h *handler) switchTheme(w http.ResponseWriter, r *http.Request,
	themeFn func(u *model.User) string,
) {
	s := session.New(h.store, r)
	s.SetTheme(themeFn(request.User(r))).Commit(r.Context())
	html.NoContent(w, r)
}
