// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package googlereader // import "miniflux.app/v2/internal/googlereader"

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/mux"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/integration"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/mediaproxy"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/fetcher"
	mff "miniflux.app/v2/internal/reader/handler"
	mfs "miniflux.app/v2/internal/reader/subscription"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/validator"
)

const (
	LoginPath  = "/accounts/ClientLogin"
	PathPrefix = "/reader/api/0"
)

type handler struct {
	store  *storage.Storage
	router *mux.ServeMux
}

var (
	errEmptyFeedTitle   = errors.New("googlereader: empty feed title")
	errFeedNotFound     = errors.New("googlereader: feed not found")
	errCategoryNotFound = errors.New("googlereader: category not found")
)

// Serve handles Google Reader API calls.
func Serve(m *mux.ServeMux, store *storage.Storage) {
	h := &handler{store: store, router: m}
	m.HandleFunc(LoginPath, h.clientLogin)

	m = m.PrefixGroup(PathPrefix).
		Use(WithKeyAuth(store), CORS, requestUserSession)

	m.HandleFunc("/", h.serveHandler)
	m.HandleFunc("/token", h.tokenHandler)
	m.HandleFunc("/edit-tag", h.editTagHandler)
	m.HandleFunc("/rename-tag", h.renameTagHandler)
	m.HandleFunc("/disable-tag", h.disableTagHandler)
	m.HandleFunc("/tag/list", h.tagListHandler)
	m.HandleFunc("/user-info", h.userInfoHandler)
	m.HandleFunc("/subscription/list", h.subscriptionListHandler)
	m.HandleFunc("/subscription/edit", h.editSubscriptionHandler)
	m.HandleFunc("/subscription/quickadd", h.quickAddHandler)
	m.HandleFunc("/stream/items/ids", h.streamItemIDsHandler)
	m.NameHandleFunc("/stream/items/contents", h.streamItemContentsHandler,
		"StreamItemsContents")
	m.HandleFunc("/mark-all-as-read", h.markAllAsReadHandler)
}

func checkAndSimplifyTags(addTags []Stream, removeTags []Stream) (map[StreamType]bool, error) {
	tags := make(map[StreamType]bool)
	for _, s := range addTags {
		switch s.Type {
		case ReadStream:
			if _, ok := tags[KeptUnreadStream]; ok {
				return nil, fmt.Errorf("googlereader: %s and %s should not be supplied simultaneously", keptUnreadStreamSuffix, readStreamSuffix)
			}
			tags[ReadStream] = true
		case KeptUnreadStream:
			if _, ok := tags[ReadStream]; ok {
				return nil, fmt.Errorf("googlereader: %s and %s should not be supplied simultaneously", keptUnreadStreamSuffix, readStreamSuffix)
			}
			tags[ReadStream] = false
		case StarredStream:
			tags[StarredStream] = true
		case BroadcastStream, LikeStream:
			slog.Debug("Broadcast & Like tags are not implemented!")
		default:
			return nil, fmt.Errorf("googlereader: unsupported tag type: %s", s.Type)
		}
	}
	for _, s := range removeTags {
		switch s.Type {
		case ReadStream:
			if _, ok := tags[ReadStream]; ok {
				return nil, fmt.Errorf("googlereader: %s and %s should not be supplied simultaneously", keptUnreadStreamSuffix, readStreamSuffix)
			}
			tags[ReadStream] = false
		case KeptUnreadStream:
			if _, ok := tags[ReadStream]; ok {
				return nil, fmt.Errorf("googlereader: %s and %s should not be supplied simultaneously", keptUnreadStreamSuffix, readStreamSuffix)
			}
			tags[ReadStream] = true
		case StarredStream:
			if _, ok := tags[StarredStream]; ok {
				return nil, fmt.Errorf("googlereader: %s should not be supplied for add and remove simultaneously", starredStreamSuffix)
			}
			tags[StarredStream] = false
		case BroadcastStream, LikeStream:
			slog.Debug("Broadcast & Like tags are not implemented!")
		default:
			return nil, fmt.Errorf("googlereader: unsupported tag type: %s", s.Type)
		}
	}

	return tags, nil
}

func checkOutputFormat(r *http.Request) error {
	var output string
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			return fmt.Errorf("googlereader: failed parse form: %w", err)
		}
		output = r.Form.Get("output")
	} else {
		output = request.QueryStringParam(r, "output", "")
	}
	if output != "json" {
		err := errors.New("googlereader: only json output is supported")
		return err
	}
	return nil
}

