// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	json_parser "encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/integration"
	"miniflux.app/v2/internal/mediaproxy"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/processor"
	"miniflux.app/v2/internal/reader/readingtime"
	"miniflux.app/v2/internal/reader/sanitizer"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) getEntryFromBuilder(w http.ResponseWriter, r *http.Request,
	b *storage.EntryQueryBuilder,
) (*model.Entry, error) {
	entry, err := b.GetEntry(r.Context())
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, response.ErrNotFound
	}

	entry.Content = mediaproxy.RewriteDocumentWithAbsoluteProxyURL(h.router,
		entry.Content)
	mediaproxy.ProxifyEnclosures(h.router, entry.Enclosures())
	return entry, nil
}

func (h *handler) getFeedEntry(w http.ResponseWriter, r *http.Request,
) (*model.Entry, error) {
	feedID := request.RouteInt64Param(r, "feedID")
	id := request.RouteInt64Param(r, "entryID")
	return h.getEntryFromBuilder(w, r,
		h.store.NewEntryQueryBuilder(request.UserID(r)).
			WithFeedID(feedID).
			WithEntryID(id).
			WithoutStatus(model.EntryStatusRemoved))
}

func (h *handler) getCategoryEntry(w http.ResponseWriter, r *http.Request,
) (*model.Entry, error) {
	categoryID := request.RouteInt64Param(r, "categoryID")
	id := request.RouteInt64Param(r, "entryID")
	return h.getEntryFromBuilder(w, r,
		h.store.NewEntryQueryBuilder(request.UserID(r)).
			WithCategoryID(categoryID).
			WithEntryID(id).
			WithoutStatus(model.EntryStatusRemoved))
}

func (h *handler) getEntry(w http.ResponseWriter, r *http.Request,
) (*model.Entry, error) {
	id := request.RouteInt64Param(r, "entryID")
	return h.getEntryFromBuilder(w, r,
		h.store.NewEntryQueryBuilder(request.UserID(r)).
			WithEntryID(id).
			WithoutStatus(model.EntryStatusRemoved))
}

func (h *handler) getFeedEntries(w http.ResponseWriter, r *http.Request,
) (*entriesResponse, error) {
	feedID := request.RouteInt64Param(r, "feedID")
	return h.findEntries(w, r, feedID, 0)
}

func (h *handler) getCategoryEntries(w http.ResponseWriter, r *http.Request,
) (*entriesResponse, error) {
	id := request.RouteInt64Param(r, "categoryID")
	return h.findEntries(w, r, 0, id)
}

func (h *handler) getEntries(w http.ResponseWriter, r *http.Request,
) (*entriesResponse, error) {
	return h.findEntries(w, r, 0, 0)
}

func (h *handler) findEntries(_ http.ResponseWriter, r *http.Request, feedID,
	categoryID int64,
) (*entriesResponse, error) {
	statuses := request.QueryStringParamList(r, "status")
	for _, status := range statuses {
		if err := validator.ValidateEntryStatus(status); err != nil {
			return nil, response.WrapBadRequest(err)
		}
	}

	order := request.QueryStringParam(r, "order", model.DefaultSortingOrder)
	if err := validator.ValidateEntryOrder(order); err != nil {
		return nil, response.WrapBadRequest(err)
	}

	direction := request.QueryStringParam(r, "direction",
		model.DefaultSortingDirection)
	if err := validator.ValidateDirection(direction); err != nil {
		return nil, response.WrapBadRequest(err)
	}

	limit := request.QueryIntParam(r, "limit", 100)
	offset := request.QueryIntParam(r, "offset", 0)
	if err := validator.ValidateRange(offset, limit); err != nil {
		return nil, response.WrapBadRequest(err)
	}

	g, ctx := errgroup.WithContext(r.Context())
	errInvalid := errors.New("invalid")

	userID := request.UserID(r)
	categoryID = request.QueryInt64Param(r, "category_id", categoryID)
	if categoryID > 0 {
		g.Go(func() error {
			if !h.store.CategoryIDExists(ctx, userID, categoryID) {
				return fmt.Errorf("%w category ID", errInvalid)
			}
			return nil
		})
	}

	feedID = request.QueryInt64Param(r, "feed_id", feedID)
	if feedID > 0 {
		g.Go(func() error {
			if !h.store.FeedExists(ctx, userID, feedID) {
				return fmt.Errorf("%w feed ID", errInvalid)
			}
			return nil
		})
	}

	tags := request.QueryStringParamList(r, "tags")
	builder := h.store.NewEntryQueryBuilder(userID).
		WithFeedID(feedID).
		WithCategoryID(categoryID).
		WithStatuses(statuses).
		WithSorting(order, direction).
		WithOffset(offset).
		WithLimit(limit).
		WithTags(tags).
		WithContent().
		WithoutStatus(model.EntryStatusRemoved)

	if request.HasQueryParam(r, "globally_visible") {
		globallyVisible := request.QueryBoolParam(r, "globally_visible", true)
		if globallyVisible {
			builder.WithGloballyVisible()
		}
	}
	configureFilters(builder, r)

	var entries model.Entries
	g.Go(func() (err error) {
		entries, err = builder.GetEntries(r.Context())
		return err
	})

	var count int
	g.Go(func() (err error) {
		count, err = builder.CountEntries(r.Context())
		return err
	})

	if err := g.Wait(); err != nil {
		if errors.Is(err, errInvalid) {
			return nil, response.WrapBadRequest(err)
		}
		return nil, err //nolint:wrapcheck // from our package inside Go
	}

	for i := range entries {
		entries[i].Content = mediaproxy.RewriteDocumentWithAbsoluteProxyURL(
			h.router, entries[i].Content)
	}
	return &entriesResponse{Total: count, Entries: entries}, nil
}

