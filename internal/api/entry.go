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
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/integration"
	"miniflux.app/v2/internal/mediaproxy"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/processor"
	"miniflux.app/v2/internal/reader/readingtime"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) getEntryFromBuilder(w http.ResponseWriter, r *http.Request,
	b *storage.EntryQueryBuilder,
) {
	entry, err := b.GetEntry(r.Context())
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if entry == nil {
		json.NotFound(w, r)
		return
	}

	entry.Content = mediaproxy.RewriteDocumentWithAbsoluteProxyURL(h.router,
		entry.Content)
	entry.Enclosures.ProxifyEnclosureURL(h.router)
	json.OK(w, r, entry)
}

func (h *handler) getFeedEntry(w http.ResponseWriter, r *http.Request) {
	feedID := request.RouteInt64Param(r, "feedID")
	id := request.RouteInt64Param(r, "entryID")
	h.getEntryFromBuilder(w, r, h.store.NewEntryQueryBuilder(request.UserID(r)).
		WithFeedID(feedID).
		WithEntryID(id))
}

func (h *handler) getCategoryEntry(w http.ResponseWriter, r *http.Request) {
	categoryID := request.RouteInt64Param(r, "categoryID")
	id := request.RouteInt64Param(r, "entryID")
	h.getEntryFromBuilder(w, r, h.store.NewEntryQueryBuilder(request.UserID(r)).
		WithCategoryID(categoryID).
		WithEntryID(id))
}

func (h *handler) getEntry(w http.ResponseWriter, r *http.Request) {
	id := request.RouteInt64Param(r, "entryID")
	h.getEntryFromBuilder(w, r, h.store.NewEntryQueryBuilder(request.UserID(r)).
		WithEntryID(id))
}

func (h *handler) getFeedEntries(w http.ResponseWriter, r *http.Request) {
	feedID := request.RouteInt64Param(r, "feedID")
	h.findEntries(w, r, feedID, 0)
}

func (h *handler) getCategoryEntries(w http.ResponseWriter, r *http.Request) {
	id := request.RouteInt64Param(r, "categoryID")
	h.findEntries(w, r, 0, id)
}

func (h *handler) getEntries(w http.ResponseWriter, r *http.Request) {
	h.findEntries(w, r, 0, 0)
}

func (h *handler) findEntries(w http.ResponseWriter, r *http.Request, feedID,
	categoryID int64,
) {
	statuses := request.QueryStringParamList(r, "status")
	for _, status := range statuses {
		if err := validator.ValidateEntryStatus(status); err != nil {
			json.BadRequest(w, r, err)
			return
		}
	}

	order := request.QueryStringParam(r, "order", model.DefaultSortingOrder)
	if err := validator.ValidateEntryOrder(order); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	direction := request.QueryStringParam(r, "direction",
		model.DefaultSortingDirection)
	if err := validator.ValidateDirection(direction); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	limit := request.QueryIntParam(r, "limit", 100)
	offset := request.QueryIntParam(r, "offset", 0)
	if err := validator.ValidateRange(offset, limit); err != nil {
		json.BadRequest(w, r, err)
		return
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
		WithEnclosures()

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
		return
	})

	var count int
	g.Go(func() (err error) {
		count, err = builder.CountEntries(r.Context())
		return
	})

	if err := g.Wait(); err != nil {
		if errors.Is(err, errInvalid) {
			json.BadRequest(w, r, err)
		} else {
			json.ServerError(w, r, err)
		}
		return
	}

	for i := range entries {
		entries[i].Content = mediaproxy.RewriteDocumentWithAbsoluteProxyURL(
			h.router, entries[i].Content)
	}
	json.OK(w, r, &entriesResponse{Total: count, Entries: entries})
}

func (h *handler) setEntryStatus(w http.ResponseWriter, r *http.Request) {
	var updateRequest model.EntriesStatusUpdateRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	err := validator.ValidateEntriesStatusUpdateRequest(&updateRequest)
	if err != nil {
		json.BadRequest(w, r, err)
		return
	}

	err = h.store.SetEntriesStatus(r.Context(), request.UserID(r),
		updateRequest.EntryIDs, updateRequest.Status)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.NoContent(w, r)
}

func (h *handler) toggleBookmark(w http.ResponseWriter, r *http.Request) {
	id := request.RouteInt64Param(r, "entryID")
	err := h.store.ToggleBookmark(r.Context(), request.UserID(r), id)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.NoContent(w, r)
}

func (h *handler) saveEntry(w http.ResponseWriter, r *http.Request) {
	user := request.User(r)
	if !user.HasSaveEntry() {
		json.BadRequest(w, r, errors.New("no third-party integration enabled"))
	}

	id := request.RouteInt64Param(r, "entryID")
	userID := request.UserID(r)
	entry, err := h.store.NewEntryQueryBuilder(userID).
		WithEntryID(id).
		WithoutStatus(model.EntryStatusRemoved).
		GetEntry(r.Context())
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if entry == nil {
		json.NotFound(w, r)
		return
	}

	integration.SendEntry(entry, user)
	json.Accepted(w, r)
}

func (h *handler) updateEntry(w http.ResponseWriter, r *http.Request) {
	var updateRequest model.EntryUpdateRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	if err := validator.ValidateEntryModification(&updateRequest); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	userID := request.UserID(r)
	id := request.RouteInt64Param(r, "entryID")
	entryBuilder := h.store.NewEntryQueryBuilder(userID).
		WithEntryID(id).
		WithoutStatus(model.EntryStatusRemoved)

	ctx := r.Context()
	entry, err := entryBuilder.GetEntry(ctx)
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if entry == nil {
		json.NotFound(w, r)
		return
	}

	updateRequest.Patch(entry)
	user := request.User(r)
	if user.ShowReadingTime {
		entry.ReadingTime = readingtime.EstimateReadingTime(entry.Content,
			user.DefaultReadingSpeed, user.CJKReadingSpeed)
	}

	err = h.store.UpdateEntryTitleAndContent(ctx, entry)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.Created(w, r, entry)
}

func (h *handler) fetchContent(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	entryID := request.RouteInt64Param(r, "entryID")
	builder := h.store.NewEntryQueryBuilder(userID).
		WithEntryID(entryID).
		WithoutStatus(model.EntryStatusRemoved)
	ctx := r.Context()

	entry, err := builder.GetEntry(ctx)
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if entry == nil {
		json.NotFound(w, r)
		return
	}

	feed, err := h.store.FeedByID(ctx, userID, entry.FeedID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if feed == nil {
		json.NotFound(w, r)
		return
	}

	user := request.User(r)
	if err := processor.ProcessEntryWebPage(ctx, feed, entry, user); err != nil {
		json.ServerError(w, r, err)
		return
	}

	shouldUpdateContent := request.QueryBoolParam(r, "update_content", false)
	if shouldUpdateContent {
		err := h.store.UpdateEntryTitleAndContent(ctx, entry)
		if err != nil {
			json.ServerError(w, r, err)
			return
		}

		json.OK(w, r, map[string]any{
			"content": mediaproxy.RewriteDocumentWithRelativeProxyURL(h.router,
				entry.Content),
			"reading_time": entry.ReadingTime,
		})
		return
	}
	json.OK(w, r, map[string]string{"content": entry.Content})
}

func (h *handler) flushHistory(w http.ResponseWriter, r *http.Request) {
	err := h.store.FlushHistory(r.Context(), request.UserID(r))
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.Accepted(w, r)
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
