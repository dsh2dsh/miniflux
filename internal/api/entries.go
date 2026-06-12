package api

import (
	json_parser "encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/http/mux"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/mediaproxy"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/readingtime"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/validator"
)

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

func (h *handler) getFeedEntries(w http.ResponseWriter, r *http.Request,
) (*entriesResponse, error) {
	feedID := request.RouteInt64Param(r, "feedID")
	return h.entriesFinder().WithFeedID(feedID).Entries(r)
}

func (h *handler) getCategoryEntries(w http.ResponseWriter, r *http.Request,
) (*entriesResponse, error) {
	id := request.RouteInt64Param(r, "categoryID")
	return h.entriesFinder().WithCategoryID(id).Entries(r)
}

func (h *handler) getEntries(w http.ResponseWriter, r *http.Request,
) (*entriesResponse, error) {
	return h.entriesFinder().Entries(r)
}

func (h *handler) getEntryIDs(w http.ResponseWriter, r *http.Request,
) (*entryIDsResponse, error) {
	entries, err := h.entriesFinder().WithContent(false).Entries(r)
	if err != nil {
		return nil, err
	}

	entryIDs := &entryIDsResponse{
		Total:    entries.Total,
		EntryIDs: make([]int64, len(entries.Entries)),
	}

	for i, e := range entries.Entries {
		entryIDs.EntryIDs[i] = e.ID
	}
	return entryIDs, nil
}

func (h *handler) entriesFinder() *entriesFinder {
	return &entriesFinder{
		store:        h.store,
		router:       h.router,
		fetchContent: true,
	}
}

type entriesFinder struct {
	store  *storage.Storage
	router *mux.ServeMux

	feedID       int64
	categoryID   int64
	fetchContent bool
}

func (self *entriesFinder) WithFeedID(id int64) *entriesFinder {
	self.feedID = id
	return self
}

func (self *entriesFinder) WithCategoryID(id int64) *entriesFinder {
	self.categoryID = id
	return self
}

func (self *entriesFinder) WithContent(v bool) *entriesFinder {
	self.fetchContent = v
	return self
}

func (self *entriesFinder) Entries(r *http.Request) (*entriesResponse, error) {
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
	categoryID := request.QueryInt64Param(r, "category_id", self.categoryID)
	if categoryID > 0 {
		g.Go(func() error {
			if !self.store.CategoryIDExists(ctx, userID, categoryID) {
				return fmt.Errorf("%w category ID", errInvalid)
			}
			return nil
		})
	}

	feedID := request.QueryInt64Param(r, "feed_id", self.feedID)
	if feedID > 0 {
		g.Go(func() error {
			if !self.store.FeedExists(ctx, userID, feedID) {
				return fmt.Errorf("%w feed ID", errInvalid)
			}
			return nil
		})
	}

	b := self.store.NewEntryQueryBuilder(userID).
		WithFeedID(feedID).
		WithCategoryID(categoryID).
		WithStatuses(statuses).
		WithSorting(order, direction).
		WithOffset(offset).
		WithLimit(limit).
		WithTags(request.QueryStringParamList(r, "tags")).
		WithContent(self.fetchContent).
		WithoutStatus(model.EntryStatusRemoved)
	self.filter(b, r)

	var entries model.Entries
	g.Go(func() (err error) {
		entries, err = b.GetEntries(r.Context())
		return err
	})

	var count int
	g.Go(func() (err error) {
		count, err = b.CountEntries(r.Context())
		return err
	})

	if err := g.Wait(); err != nil {
		if errors.Is(err, errInvalid) {
			return nil, response.WrapBadRequest(err)
		}
		return nil, err //nolint:wrapcheck // from our package inside Go
	}

	resp := &entriesResponse{Total: count, Entries: entries}
	if !self.fetchContent {
		return resp, nil
	}

	for i := range entries {
		entries[i].Content = mediaproxy.RewriteDocumentWithAbsoluteProxyURL(
			self.router, entries[i].Content)
	}
	return resp, nil
}

func (self *entriesFinder) filter(b *storage.EntryQueryBuilder, r *http.Request,
) {
	if request.HasQueryParam(r, "globally_visible") {
		globallyVisible := request.QueryBoolParam(r, "globally_visible", true)
		if globallyVisible {
			b.WithGloballyVisible()
		}
	}

	beforeEntryID := request.QueryInt64Param(r, "before_entry_id", 0)
	if beforeEntryID > 0 {
		b.BeforeEntryID(beforeEntryID)
	}

	afterEntryID := request.QueryInt64Param(r, "after_entry_id", 0)
	if afterEntryID > 0 {
		b.AfterEntryID(afterEntryID)
	}

	beforePublished := request.QueryInt64Param(r, "before", 0)
	if beforePublished > 0 {
		b.BeforePublishedDate(time.Unix(beforePublished, 0))
	}

	afterPublished := request.QueryInt64Param(r, "after", 0)
	if afterPublished > 0 {
		b.AfterPublishedDate(time.Unix(afterPublished, 0))
	}

	beforePublished = request.QueryInt64Param(r, "published_before", 0)
	if beforePublished > 0 {
		b.BeforePublishedDate(time.Unix(beforePublished, 0))
	}

	afterPublished = request.QueryInt64Param(r, "published_after", 0)
	if afterPublished > 0 {
		b.AfterPublishedDate(time.Unix(afterPublished, 0))
	}

	beforeChanged := request.QueryInt64Param(r, "changed_before", 0)
	if beforeChanged > 0 {
		b.BeforeChangedDate(time.Unix(beforeChanged, 0))
	}

	afterChanged := request.QueryInt64Param(r, "changed_after", 0)
	if afterChanged > 0 {
		b.AfterChangedDate(time.Unix(afterChanged, 0))
	}

	categoryID := request.QueryInt64Param(r, "category_id", 0)
	if categoryID > 0 {
		b.WithCategoryID(categoryID)
	}

	if request.HasQueryParam(r, "starred") {
		starred, err := strconv.ParseBool(r.URL.Query().Get("starred"))
		if err == nil {
			b.WithStarred(starred)
		}
	}

	searchQuery := request.QueryStringParam(r, "search", "")
	if searchQuery != "" {
		b.WithSearchQuery(searchQuery)
	}
}
