// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package fever // import "miniflux.app/v2/internal/fever"

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/http/mux"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/integration"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/mediaproxy"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/storage"
)

const PathPrefix = "/fever"

// Serve handles Fever API calls.
func Serve(router *mux.ServeMux, store *storage.Storage) {
	h := &handler{store: store, router: router}
	router.PrefixGroup(PathPrefix).
		Use(requestUser).
		NameHandleFunc("/", h.serve, "feverEndpoint")
}

type handler struct {
	store  *storage.Storage
	router *mux.ServeMux
}

func (h *handler) serve(w http.ResponseWriter, r *http.Request) {
	switch {
	case request.HasQueryParam(r, "groups"):
		h.handleGroups(w, r)
	case request.HasQueryParam(r, "feeds"):
		h.handleFeeds(w, r)
	case request.HasQueryParam(r, "favicons"):
		h.handleFavicons(w, r)
	case request.HasQueryParam(r, "unread_item_ids"):
		h.handleUnreadItems(w, r)
	case request.HasQueryParam(r, "saved_item_ids"):
		h.handleSavedItems(w, r)
	case request.HasQueryParam(r, "items"):
		h.handleItems(w, r)
	case r.FormValue("mark") == "item":
		h.handleWriteItems(w, r)
	case r.FormValue("mark") == "feed":
		h.handleWriteFeeds(w, r)
	case r.FormValue("mark") == "group":
		h.handleWriteGroups(w, r)
	default:
		json.OK(w, r, newBaseResponse())
	}
}