func (h *handler) clientLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logging.FromContext(ctx).With(
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))
	log.Debug("[GoogleReader] Handle /accounts/ClientLogin",
		slog.String("handler", "clientLoginHandler"))

	if err := r.ParseForm(); err != nil {
		log.Warn("[GoogleReader] Could not parse request form data",
			slog.Bool("authentication_failed", true),
			slog.Any("error", err))
		json.Unauthorized(w, r)
		return
	}

	username := r.Form.Get("Email")
	password := r.Form.Get("Passwd")
	if username == "" || password == "" {
		log.Warn("[GoogleReader] Empty username or password",
			slog.Bool("authentication_failed", true))
		json.Unauthorized(w, r)
		return
	}
	log = log.With(slog.String("username", username))

	const invalidUserMsg = "[GoogleReader] Invalid username or password"
	user, err := h.store.UserByUsername(ctx, username)
	if err != nil {
		log.Warn(invalidUserMsg,
			slog.Bool("authentication_failed", true),
			slog.Any("error", err))
		json.Unauthorized(w, r)
		return
	}

	if user == nil || !user.Integration().GoogleReaderEnabled {
		log.Warn(invalidUserMsg,
			slog.Bool("authentication_failed", true),
			slog.String("error",
				"unable find user with google reader integration enabled"))
		json.Unauthorized(w, r)
		return
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Integration().GoogleReaderPassword),
		[]byte(password))
	if err != nil {
		log.Warn(invalidUserMsg,
			slog.Bool("authentication_failed", true),
			slog.Any("error", err))
		json.Unauthorized(w, r)
		return
	}
	log.Info("[GoogleReader] User authenticated successfully",
		slog.Bool("authentication_successful", true))

	if err := h.store.SetLastLogin(ctx, user.ID); err != nil {
		log.Warn("[GoogleReader] Unable update last login",
			slog.Bool("authentication_successful", true),
			slog.Any("error", err))
		json.Unauthorized(w, r)
		return
	}

	sess, err := h.store.CreateAppSessionForUser(ctx, user, r.UserAgent(),
		request.ClientIP(r))
	if err != nil {
		log.Warn("[GoogleReader] Unable create user session",
			slog.Bool("authentication_successful", true),
			slog.Any("error", err))
		json.Unauthorized(w, r)
		return
	}

	token := sess.ID
	log.Debug("[GoogleReader] Created token", slog.String("token", token))

	result := loginResponse{SID: token, LSID: token, Auth: token}
	if r.Form.Get("output") == "json" {
		json.OK(w, r, &result)
		return
	}

	response.New(w, r).
		WithHeader("Content-Type", "text/plain; charset=UTF-8").
		WithBody(result.String()).
		Write()
}

func (h *handler) tokenHandler(w http.ResponseWriter, r *http.Request) {
	log := logging.FromContext(r.Context()).With(
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))

	log.Debug("[GoogleReader] Handle /token",
		slog.String("handler", "tokenHandler"))

	if !request.IsAuthenticated(r) {
		log.Warn("[GoogleReader] User is not authenticated")
		json.Unauthorized(w, r)
		return
	}

	token := request.GoolgeReaderToken(r)
	if token == "" {
		log.Warn("[GoogleReader] User does not have token",
			slog.Int64("user_id", request.UserID(r)))
		json.Unauthorized(w, r)
		return
	}

	log.Debug("[GoogleReader] Token handler",
		slog.Int64("user_id", request.UserID(r)),
		slog.String("token", token))

	w.Header().Add("Content-Type", "text/plain; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(token)); err != nil {
		json.ServerError(w, r, err)
	}
}

