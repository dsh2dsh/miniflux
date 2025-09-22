package ui

import (
	"html/template"
	"net/http"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/mediaproxy"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/processor"
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

func (h *handler) downloadEntry(w http.ResponseWriter, r *http.Request) {
	user := request.User(r)
	g, ctx := errgroup.WithContext(r.Context())

	entryID := request.RouteInt64Param(r, "entryID")
	var entry *model.Entry
	g.Go(func() (err error) {
		entry, err = h.store.NewEntryQueryBuilder(user.ID).
			WithEntryID(entryID).
			WithoutStatus(model.EntryStatusRemoved).
			GetEntry(ctx)
		return err
	})

	feedID := request.RouteInt64Param(r, "feedID")
	var feed *model.Feed
	g.Go(func() (err error) {
		feed, err = h.store.FeedByID(ctx, user.ID, feedID)
		return err
	})

	if err := g.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	} else if entry == nil || feed == nil {
		html.NotFound(w, r)
		return
	}

	ctx = r.Context()
	err := processor.ProcessEntryWebPage(ctx, feed, entry, user)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	content := mediaproxy.RewriteDocumentWithRelativeProxyURL(
		h.router, entry.Content)

	v := view.New(h.tpl, r, nil).
		Set("entry", entry).
		Set("safeContent", template.HTML(content)).
		Set("user", request.User(r))
	html.OK(w, r, v.Render("entry_download"))
}