func (h *handler) setEntryStatus(w http.ResponseWriter, r *http.Request) error {
	var updateRequest model.EntriesStatusUpdateRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		return response.WrapBadRequest(err)
	}

	err := validator.ValidateEntriesStatusUpdateRequest(&updateRequest)
	if err != nil {
		return response.WrapBadRequest(err)
	}

	err = h.store.SetEntriesStatus(r.Context(), request.UserID(r),
		updateRequest.EntryIDs, updateRequest.Status)
	if err != nil {
		return err
	}
	return nil
}

func (h *handler) toggleBookmark(w http.ResponseWriter, r *http.Request) error {
	id := request.RouteInt64Param(r, "entryID")
	err := h.store.ToggleBookmark(r.Context(), request.UserID(r), id)
	if err != nil {
		return err
	}
	return nil
}

func (h *handler) saveEntry(w http.ResponseWriter, r *http.Request) error {
	user := request.User(r)
	if !user.HasSaveEntry() {
		return response.WrapBadRequest(errors.New(
			"no third-party integration enabled"))
	}

	id := request.RouteInt64Param(r, "entryID")
	userID := request.UserID(r)
	entry, err := h.store.NewEntryQueryBuilder(userID).
		WithEntryID(id).
		WithoutStatus(model.EntryStatusRemoved).
		GetEntry(r.Context())
	if err != nil {
		return err
	} else if entry == nil {
		return response.ErrNotFound
	}

	integration.SendEntry(r.Context(), entry, user)
	return nil
}

func (h *handler) updateEntry(w http.ResponseWriter, r *http.Request,
) (*model.Entry, error) {
	var updateRequest model.EntryUpdateRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		return nil, response.WrapBadRequest(err)
	}

	if err := validator.ValidateEntryModification(&updateRequest); err != nil {
		return nil, response.WrapBadRequest(err)
	}

	userID := request.UserID(r)
	id := request.RouteInt64Param(r, "entryID")
	entryBuilder := h.store.NewEntryQueryBuilder(userID).
		WithEntryID(id).
		WithoutStatus(model.EntryStatusRemoved)

	ctx := r.Context()
	entry, err := entryBuilder.GetEntry(ctx)
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, response.ErrNotFound
	}

	updateRequest.Patch(entry)
	if err := processor.UpdateEntry(request.User(r), entry); err != nil {
		return nil, err
	}

	err = h.store.UpdateEntryTitleAndContent(ctx, entry)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (h *handler) importEntries(w http.ResponseWriter, r *http.Request,
) (*entriesResponse, error) {
	var importReq model.ImportEntries
	if err := json_parser.NewDecoder(r.Body).Decode(&importReq); err != nil {
		return nil, response.WrapBadRequest(err)
	}

	if importReq.FeedURL == "" {
		return nil, response.WrapBadRequest(errors.New("empty feed URL"))
	} else if len(importReq.Entries) == 0 {
		return nil, response.WrapBadRequest(errors.New("empty entries list"))
	}

	ctx := r.Context()
	user := request.User(r)

	feed, err := h.store.FeedByURL(ctx, user.ID, importReq.FeedURL)
	if err != nil {
		return nil, err
	} else if feed == nil {
		return nil, response.WrapBadRequest(errors.New("feed does not exists"))
	}

	entries := make(model.Entries, len(importReq.Entries))
	for i := range importReq.Entries {
		importEntry := importReq.Entries[i]
		if importEntry.URL == "" {
			return nil, response.WrapBadRequest(errors.New("url is required"))
		}

		if importEntry.Status == "" {
			importEntry.Status = model.EntryStatusRead
		}
		if err := validator.ValidateEntryStatus(importEntry.Status); err != nil {
			return nil, response.WrapBadRequest(err)
		}

		entry := model.NewEntryFrom(importEntry)
		if user.ShowReadingTime {
			entry.ReadingTime = readingtime.EstimateReadingTime(entry.Content,
				user.DefaultReadingSpeed, user.CJKReadingSpeed)
		}
		entries[i] = entry
	}

	_, err = h.store.StoreFeedEntries(ctx, user.ID, feed.ID, entries, true)
	if err != nil {
		return nil, err
	}
	return &entriesResponse{Total: len(entries), Entries: entries}, nil
}