func (h *handler) editTagHandler(w http.ResponseWriter, r *http.Request) {
	user := request.User(r)
	clientIP := request.ClientIP(r)
	ctx := r.Context()
	log := logging.FromContext(ctx).With(
		slog.String("handler", "editTagHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", user.ID))
	log.Debug("[GoogleReader] Handle /edit-tag")

	if err := r.ParseForm(); err != nil {
		json.ServerError(w, r, err)
		return
	}

	addTags, err := getStreams(r.PostForm[paramTagsAdd], user.ID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	removeTags, err := getStreams(r.PostForm[paramTagsRemove], user.ID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	if len(addTags) == 0 && len(removeTags) == 0 {
		json.ServerError(w, r, errors.New(
			"googlreader: add or/and remove tags should be supplied"))
		return
	}

	tags, err := checkAndSimplifyTags(addTags, removeTags)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	itemIDs, err := parseItemIDsFromRequest(r)
	if err != nil {
		json.BadRequest(w, r, err)
		return
	}

	log.Debug("[GoogleReader] Edited tags",
		slog.Any("item_ids", itemIDs),
		slog.Any("tags", tags))

	entries, err := h.store.NewEntryQueryBuilder(user.ID).
		WithEntryIDs(itemIDs).
		WithoutStatus(model.EntryStatusRemoved).
		GetEntries(ctx)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	var n int
	readEntryIDs := make([]int64, 0)
	unreadEntryIDs := make([]int64, 0)
	starredEntryIDs := make([]int64, 0)
	unstarredEntryIDs := make([]int64, 0)
	for _, entry := range entries {
		if read, exists := tags[ReadStream]; exists {
			if read && entry.Status == model.EntryStatusUnread {
				readEntryIDs = append(readEntryIDs, entry.ID)
			} else if entry.Status == model.EntryStatusRead {
				unreadEntryIDs = append(unreadEntryIDs, entry.ID)
			}
		}
		if starred, exists := tags[StarredStream]; exists {
			if starred && !entry.Starred {
				starredEntryIDs = append(starredEntryIDs, entry.ID)
				// filter the original array
				entries[n] = entry
				n++
			} else if entry.Starred {
				unstarredEntryIDs = append(unstarredEntryIDs, entry.ID)
			}
		}
	}
	entries = entries[:n]

	g, ctx := errgroup.WithContext(ctx)
	if len(readEntryIDs) > 0 {
		g.Go(func() error {
			return h.store.SetEntriesStatus(ctx, user.ID, readEntryIDs,
				model.EntryStatusRead)
		})
	}

	if len(unreadEntryIDs) > 0 {
		g.Go(func() error {
			return h.store.SetEntriesStatus(ctx, user.ID, unreadEntryIDs,
				model.EntryStatusUnread)
		})
	}

	if len(unstarredEntryIDs) > 0 {
		g.Go(func() error {
			return h.store.SetEntriesBookmarkedState(ctx, user.ID, unstarredEntryIDs,
				false)
		})
	}

	if len(starredEntryIDs) > 0 {
		g.Go(func() error {
			return h.store.SetEntriesBookmarkedState(ctx, user.ID, starredEntryIDs,
				true)
		})
	}

	if err := g.Wait(); err != nil {
		json.ServerError(w, r, err)
		return
	}

	for _, entry := range entries {
		integration.SendEntry(entry, user)
	}
	sendOkayResponse(w)
}

func (h *handler) quickAddHandler(w http.ResponseWriter, r *http.Request) {
	user := request.User(r)
	clientIP := request.ClientIP(r)
	ctx := r.Context()
	log := logging.FromContext(ctx).With(
		slog.String("handler", "quickAddHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", user.ID))
	log.Debug("[GoogleReader] Handle /subscription/quickadd")

	if err := r.ParseForm(); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	feedURL := r.Form.Get(paramQuickAdd)
	if !validator.IsValidURL(feedURL) {
		json.BadRequest(w, r, fmt.Errorf("googlereader: invalid URL: %s", feedURL))
		return
	}

	requestBuilder := fetcher.NewRequestBuilder()
	subscriptions, lerr := mfs.
		NewSubscriptionFinder(requestBuilder).
		FindSubscriptions(ctx, feedURL,
			user.Integration().RSSBridgeURLIfEnabled(),
			user.Integration().RSSBridgeTokenIfEnabled())
	if lerr != nil {
		json.ServerError(w, r, lerr)
		return
	}

	if len(subscriptions) == 0 {
		json.OK(w, r, quickAddResponse{NumResults: 0})
		return
	}

	toSubscribe := Stream{FeedStream, subscriptions[0].URL}
	category := Stream{NoStream, ""}
	feed, err := subscribe(ctx, toSubscribe, category, "", h.store, user.ID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	log.Debug("[GoogleReader] Added a new feed",
		slog.String("feed_url", feed.FeedURL))

	json.OK(w, r, quickAddResponse{
		NumResults: 1,
		Query:      feed.FeedURL,
		StreamID:   feedPrefix + strconv.FormatInt(feed.ID, 10),
		StreamName: feed.Title,
	})
}

func getFeed(ctx context.Context, stream Stream, store *storage.Storage,
	userID int64,
) (*model.Feed, error) {
	id, err := strconv.ParseInt(stream.ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("googlereader: %w", err)
	}
	return store.FeedByID(ctx, userID, id)
}

func getOrCreateCategory(ctx context.Context, streamCategory Stream,
	store *storage.Storage, userID int64,
) (*model.Category, error) {
	if streamCategory.ID == "" {
		return store.FirstCategory(ctx, userID)
	}

	category, err := store.CategoryByTitle(ctx, userID, streamCategory.ID)
	if err != nil {
		return nil, err
	} else if category == nil {
		return store.CreateCategory(ctx, userID, &model.CategoryCreationRequest{
			Title: streamCategory.ID,
		})
	}
	return category, nil
}

func subscribe(ctx context.Context, newFeed Stream, category Stream,
	title string, store *storage.Storage, userID int64,
) (*model.Feed, error) {
	destCategory, err := getOrCreateCategory(ctx, category, store, userID)
	if err != nil {
		return nil, err
	}

	createRequest := model.FeedCreationRequest{
		FeedURL:    newFeed.ID,
		CategoryID: destCategory.ID,
	}
	lerr := validator.ValidateFeedCreation(ctx, store, userID, &createRequest)
	if lerr != nil {
		return nil, lerr.Error()
	}

	feed, lwerr := mff.CreateFeed(ctx, store, userID, &createRequest)
	if lwerr != nil {
		return nil, lwerr
	}

	if title != "" {
		modifyRequest := model.FeedModificationRequest{
			Title: &title,
		}
		modifyRequest.Patch(feed)
		if err := store.UpdateFeed(ctx, feed); err != nil {
			return nil, err
		}
	}
	return feed, nil
}

func unsubscribe(ctx context.Context, streams []Stream, store *storage.Storage,
	userID int64,
) error {
	feedIDs := make([]int64, len(streams))
	for i, stream := range streams {
		feedID, err := strconv.ParseInt(stream.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("googlereader: parse stream ID %q: %w", stream.ID, err)
		}
		feedIDs[i] = feedID
	}

	if err := store.RemoveMultipleFeeds(ctx, userID, feedIDs); err != nil {
		return err
	}
	return nil
}

func rename(ctx context.Context, feedStream Stream, title string,
	store *storage.Storage, userID int64,
) error {
	logging.FromContext(ctx).Debug("[GoogleReader] Renaming feed",
		slog.Int64("user_id", userID),
		slog.Any("feed_stream", feedStream),
		slog.String("new_title", title))

	if title == "" {
		return errEmptyFeedTitle
	}

	feed, err := getFeed(ctx, feedStream, store, userID)
	if err != nil {
		return err
	} else if feed == nil {
		return errFeedNotFound
	}

	modifyRequest := model.FeedModificationRequest{Title: &title}
	modifyRequest.Patch(feed)
	return store.UpdateFeed(ctx, feed)
}

func move(ctx context.Context, feedStream Stream, labelStream Stream,
	store *storage.Storage, userID int64,
) error {
	logging.FromContext(ctx).Debug("[GoogleReader] Moving feed",
		slog.Int64("user_id", userID),
		slog.Any("feed_stream", feedStream),
		slog.Any("label_stream", labelStream))

	feed, err := getFeed(ctx, feedStream, store, userID)
	if err != nil {
		return err
	} else if feed == nil {
		return errFeedNotFound
	}

	category, err := getOrCreateCategory(ctx, labelStream, store, userID)
	if err != nil {
		return err
	} else if category == nil {
		return errCategoryNotFound
	}

	modifyRequest := model.FeedModificationRequest{CategoryID: &category.ID}
	modifyRequest.Patch(feed)
	return store.UpdateFeed(ctx, feed)
}

func (h *handler) feedIconURL(f *model.Feed) string {
	if f.Icon == nil || f.Icon.ExternalIconID == "" {
		return ""
	}
	return config.Opts.RootURL() + route.Path(
		h.router, "feedIcon", "externalIconID", f.Icon.ExternalIconID)
}

func (h *handler) editSubscriptionHandler(w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	userID := request.UserID(r)

	logging.FromContext(ctx).Debug("[GoogleReader] Handle /subscription/edit",
		slog.String("handler", "editSubscriptionHandler"),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", userID),
	)

	if err := r.ParseForm(); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	streamIds, err := getStreams(r.Form[paramStreamID], userID)
	if err != nil || len(streamIds) == 0 {
		json.BadRequest(w, r, errors.New("googlereader: no valid stream IDs provided"))
		return
	}

	newLabel, err := getStream(r.Form.Get(paramTagsAdd), userID)
	if err != nil {
		json.BadRequest(w, r, fmt.Errorf("googlereader: invalid data in %s", paramTagsAdd))
		return
	}

	title := r.Form.Get(paramTitle)
	action := r.Form.Get(paramSubscribeAction)

	switch action {
	case "subscribe":
		_, err := subscribe(ctx, streamIds[0], newLabel, title, h.store, userID)
		if err != nil {
			json.ServerError(w, r, err)
			return
		}
	case "unsubscribe":
		err := unsubscribe(ctx, streamIds, h.store, userID)
		if err != nil {
			json.ServerError(w, r, err)
			return
		}
	case "edit":
		if title != "" {
			err := rename(ctx, streamIds[0], title, h.store, userID)
			if err != nil {
				badRequest := errors.Is(err, errFeedNotFound) ||
					errors.Is(err, errEmptyFeedTitle)
				if badRequest {
					json.BadRequest(w, r, err)
				} else {
					json.ServerError(w, r, err)
				}
				return
			}
		}

		if r.Form.Has(paramTagsAdd) {
			if newLabel.Type != LabelStream {
				json.BadRequest(w, r, errors.New("destination must be a label"))
				return
			}

			err := move(ctx, streamIds[0], newLabel, h.store, userID)
			if err != nil {
				badRequest := errors.Is(err, errFeedNotFound) ||
					errors.Is(err, errCategoryNotFound)
				if badRequest {
					json.BadRequest(w, r, err)
				} else {
					json.ServerError(w, r, err)
				}
				return
			}
		}
	default:
		json.BadRequest(w, r, fmt.Errorf(
			"googlereader: unrecognized action %s", action))
		return
	}

	sendOkayResponse(w)
}

func (h *handler) streamItemContentsHandler(w http.ResponseWriter,
	r *http.Request,
) {
	user := request.User(r)
	ctx := r.Context()
	log := logging.FromContext(ctx).With(
		slog.String("handler", "streamItemContentsHandler"),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", user.ID))
	log.Debug("[GoogleReader] Handle /stream/items/contents")

	if err := checkOutputFormat(r); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	if err := r.ParseForm(); err != nil {
		json.ServerError(w, r, err)
		return
	}

	modifiers, err := parseStreamFilterFromRequest(r, user)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	log.Debug("[GoogleReader] Request Modifiers",
		slog.Any("modifiers", modifiers))

	itemIDs, err := parseItemIDsFromRequest(r)
	if err != nil {
		json.BadRequest(w, r, err)
		return
	}

	log.Debug("[GoogleReader] Fetching item contents",
		slog.Any("item_ids", itemIDs))

	entries, err := h.store.NewEntryQueryBuilder(user.ID).
		WithContent().
		WithEntryIDs(itemIDs).
		WithoutStatus(model.EntryStatusRemoved).
		WithSorting(model.DefaultSortingOrder, modifiers.SortDirection).
		GetEntries(ctx)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	result := streamContentItemsResponse{
		Direction: "ltr",
		ID:        "user/-/state/com.google/reading-list",
		Title:     "Reading List",
		Updated:   time.Now().Unix(),
		Self: []contentHREF{
			{HREF: config.Opts.RootURL() +
				route.Path(h.router, "StreamItemsContents")},
		},
		Author: user.Username,
	}

	userReadingList := fmt.Sprintf(userStreamPrefix, user.ID) + readingListStreamSuffix
	userRead := fmt.Sprintf(userStreamPrefix, user.ID) + readStreamSuffix
	userStarred := fmt.Sprintf(userStreamPrefix, user.ID) + starredStreamSuffix

	items := make([]contentItem, len(entries))
	for i, entry := range entries {
		enclosures := make([]contentItemEnclosure, len(entry.Enclosures()))
		for j, enclosure := range entry.Enclosures() {
			enclosures[j] = contentItemEnclosure{
				URL:  enclosure.URL,
				Type: enclosure.MimeType,
			}
		}

		categories := make([]string, 0, 4)
		categories = append(categories, userReadingList)
		if entry.Feed.Category.Title != "" {
			categories = append(categories,
				fmt.Sprintf(userLabelPrefix, user.ID)+entry.Feed.Category.Title)
		}
		if entry.Status == model.EntryStatusRead {
			categories = append(categories, userRead)
		}
		if entry.Starred {
			categories = append(categories, userStarred)
		}

		entry.Content = mediaproxy.RewriteDocumentWithAbsoluteProxyURL(
			h.router, entry.Content)
		entry.Enclosures().ProxifyEnclosureURL(h.router,
			config.Opts.MediaProxyMode(), config.Opts.MediaProxyResourceTypes())

		items[i] = contentItem{
			ID:            convertEntryIDToLongFormItemID(entry.ID),
			Title:         entry.Title,
			Author:        entry.Author,
			TimestampUsec: strconv.FormatInt(entry.Date.UnixMicro(), 10),
			CrawlTimeMsec: strconv.FormatInt(entry.CreatedAt.UnixMilli(), 10),
			Published:     entry.Date.Unix(),
			Updated:       entry.ChangedAt.Unix(),
			Categories:    categories,
			Canonical:     []contentHREF{{HREF: entry.URL}},
			Alternate:     []contentHREFType{{HREF: entry.URL, Type: "text/html"}},
			Content: contentItemContent{
				Direction: "ltr",
				Content:   entry.Content,
			},
			Summary: contentItemContent{
				Direction: "ltr",
				Content:   entry.Content,
			},
			Origin: contentItemOrigin{
				StreamID: fmt.Sprintf("feed/%d", entry.FeedID),
				Title:    entry.Feed.Title,
				HTMLUrl:  entry.Feed.SiteURL,
			},
			Enclosure: enclosures,
		}
	}

	result.Items = items
	json.OK(w, r, result)
}

func (h *handler) disableTagHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := request.UserID(r)

	logging.FromContext(ctx).Debug("[GoogleReader] Handle /disable-tags",
		slog.String("handler", "disableTagHandler"),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", userID))

	if err := r.ParseForm(); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	streams, err := getStreams(r.Form[paramStreamID], userID)
	if err != nil {
		json.BadRequest(w, r, fmt.Errorf(
			"googlereader: invalid data in %s", paramStreamID))
		return
	}

	titles := make([]string, len(streams))
	for i, stream := range streams {
		if stream.Type != LabelStream {
			json.BadRequest(w, r, errors.New("googlereader: only labels are supported"))
			return
		}
		titles[i] = stream.ID
	}

	err = h.store.RemoveAndReplaceCategoriesByName(ctx, userID, titles)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	sendOkayResponse(w)
}

func (h *handler) renameTagHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := request.UserID(r)

	logging.FromContext(ctx).Debug("[GoogleReader] Handle /rename-tag",
		slog.String("handler", "renameTagHandler"),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))

	if err := r.ParseForm(); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	source, err := getStream(r.Form.Get(paramStreamID), userID)
	if err != nil {
		json.BadRequest(w, r, fmt.Errorf(
			"googlereader: invalid data in %s", paramStreamID))
		return
	}

	destination, err := getStream(r.Form.Get(paramDestination), userID)
	if err != nil {
		json.BadRequest(w, r, fmt.Errorf(
			"googlereader: invalid data in %s", paramDestination))
		return
	}

	if source.Type != LabelStream || destination.Type != LabelStream {
		json.BadRequest(w, r, errors.New("googlereader: only labels supported"))
		return
	} else if destination.ID == "" {
		json.BadRequest(w, r, errors.New("googlereader: empty destination name"))
		return
	}

	category, err := h.store.CategoryByTitle(ctx, userID, source.ID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if category == nil {
		json.NotFound(w, r)
		return
	}

	modifyRequest := model.CategoryModificationRequest{
		Title: model.SetOptionalField(destination.ID),
	}
	lerr := validator.ValidateCategoryModification(ctx, h.store, userID,
		category.ID, &modifyRequest)
	if lerr != nil {
		json.BadRequest(w, r, lerr.Error())
		return
	}

	modifyRequest.Patch(category)
	affected, err := h.store.UpdateCategory(ctx, category)
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if !affected {
		json.NotFound(w, r)
		return
	}
	sendOkayResponse(w)
}

func (h *handler) tagListHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := request.UserID(r)

	logging.FromContext(ctx).Debug("[GoogleReader] Handle /tags/list",
		slog.String("handler", "tagListHandler"),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))

	if err := checkOutputFormat(r); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	var result tagsResponse
	categories, err := h.store.Categories(ctx, userID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	result.Tags = make([]subscriptionCategoryResponse, 0, len(categories)+1)
	result.Tags = append(result.Tags, subscriptionCategoryResponse{
		ID: fmt.Sprintf(userStreamPrefix, userID) + starredStreamSuffix,
	})
	for _, category := range categories {
		result.Tags = append(result.Tags, subscriptionCategoryResponse{
			ID:    fmt.Sprintf(userLabelPrefix, userID) + category.Title,
			Label: category.Title,
			Type:  "folder",
		})
	}
	json.OK(w, r, result)
}

func (h *handler) subscriptionListHandler(w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	userID := request.UserID(r)

	logging.FromContext(ctx).Debug("[GoogleReader] Handle /subscription/list",
		slog.String("handler", "subscriptionListHandler"),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))

	if err := checkOutputFormat(r); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	feeds, err := h.store.Feeds(ctx, userID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	result := subscriptionsResponse{
		Subscriptions: make([]subscriptionResponse, len(feeds)),
	}
	for i, feed := range feeds {
		result.Subscriptions[i] = subscriptionResponse{
			ID:    feedPrefix + strconv.FormatInt(feed.ID, 10),
			Title: feed.Title,
			URL:   feed.FeedURL,
			Categories: []subscriptionCategoryResponse{
				{
					ID:    fmt.Sprintf(userLabelPrefix, userID) + feed.Category.Title,
					Label: feed.Category.Title,
					Type:  "folder",
				},
			},
			HTMLURL: feed.SiteURL,
			IconURL: h.feedIconURL(feed),
		}
	}
	json.OK(w, r, result)
}

func (h *handler) serveHandler(w http.ResponseWriter, r *http.Request) {
	logging.FromContext(r.Context()).Debug(
		"[GoogleReader] API endpoint not implemented yet",
		slog.Any("url", r.RequestURI),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))
	json.OK(w, r, []string{})
}

func (h *handler) userInfoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logging.FromContext(ctx).Debug("[GoogleReader] Handle /user-info",
		slog.String("handler", "userInfoHandler"),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))

	if err := checkOutputFormat(r); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	user := request.User(r)
	userId := strconv.FormatInt(user.ID, 10)
	userInfo := userInfoResponse{
		UserID:        userId,
		UserName:      user.Username,
		UserProfileID: userId,
		UserEmail:     user.Username,
	}
	json.OK(w, r, userInfo)
}