/*
A request with the groups argument will return two additional members:

	groups contains an array of group objects
	feeds_groups contains an array of feeds_group objects

A group object has the following members:

	id (positive integer)
	title (utf-8 string)

The feeds_group object is documented under “Feeds/Groups Relationships.”

The “Kindling” super group is not included in this response and is composed of all feeds with
an is_spark equal to 0.

The “Sparks” super group is not included in this response and is composed of all feeds with an
is_spark equal to 1.
*/
func (h *handler) handleGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := request.UserID(r)
	logging.FromContext(ctx).Debug("[Fever] Fetching groups",
		slog.Int64("user_id", userID))

	categories, err := h.store.Categories(ctx, userID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	feeds, err := h.store.Feeds(ctx, userID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	result := groupsResponse{Groups: make([]group, len(categories))}
	for i, category := range categories {
		result.Groups[i] = group{ID: category.ID, Title: category.Title}
	}

	result.FeedsGroups = h.buildFeedGroups(feeds)
	result.SetCommonValues()
	json.OK(w, r, result)
}

/*
A request with the feeds argument will return two additional members:

	feeds contains an array of group objects
	feeds_groups contains an array of feeds_group objects

A feed object has the following members:

	id (positive integer)
	favicon_id (positive integer)
	title (utf-8 string)
	url (utf-8 string)
	site_url (utf-8 string)
	is_spark (boolean integer)
	last_updated_on_time (Unix timestamp/integer)

The feeds_group object is documented under “Feeds/Groups Relationships.”

The “All Items” super feed is not included in this response and is composed of all items from all feeds
that belong to a given group. For the “Kindling” super group and all user created groups the items
should be limited to feeds with an is_spark equal to 0.

For the “Sparks” super group the items should be limited to feeds with an is_spark equal to 1.
*/
func (h *handler) handleFeeds(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := request.UserID(r)
	logging.FromContext(ctx).Debug("[Fever] Fetching feeds",
		slog.Int64("user_id", userID))

	feeds, err := h.store.Feeds(ctx, userID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	result := feedsResponse{Feeds: make([]feed, len(feeds))}
	for i, f := range feeds {
		subscripion := feed{
			ID:          f.ID,
			Title:       f.Title,
			URL:         f.FeedURL,
			SiteURL:     f.SiteURL,
			IsSpark:     0,
			LastUpdated: f.CheckedAt.Unix(),
		}

		if f.Icon != nil {
			subscripion.FaviconID = f.Icon.IconID
		}
		result.Feeds[i] = subscripion
	}

	result.FeedsGroups = h.buildFeedGroups(feeds)
	result.SetCommonValues()
	json.OK(w, r, result)
}

/*
A request with the favicons argument will return one additional member:

	favicons contains an array of favicon objects

A favicon object has the following members:

	id (positive integer)
	data (base64 encoded image data; prefixed by image type)

An example data value:

	image/gif;base64,R0lGODlhAQABAIAAAObm5gAAACH5BAEAAAAALAAAAAABAAEAAAICRAEAOw==

The data member of a favicon object can be used with the data: protocol to embed an image in CSS or HTML.
A PHP/HTML example:

	echo '<img src="data:'.$favicon['data'].'">';
*/
func (h *handler) handleFavicons(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := request.UserID(r)
	logging.FromContext(ctx).Debug("[Fever] Fetching favicons",
		slog.Int64("user_id", userID))

	icons, err := h.store.Icons(ctx, userID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	result := faviconsResponse{Favicons: make([]favicon, len(icons))}
	for i, icon := range icons {
		result.Favicons[i] = favicon{ID: icon.ID, Data: icon.DataURL()}
	}
	result.SetCommonValues()
	json.OK(w, r, result)
}

/*
A request with the items argument will return two additional members:

	items contains an array of item objects
	total_items contains the total number of items stored in the database (added in API version 2)

An item object has the following members:

	id (positive integer)
	feed_id (positive integer)
	title (utf-8 string)
	author (utf-8 string)
	html (utf-8 string)
	url (utf-8 string)
	is_saved (boolean integer)
	is_read (boolean integer)
	created_on_time (Unix timestamp/integer)

Most servers won’t have enough memory allocated to PHP to dump all items at once.
Three optional arguments control determine the items included in the response.

	Use the since_id argument with the highest id of locally cached items to request 50 additional items.
	Repeat until the items array in the response is empty.

	Use the max_id argument with the lowest id of locally cached items (or 0 initially) to request 50 previous items.
	Repeat until the items array in the response is empty. (added in API version 2)

	Use the with_ids argument with a comma-separated list of item ids to request (a maximum of 50) specific items.
	(added in API version 2)
*/
func (h *handler) handleItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := request.UserID(r)
	log := logging.FromContext(ctx).With(
		slog.Int64("user_id", userID))

	builder := h.store.NewEntryQueryBuilder(userID).
		WithoutStatus(model.EntryStatusRemoved).
		WithLimit(50)

	switch {
	case request.HasQueryParam(r, "since_id"):
		sinceID := request.QueryInt64Param(r, "since_id", 0)
		if sinceID > 0 {
			log.Debug("[Fever] Fetching items since a given date",
				slog.Int64("since_id", sinceID))
			builder.AfterEntryID(sinceID).WithSorting("id", "ASC")
		}
	case request.HasQueryParam(r, "max_id"):
		maxID := request.QueryInt64Param(r, "max_id", 0)
		if maxID == 0 {
			log.Debug("[Fever] Fetching most recent items")
			builder.WithSorting("id", "DESC")
		} else if maxID > 0 {
			log.Debug("[Fever] Fetching items before a given item ID",
				slog.Int64("max_id", maxID))
			builder.BeforeEntryID(maxID).WithSorting("id", "DESC")
		}
	case request.HasQueryParam(r, "with_ids"):
		csvItemIDs := request.QueryStringParam(r, "with_ids", "")
		if csvItemIDs != "" {
			var itemIDs []int64
			for strItemID := range strings.SplitSeq(csvItemIDs, ",") {
				strItemID = strings.TrimSpace(strItemID)
				itemID, _ := strconv.ParseInt(strItemID, 10, 64)
				itemIDs = append(itemIDs, itemID)
			}
			builder.WithEntryIDs(itemIDs)
		}
	default:
		log.Debug("[Fever] Fetching oldest items")
	}

	g, ctx := errgroup.WithContext(ctx)
	var entries model.Entries
	g.Go(func() (err error) {
		entries, err = builder.GetEntries(ctx)
		return
	})

	var result itemsResponse
	g.Go(func() (err error) {
		result.Total, err = h.store.NewEntryQueryBuilder(userID).
			WithoutStatus(model.EntryStatusRemoved).
			CountEntries(ctx)
		return
	})

	if err := g.Wait(); err != nil {
		json.ServerError(w, r, err)
		return
	}

	result.Items = make([]item, len(entries))
	for i, entry := range entries {
		var isRead int
		if entry.Status == model.EntryStatusRead {
			isRead = 1
		}

		var isSaved int
		if entry.Starred {
			isSaved = 1
		}

		result.Items[i] = item{
			ID:     entry.ID,
			FeedID: entry.FeedID,
			Title:  entry.Title,
			Author: entry.Author,
			HTML: mediaproxy.RewriteDocumentWithAbsoluteProxyURL(h.router,
				entry.Content),
			URL:       entry.URL,
			IsSaved:   isSaved,
			IsRead:    isRead,
			CreatedAt: entry.Date.Unix(),
		}
	}

	result.SetCommonValues()
	json.OK(w, r, result)
}

/*
The unread_item_ids and saved_item_ids arguments can be used to keep your local cache synced
with the remote Fever installation.

A request with the unread_item_ids argument will return one additional member:

	unread_item_ids (string/comma-separated list of positive integers)
*/
func (h *handler) handleUnreadItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := request.UserID(r)
	logging.FromContext(ctx).Debug("[Fever] Fetching unread items",
		slog.Int64("user_id", userID))

	rawEntryIDs, err := h.store.NewEntryQueryBuilder(userID).
		WithStatus(model.EntryStatusUnread).
		GetEntryIDs(ctx)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	itemIDs := make([]string, len(rawEntryIDs))
	for i, entryID := range rawEntryIDs {
		itemIDs[i] = strconv.FormatInt(entryID, 10)
	}

	result := unreadResponse{ItemIDs: strings.Join(itemIDs, ",")}
	result.SetCommonValues()
	json.OK(w, r, result)
}

/*
The unread_item_ids and saved_item_ids arguments can be used to keep your local cache synced
with the remote Fever installation.

	A request with the saved_item_ids argument will return one additional member:

	saved_item_ids (string/comma-separated list of positive integers)
*/
func (h *handler) handleSavedItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := request.UserID(r)
	logging.FromContext(ctx).Debug("[Fever] Fetching saved items",
		slog.Int64("user_id", userID))

	entryIDs, err := h.store.NewEntryQueryBuilder(userID).
		WithStarred(true).
		GetEntryIDs(ctx)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	itemsIDs := make([]string, len(entryIDs))
	for i, entryID := range entryIDs {
		itemsIDs[i] = strconv.FormatInt(entryID, 10)
	}

	result := &savedResponse{ItemIDs: strings.Join(itemsIDs, ",")}
	result.SetCommonValues()
	json.OK(w, r, result)
}