func (h *handler) fetchContent(w http.ResponseWriter, r *http.Request,
) (*entryContentResponse, error) {
	userID := request.UserID(r)
	entryID := request.RouteInt64Param(r, "entryID")
	builder := h.store.NewEntryQueryBuilder(userID).
		WithEntryID(entryID).
		WithoutStatus(model.EntryStatusRemoved)
	ctx := r.Context()

	entry, err := builder.GetEntry(ctx)
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, response.ErrNotFound
	}

	feed, err := h.store.FeedByID(ctx, userID, entry.FeedID)
	if err != nil {
		return nil, err
	} else if feed == nil {
		return nil, response.ErrNotFound
	}

	user := request.User(r)
	if request.QueryBoolParam(r, "update_content", false) {
		err := processor.ProcessEntryWebPage(ctx, feed, entry, user)
		if err != nil {
			return nil, err
		}

		err = h.store.UpdateEntryTitleAndContent(ctx, entry)
		if err != nil {
			return nil, err
		}
		entry.Content = mediaproxy.RewriteDocumentWithAbsoluteProxyURL(h.router,
			entry.Content)
	} else {
		err := processor.ProcessEntryWebPage(ctx, feed, entry, user,
			sanitizer.WithRewriteURL(
				mediaproxy.New(h.router).WithAbsoluteProxy().RewriteURL))
		if err != nil {
			return nil, err
		}
	}

	return &entryContentResponse{
		Content:     entry.Content,
		ReadingTime: entry.ReadingTime,
	}, nil
}

func (h *handler) flushHistory(w http.ResponseWriter, r *http.Request) error {
	err := h.store.FlushHistory(r.Context(), request.UserID(r))
	if err != nil {
		return err
	}
	return nil
}

func configureFilters(builder *storage.EntryQueryBuilder, r *http.Request) {
	beforeEntryID := request.QueryInt64Param(r, "before_entry_id", 0)
	if beforeEntryID > 0 {
		builder.BeforeEntryID(beforeEntryID)
	}

	afterEntryID := request.QueryInt64Param(r, "after_entry_id", 0)
	if afterEntryID > 0 {
		builder.AfterEntryID(afterEntryID)
	}

	beforePublished := request.QueryInt64Param(r, "before", 0)
	if beforePublished > 0 {
		builder.BeforePublishedDate(time.Unix(beforePublished, 0))
	}

	afterPublished := request.QueryInt64Param(r, "after", 0)
	if afterPublished > 0 {
		builder.AfterPublishedDate(time.Unix(afterPublished, 0))
	}

	beforePublished = request.QueryInt64Param(r, "published_before", 0)
	if beforePublished > 0 {
		builder.BeforePublishedDate(time.Unix(beforePublished, 0))
	}

	afterPublished = request.QueryInt64Param(r, "published_after", 0)
	if afterPublished > 0 {
		builder.AfterPublishedDate(time.Unix(afterPublished, 0))
	}

	beforeChanged := request.QueryInt64Param(r, "changed_before", 0)
	if beforeChanged > 0 {
		builder.BeforeChangedDate(time.Unix(beforeChanged, 0))
	}

	afterChanged := request.QueryInt64Param(r, "changed_after", 0)
	if afterChanged > 0 {
		builder.AfterChangedDate(time.Unix(afterChanged, 0))
	}

	categoryID := request.QueryInt64Param(r, "category_id", 0)
	if categoryID > 0 {
		builder.WithCategoryID(categoryID)
	}

	if request.HasQueryParam(r, "starred") {
		starred, err := strconv.ParseBool(r.URL.Query().Get("starred"))
		if err == nil {
			builder.WithStarred(starred)
		}
	}

	searchQuery := request.QueryStringParam(r, "search", "")
	if searchQuery != "" {
		builder.WithSearchQuery(searchQuery)
	}
}