func (h *handler) streamItemIDsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := request.User(r)

	log := logging.FromContext(ctx).With(
		slog.String("handler", "streamItemIDsHandler"),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", user.ID))
	log.Debug("[GoogleReader] Handle /stream/items/ids")

	if err := checkOutputFormat(r); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	modifiers, err := parseStreamFilterFromRequest(r, user)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	log.Debug("[GoogleReader] Request Modifiers",
		slog.Any("modifiers", modifiers))

	if len(modifiers.Streams) != 1 {
		json.ServerError(w, r, errors.New(
			"googlereader: only one stream type expected"))
		return
	}

	switch modifiers.Streams[0].Type {
	case ReadingListStream:
		h.handleReadingListStreamHandler(w, r, modifiers)
	case StarredStream:
		h.handleStarredStreamHandler(w, r, modifiers)
	case ReadStream:
		h.handleReadStreamHandler(w, r, modifiers)
	case FeedStream:
		h.handleFeedStreamHandler(w, r, modifiers)
	default:
		log.Warn("[GoogleReader] Unknown Stream",
			slog.Any("stream_type", modifiers.Streams[0].Type))
		json.ServerError(w, r, fmt.Errorf(
			"googlereader: unknown stream type %s", modifiers.Streams[0].Type))
	}
}