/*
mark=item
as=? where ? is replaced with read, saved or unsaved
id=? where ? is replaced with the id of the item to modify
*/
func (h *handler) handleWriteItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := request.UserID(r)
	log := logging.FromContext(ctx).With(slog.Int64("user_id", userID))
	log.Debug("[Fever] Receiving mark=item call")

	entryID := request.FormInt64Value(r, "id")
	if entryID <= 0 {
		return
	}

	entry, err := h.store.NewEntryQueryBuilder(userID).
		WithEntryID(entryID).
		WithoutStatus(model.EntryStatusRemoved).
		GetEntry(ctx)
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if entry == nil {
		log.Debug("[Fever] Entry not found", slog.Int64("entry_id", entryID))
		json.OK(w, r, newBaseResponse())
		return
	}
	log = log.With(slog.Int64("entry_id", entryID))

	switch r.FormValue("as") {
	case "read":
		log.Debug("[Fever] Mark entry as read")
		err = h.store.SetEntriesStatus(ctx, userID, []int64{entryID},
			model.EntryStatusRead)
	case "unread":
		log.Debug("[Fever] Mark entry as unread")
		err = h.store.SetEntriesStatus(ctx, userID, []int64{entryID},
			model.EntryStatusUnread)
	case "saved":
		log.Debug("[Fever] Mark entry as saved")
		err := h.store.ToggleBookmark(ctx, userID, entryID)
		if err != nil {
			json.ServerError(w, r, err)
			return
		}
		integration.SendEntry(entry, request.User(r))
	case "unsaved":
		log.Debug("[Fever] Mark entry as unsaved")
		err = h.store.ToggleBookmark(ctx, userID, entryID)
	}
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.OK(w, r, newBaseResponse())
}

/*
mark=feed
as=read
id=? where ? is replaced with the id of the feed or group to modify
before=? where ? is replaced with the Unix timestamp of the the local client’s most recent items API request
*/
func (h *handler) handleWriteFeeds(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	feedID := request.FormInt64Value(r, "id")
	before := time.Unix(request.FormInt64Value(r, "before"), 0)

	ctx := r.Context()
	log := logging.FromContext(ctx).With(
		slog.Int64("user_id", userID),
		slog.Int64("feed_id", feedID),
		slog.Time("before_ts", before))
	log.Debug("[Fever] Mark feed as read before a given date")

	if feedID <= 0 {
		return
	}

	_, err := h.store.MarkFeedAsRead(ctx, userID, feedID, before)
	if err != nil {
		log.Error("[Fever] Unable to mark feed as read", slog.Any("error", err))
	}
	json.OK(w, r, newBaseResponse())
}

/*
mark=group
as=read
id=? where ? is replaced with the id of the feed or group to modify
before=? where ? is replaced with the Unix timestamp of the the local client’s most recent items API request
*/
func (h *handler) handleWriteGroups(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	groupID := request.FormInt64Value(r, "id")
	before := time.Unix(request.FormInt64Value(r, "before"), 0)

	ctx := r.Context()
	log := logging.FromContext(ctx).With(
		slog.Int64("user_id", userID),
		slog.Int64("group_id", groupID),
		slog.Time("before_ts", before))
	log.Debug("[Fever] Mark group as read before a given date")

	if groupID < 0 {
		return
	}

	var err error
	if groupID == 0 {
		err = h.store.MarkAllAsRead(ctx, userID)
	} else {
		_, err = h.store.MarkCategoryAsRead(ctx, userID, groupID, before)
	}
	if err != nil {
		log.Error("[Fever] Unable to mark group as read", slog.Any("error", err))
	}
	json.OK(w, r, newBaseResponse())
}

/*
A feeds_group object has the following members:

	group_id (positive integer)
	feed_ids (string/comma-separated list of positive integers)
*/
func (h *handler) buildFeedGroups(feeds model.Feeds) []feedsGroups {
	byCategory := make(map[int64][]string)
	for _, feed := range feeds {
		byCategory[feed.Category.ID] = append(byCategory[feed.Category.ID],
			strconv.FormatInt(feed.ID, 10))
	}

	result := make([]feedsGroups, 0, len(byCategory))
	for categoryID, feedIDs := range byCategory {
		result = append(result,
			feedsGroups{GroupID: categoryID, FeedIDs: strings.Join(feedIDs, ",")})
	}
	return result
}
