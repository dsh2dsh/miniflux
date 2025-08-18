package ui

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
)

func (h *handler) userStylesheet(w http.ResponseWriter, r *http.Request) {
	user := request.User(r)
	response.New(w, r).
		WithLongCaching().
		WithHeader("Content-Type", "text/css; charset=utf-8").
		WithBody(user.Stylesheet).
		Write()
}

func (h *handler) userJavascript(w http.ResponseWriter, r *http.Request) {
	user := request.User(r)
	response.New(w, r).
		WithLongCaching().
		WithHeader("Content-Type", "text/javascript; charset=utf-8").
		WithBody(user.CustomJS).
		Write()
}