func (h *handler) handleReadingListStreamHandler(w http.ResponseWriter,
	r *http.Request, rm RequestModifiers,
) {
	ctx := r.Context()
	log := logging.FromContext(ctx).With(
		slog.String("handler", "handleReadingListStreamHandler"),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))
	slog.Debug("[GoogleReader] Handle ReadingListStream")

	builder := h.store.NewEntryQueryBuilder(rm.UserID)
	for _, s := range rm.ExcludeTargets {
		switch s.Type {
		case ReadStream:
			builder.WithStatus(model.EntryStatusUnread)
		default:
			log.Warn("[GoogleReader] Unknown ExcludeTargets filter type",
				slog.Int("filter_type", int(s.Type)))
		}
	}

	builder.WithoutStatus(model.EntryStatusRemoved)
	streamId, err := makeStreamIDResp(ctx, builder, &rm)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.OK(w, r, streamId)
}

func makeStreamIDResp(ctx context.Context, builder *storage.EntryQueryBuilder,
	rm *RequestModifiers,
) (streamIDResponse, error) {
	builder.WithLimit(rm.Count).
		WithOffset(rm.Offset).
		WithSorting(model.DefaultSortingOrder, rm.SortDirection)

	if rm.StartTime > 0 {
		builder.AfterPublishedDate(time.Unix(rm.StartTime, 0))
	}
	if rm.StopTime > 0 {
		builder.BeforePublishedDate(time.Unix(rm.StopTime, 0))
	}

	g, ctx := errgroup.WithContext(ctx)
	var rawEntryIDs []int64
	g.Go(func() (err error) {
		rawEntryIDs, err = builder.GetEntryIDs(ctx)
		return
	})

	var totalEntries int
	g.Go(func() (err error) {
		totalEntries, err = builder.CountEntries(ctx)
		return
	})

	if err := g.Wait(); err != nil {
		//nolint:wrapcheck // this err from our package
		return streamIDResponse{}, err
	}

	itemRefs := make([]itemRef, len(rawEntryIDs))
	for i, entryID := range rawEntryIDs {
		itemRefs[i] = itemRef{ID: strconv.FormatInt(entryID, 10)}
	}

	var continuation int
	if len(itemRefs)+rm.Offset < totalEntries {
		continuation = len(itemRefs) + rm.Offset
	}
	return streamIDResponse{itemRefs, continuation}, nil
}

