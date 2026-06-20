// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	json_parser "encoding/json"
	"errors"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/integration"
	"miniflux.app/v2/internal/mediaproxy"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/processor"
	"miniflux.app/v2/internal/reader/sanitizer"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) getEntryFromBuilder(_ http.ResponseWriter, r *http.Request,
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

func (h *handler) setEntryStatus(w http.ResponseWriter, r *http.Request) error {
	var updateRequest model.EntriesStatusUpdateRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		return response.WrapBadRequest(err)
	}

	err := validator.ValidateEntriesStatusUpdateRequest(&updateRequest)
	if err != nil {
		return response.WrapBadRequest(err)
	}

	if updateRequest.Status != "" {
		err = h.store.SetEntriesStatus(r.Context(), request.UserID(r),
			updateRequest.EntryIDs, updateRequest.Status)
		if err != nil {
			return err
		}
	}

	if updateRequest.Starred != nil {
		err = h.store.SetEntriesBookmarkedState(r.Context(), request.UserID(r),
			updateRequest.EntryIDs, *updateRequest.Starred)
		if err != nil {
			return err
		}
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
