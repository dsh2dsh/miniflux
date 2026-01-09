package ui

import (
	"context"
	"net/http"
	"net/url"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
)

func (h *handler) showAuthorEntries(w http.ResponseWriter, r *http.Request) {
	h.renderAuthorEntries(w, r, true)
}

func (h *handler) showAuthorEntriesAll(w http.ResponseWriter, r *http.Request) {
	h.renderAuthorEntries(w, r, false)
}

func (h *handler) renderAuthorEntries(w http.ResponseWriter, r *http.Request,
	unread bool,
) {
	authorName, err := url.PathUnescape(request.RouteStringParam(r, "authorName"))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	v := h.View(r).WithSaveEntry()
	user := v.User()

	feedID := request.RouteInt64Param(r, "feedID")
	var feed *model.Feed
	v.Go(func(ctx context.Context) (err error) {
		feed, err = h.store.FeedByID(ctx, v.UserID(), feedID)
		return err
	})

	offset := request.QueryIntParam(r, "offset", 0)
	query := h.store.NewEntryQueryBuilder(v.UserID()).
		WithFeedID(feedID).
		WithAuthor(authorName).
		WithSorting("status", "asc").
		WithSorting(user.EntryOrder, user.EntryDirection).
		WithSorting("id", user.EntryDirection).
		WithOffset(offset).
		WithLimit(user.EntriesPerPage)

	if unread {
		query.WithStatus(model.EntryStatusUnread)
	} else {
		query.WithoutStatus(model.EntryStatusRemoved)
	}

	entries, count, err := v.WaitEntriesCount(query)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("authorName", authorName).
		Set("feed", feed).
		Set("total", count).
		Set("entries", entries).
		Set("lastEntry", lastEntry(entries)).
		Set("pagination", getPagination(
			route.Path(h.router, "authorEntries", "authorName",
				url.PathEscape(authorName)),
			count, offset, user.EntriesPerPage)).
		Set("showOnlyUnreadEntries", unread)
	html.OK(w, r, v.Render("author_entries"))
}