func (h *handler) handleStarredStreamHandler(w http.ResponseWriter,
	r *http.Request, rm RequestModifiers,
) {
	builder := h.store.NewEntryQueryBuilder(rm.UserID).
		WithoutStatus(model.EntryStatusRemoved).
		WithStarred(true)

	streamId, err := makeStreamIDResp(r.Context(), builder, &rm)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.OK(w, r, streamId)
}

func (h *handler) handleReadStreamHandler(w http.ResponseWriter,
	r *http.Request, rm RequestModifiers,
) {
	builder := h.store.NewEntryQueryBuilder(rm.UserID).
		WithoutStatus(model.EntryStatusRemoved).
		WithStatus(model.EntryStatusRead)

	streamId, err := makeStreamIDResp(r.Context(), builder, &rm)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.OK(w, r, streamId)
}

func (h *handler) handleFeedStreamHandler(w http.ResponseWriter,
	r *http.Request, rm RequestModifiers,
) {
	id, err := strconv.ParseInt(rm.Streams[0].ID, 10, 64)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	builder := h.store.NewEntryQueryBuilder(rm.UserID).
		WithFeedID(id).
		WithoutStatus(model.EntryStatusRemoved)

	for _, s := range rm.ExcludeTargets {
		if s.Type == ReadStream {
			builder.WithoutStatus(model.EntryStatusRead)
			break
		}
	}

	streamId, err := makeStreamIDResp(r.Context(), builder, &rm)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.OK(w, r, streamId)
}

