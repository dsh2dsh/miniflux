package ui

import (
	"html/template"
	"net/http"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/mediaproxy"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/view"
)

func (h *handler) inlineEntry(w http.ResponseWriter, r *http.Request) {
	entry, err := h.store.NewEntryQueryBuilder(request.UserID(r)).
		WithEntryID(request.RouteInt64Param(r, "entryID")).
		WithoutStatus(model.EntryStatusRemoved).
		GetEntry(r.Context())
	if err != nil {
		html.ServerError(w, r, err)
		return
	} else if entry == nil {
		html.NotFound(w, r)
		return
	}

	content := mediaproxy.RewriteDocumentWithRelativeProxyURL(
		h.router, entry.Content)
	entry.Enclosures().ProxifyEnclosureURL(h.router, config.Opts.MediaProxyMode(),
		config.Opts.MediaProxyResourceTypes())

	v := view.New(h.tpl, r, nil).
		Set("entry", entry).
		Set("safeContent", template.HTML(content)).
		Set("user", request.User(r))
	html.OK(w, r, v.Render("entry_inline"))
}
