package ui

import (
	"errors"
	"html/template"
	"net/http"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/mediaproxy"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/reader/processor"
	"miniflux.app/v2/internal/reader/sanitizer"
	"miniflux.app/v2/internal/sites"
	"miniflux.app/v2/internal/ui/view"
)

func (h *handler) inlineEntry(w http.ResponseWriter, r *http.Request) {
	entry, err := h.store.NewEntryQueryBuilder(request.UserID(r)).
		WithEntryID(request.RouteInt64Param(r, "entryID")).
		WithoutStatus(model.EntryStatusRemoved).
		GetEntry(r.Context())
	if err != nil {
		response.ServerError(w, r, err)
		return
	} else if entry == nil {
		response.NotFound(w, r)
		return
	}

	user := request.User(r)

	var contentError string
	content := entry.Content
	b, err := sites.Render(r.Context(), user, entry, h.tpl)
	if err != nil {
		contentError = err.Error()
	} else if len(b) != 0 {
		content = string(b)
	}

	content = mediaproxy.RewriteDocumentWithRelativeProxyURL(h.router, content)
	mediaproxy.ProxifyEnclosures(h.router, entry.Enclosures())

	view.New(h.tpl, r).WithUser(user).WithEntry(entry).
		Set("showOnlyUnreadEntries", request.QueryBoolParam(r, "unread", false)).
		Set("inlined", true).
		Set("downloaded", false).
		Set("contentError", contentError).
		Set("safeContent", template.HTML(content)).
		HTML(w, r, "inline_entry.html", "item_inner.html")
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
		response.ServerError(w, r, err)
		return
	} else if entry == nil || feed == nil {
		response.NotFound(w, r)
		return
	}

	origErr := processor.ProcessEntryWebPage(r.Context(), feed, entry, user,
		sanitizer.WithRewriteURL(mediaproxy.New(h.router).RewriteURL))
	contentError, err := h.unexpectedContent(origErr, entry)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}

	view.New(h.tpl, r).WithUser(user).WithEntry(entry).
		Set("showOnlyUnreadEntries", request.QueryBoolParam(r, "unread", false)).
		Set("inlined", true).
		Set("downloaded", true).
		Set("contentError", contentError).
		Set("safeContent", template.HTML(entry.Content)).
		HTML(w, r, "inline_entry.html", "item_inner.html")
}

func (h *handler) unexpectedContent(err error, entry *model.Entry,
) (string, error) {
	badStatus, ok := errors.AsType[*fetcher.ErrBadStatus](err)
	if !ok {
		return "", err
	}

	if len(badStatus.Body) == 0 {
		return badStatus.String(), nil
	}

	u, err := entry.ParsedURL()
	if err != nil {
		return "", err
	}

	entry.Content = sanitizer.SanitizeContent(string(badStatus.Body), u,
		sanitizer.WithRewriteURL(mediaproxy.New(h.router).RewriteURL))
	return badStatus.String(), nil
}