func (h *handler) markAllAsReadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logging.FromContext(ctx).Debug("[GoogleReader] Handle /mark-all-as-read",
		slog.String("handler", "markAllAsReadHandler"),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))

	if err := r.ParseForm(); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	userID := request.UserID(r)
	stream, err := getStream(r.Form.Get(paramStreamID), userID)
	if err != nil {
		json.BadRequest(w, r, err)
		return
	}

	var before time.Time
	if timestampString := r.Form.Get(paramTimestamp); timestampString != "" {
		ts, err := strconv.ParseInt(timestampString, 10, 64)
		if err != nil {
			json.BadRequest(w, r, err)
			return
		}

		if ts > 0 {
			// It's unclear if the timestamp is in seconds or microseconds, so we try
			// both using a naive approach.
			if len(timestampString) >= 16 {
				before = time.UnixMicro(ts)
			} else {
				before = time.Unix(ts, 0)
			}
		}
	}

	if before.IsZero() {
		before = time.Now()
	}

	switch stream.Type {
	case FeedStream:
		feedID, err := strconv.ParseInt(stream.ID, 10, 64)
		if err != nil {
			json.BadRequest(w, r, err)
			return
		}
		affected, err := h.store.MarkFeedAsRead(ctx, userID, feedID, before)
		if err != nil {
			json.ServerError(w, r, err)
			return
		} else if !affected {
			json.NotFound(w, r)
			return
		}
	case LabelStream:
		category, err := h.store.CategoryByTitle(ctx, userID, stream.ID)
		if err != nil {
			json.ServerError(w, r, err)
			return
		} else if category == nil {
			json.NotFound(w, r)
			return
		}
		affected, err := h.store.MarkCategoryAsRead(ctx, userID, category.ID,
			before)
		if err != nil {
			json.ServerError(w, r, err)
			return
		} else if !affected {
			json.NotFound(w, r)
			return
		}
	case ReadingListStream:
		err := h.store.MarkAllAsReadBeforeDate(ctx, userID, before)
		if err != nil {
			json.ServerError(w, r, err)
			return
		}
	}

	sendOkayResponse(w)
}
