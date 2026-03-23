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
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/integration"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/mediaproxy"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/fetcher"
	mff "miniflux.app/v2/internal/reader/handler"
	mfs "miniflux.app/v2/internal/reader/subscription"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/template"
	"miniflux.app/v2/internal/urllib"
	"miniflux.app/v2/internal/validator"
)

const (
	LoginPath  = "/accounts/ClientLogin"
	PathPrefix = "/reader/api/0"
)

type handler struct {
	store     *storage.Storage
	router    *mux.ServeMux
	templates *template.Engine
}

var (
	errEmptyFeedTitle   = errors.New("googlereader: empty feed title")
	errFeedNotFound     = errors.New("googlereader: feed not found")
	errCategoryNotFound = errors.New("googlereader: category not found")
	errSimultaneously   = fmt.Errorf("googlereader: %s and %s should not be supplied simultaneously", keptUnreadStreamSuffix, readStreamSuffix)
)

// Serve handles Google Reader API calls.
func Serve(m *mux.ServeMux, store *storage.Storage, t *template.Engine) {
	h := &handler{store: store, router: m, templates: t}
	m.HandleFunc(LoginPath, h.clientLogin)

	m = m.PrefixGroup(PathPrefix)
	m.Use(WithKeyAuth(store), requestUserSession)

	m.HandleFunc("/", response.JSON(h.serveHandler))
	m.HandleFunc("/token", h.tokenHandler)
	m.HandleFunc("/edit-tag", h.editTagHandler)
	m.HandleFunc("/rename-tag", h.renameTagHandler)
	m.HandleFunc("/disable-tag", h.disableTagHandler)
	m.HandleFunc("/tag/list", response.JSON(h.tagListHandler))
	m.HandleFunc("/user-info", response.JSON(h.userInfoHandler))
	m.HandleFunc("/subscription/list", response.JSON(h.subscriptionListHandler))
	m.HandleFunc("/subscription/edit", h.editSubscriptionHandler)
	m.HandleFunc("/subscription/quickadd", response.JSON(h.quickAddHandler))
	m.HandleFunc("/stream/items/ids", response.JSON(h.streamItemIDsHandler))
	m.NameHandleFunc("/stream/items/contents",
		response.JSON(h.streamItemContentsHandler), "StreamItemsContents")
	m.HandleFunc("/mark-all-as-read", h.markAllAsReadHandler)
}

func checkAndSimplifyTags(addTags, removeTags []Stream) (map[StreamType]bool, error) {
	tags := make(map[StreamType]bool)
	for _, s := range addTags {
		switch s.Type {
		case ReadStream:
			if _, ok := tags[KeptUnreadStream]; ok {
				return nil, errSimultaneously
			}
			tags[ReadStream] = true
		case KeptUnreadStream:
			if _, ok := tags[ReadStream]; ok {
				return nil, errSimultaneously
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
				return nil, errSimultaneously
			}
			tags[ReadStream] = false
		case KeptUnreadStream:
			if _, ok := tags[ReadStream]; ok {
				return nil, errSimultaneously
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
		response.UnauthorizedJSON(w, r)
		return
	}

	username := r.Form.Get("Email")
	password := r.Form.Get("Passwd")
	if username == "" || password == "" {
		log.Warn("[GoogleReader] Empty username or password",
			slog.Bool("authentication_failed", true))
		response.UnauthorizedJSON(w, r)
		return
	}
	log = log.With(slog.String("username", username))

	const invalidUserMsg = "[GoogleReader] Invalid username or password"
	user, err := h.store.UserByUsername(ctx, username)
	if err != nil {
		log.Warn(invalidUserMsg,
			slog.Bool("authentication_failed", true),
			slog.Any("error", err))
		response.UnauthorizedJSON(w, r)
		return
	}

	if user == nil || !user.Integration().GoogleReaderEnabled {
		log.Warn(invalidUserMsg,
			slog.Bool("authentication_failed", true),
			slog.String("error",
				"unable find user with google reader integration enabled"))
		response.UnauthorizedJSON(w, r)
		return
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Integration().GoogleReaderPassword),
		[]byte(password))
	if err != nil {
		log.Warn(invalidUserMsg,
			slog.Bool("authentication_failed", true),
			slog.Any("error", err))
		response.UnauthorizedJSON(w, r)
		return
	}
	log.Info("[GoogleReader] User authenticated successfully",
		slog.Bool("authentication_successful", true))

	if err := h.store.SetLastLogin(ctx, user.ID); err != nil {
		log.Warn("[GoogleReader] Unable update last login",
			slog.Bool("authentication_successful", true),
			slog.Any("error", err))
		response.UnauthorizedJSON(w, r)
		return
	}

	sess, err := h.store.CreateAppSessionForUser(ctx, user, r.UserAgent(),
		request.ClientIP(r))
	if err != nil {
		log.Warn("[GoogleReader] Unable create user session",
			slog.Bool("authentication_successful", true),
			slog.Any("error", err))
		response.UnauthorizedJSON(w, r)
		return
	}

	token := sess.ID
	log.Debug("[GoogleReader] Created token", slog.String("token", token))

	result := loginResponse{SID: token, LSID: token, Auth: token}
	if r.Form.Get("output") == "json" {
		response.MarshalJSON(w, r, &result)
		return
	}
	response.Text(w, r, result.String())
}

func (h *handler) tokenHandler(w http.ResponseWriter, r *http.Request) {
	log := logging.FromContext(r.Context()).With(
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))

	log.Debug("[GoogleReader] Handle /token",
		slog.String("handler", "tokenHandler"))

	if !request.IsAuthenticated(r) {
		log.Warn("[GoogleReader] User is not authenticated")
		response.UnauthorizedJSON(w, r)
		return
	}

	token := request.GoogleReaderToken(r)
	if token == "" {
		log.Warn("[GoogleReader] User does not have token",
			slog.Int64("user_id", request.UserID(r)))
		response.UnauthorizedJSON(w, r)
		return
	}

	log.Debug("[GoogleReader] Token handler",
		slog.Int64("user_id", request.UserID(r)),
		slog.String("token", token))
	response.Text(w, r, token)
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
		response.ServerErrorJSON(w, r, err)
		return
	}

	addTags, err := getStreams(r.PostForm[paramTagsAdd], user.ID)
	if err != nil {
		response.ServerErrorJSON(w, r, err)
		return
	}

	removeTags, err := getStreams(r.PostForm[paramTagsRemove], user.ID)
	if err != nil {
		response.ServerErrorJSON(w, r, err)
		return
	}

	if len(addTags) == 0 && len(removeTags) == 0 {
		response.ServerErrorJSON(w, r, errors.New(
			"googlreader: add or/and remove tags should be supplied"))
		return
	}

	tags, err := checkAndSimplifyTags(addTags, removeTags)
	if err != nil {
		response.ServerErrorJSON(w, r, err)
		return
	}

	itemIDs, err := parseItemIDsFromRequest(r)
	if err != nil {
		response.BadRequestJSON(w, r, err)
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
		response.ServerErrorJSON(w, r, err)
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
		response.ServerErrorJSON(w, r, err)
		return
	}

	for _, entry := range entries {
		integration.SendEntry(ctx, entry, user)
	}
	response.Text(w, r, "OK")
}

func (h *handler) quickAddHandler(w http.ResponseWriter, r *http.Request,
) (*quickAddResponse, error) {
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
		return nil, response.WrapBadRequest(err)
	}

	feedURL := r.Form.Get(paramQuickAdd)
	if !urllib.IsAbsoluteURL(feedURL) {
		return nil, response.WrapBadRequest(fmt.Errorf(
			"googlereader: invalid URL: %s", feedURL))
	}

	requestBuilder := fetcher.NewRequestBuilder()
	subscriptions, lerr := mfs.
		NewSubscriptionFinder(requestBuilder).
		FindSubscriptions(ctx, feedURL,
			user.Integration().RSSBridgeURLIfEnabled(),
			user.Integration().RSSBridgeTokenIfEnabled())
	if lerr != nil {
		return nil, response.WrapServerError(lerr)
	}

	if len(subscriptions) == 0 {
		return &quickAddResponse{NumResults: 0}, nil
	}

	toSubscribe := Stream{FeedStream, subscriptions[0].URL}
	category := Stream{NoStream, ""}
	feed, err := h.subscribe(ctx, toSubscribe, category, "", user.ID)
	if err != nil {
		return nil, response.WrapServerError(err)
	}

	log.Debug("[GoogleReader] Added a new feed",
		slog.String("feed_url", feed.FeedURL))

	return &quickAddResponse{
		NumResults: 1,
		Query:      feed.FeedURL,
		StreamID:   feedPrefix + strconv.FormatInt(feed.ID, 10),
		StreamName: feed.Title,
	}, nil
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

func (h *handler) subscribe(ctx context.Context, newFeed, category Stream, title string,
	userID int64,
) (*model.Feed, error) {
	destCategory, err := getOrCreateCategory(ctx, category, h.store, userID)
	if err != nil {
		return nil, err
	}

	createRequest := model.FeedCreationRequest{
		FeedURL:    newFeed.ID,
		CategoryID: destCategory.ID,
	}
	lerr := validator.ValidateFeedCreation(ctx, h.store, userID, &createRequest)
	if lerr != nil {
		return nil, lerr.Error()
	}

	feed, lwerr := mff.New(h.store, userID, h.templates).FromRequest(ctx,
		&createRequest)
	if lwerr != nil {
		return nil, lwerr
	}

	if title != "" {
		modifyRequest := model.FeedModificationRequest{
			Title: &title,
		}
		modifyRequest.Patch(feed)
		if err := h.store.UpdateFeed(ctx, feed); err != nil {
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

func move(ctx context.Context, feedStream, labelStream Stream,
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
	if f.Icon == nil || f.Icon.ExternalId() == "" {
		return ""
	}
	return config.RootURL() + route.Path(h.router, "feedIcon", "externalIconID",
		f.Icon.ExternalId())
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
		response.BadRequestJSON(w, r, err)
		return
	}

	streamIds, err := getStreams(r.Form[paramStreamID], userID)
	if err != nil || len(streamIds) == 0 {
		response.BadRequestJSON(w, r, errors.New(
			"googlereader: no valid stream IDs provided"))
		return
	}

	newLabel, err := getStream(r.Form.Get(paramTagsAdd), userID)
	if err != nil {
		response.BadRequestJSON(w, r, fmt.Errorf(
			"googlereader: invalid data in %s", paramTagsAdd))
		return
	}

	title := r.Form.Get(paramTitle)
	action := r.Form.Get(paramSubscribeAction)

	switch action {
	case "subscribe":
		_, err := h.subscribe(ctx, streamIds[0], newLabel, title, userID)
		if err != nil {
			response.ServerErrorJSON(w, r, err)
			return
		}
	case "unsubscribe":
		err := unsubscribe(ctx, streamIds, h.store, userID)
		if err != nil {
			response.ServerErrorJSON(w, r, err)
			return
		}
	case "edit":
		if title != "" {
			err := rename(ctx, streamIds[0], title, h.store, userID)
			if err != nil {
				badRequest := errors.Is(err, errFeedNotFound) ||
					errors.Is(err, errEmptyFeedTitle)
				if badRequest {
					response.BadRequestJSON(w, r, err)
				} else {
					response.ServerErrorJSON(w, r, err)
				}
				return
			}
		}

		if r.Form.Has(paramTagsAdd) {
			if newLabel.Type != LabelStream {
				response.BadRequestJSON(w, r, errors.New("destination must be a label"))
				return
			}

			err := move(ctx, streamIds[0], newLabel, h.store, userID)
			if err != nil {
				badRequest := errors.Is(err, errFeedNotFound) ||
					errors.Is(err, errCategoryNotFound)
				if badRequest {
					response.BadRequestJSON(w, r, err)
				} else {
					response.ServerErrorJSON(w, r, err)
				}
				return
			}
		}
	default:
		response.BadRequestJSON(w, r, fmt.Errorf(
			"googlereader: unrecognized action %s", action))
		return
	}

	response.Text(w, r, "OK")
}

func (h *handler) streamItemContentsHandler(w http.ResponseWriter,
	r *http.Request,
) (*streamContentItemsResponse, error) {
	user := request.User(r)
	ctx := r.Context()
	log := logging.FromContext(ctx).With(
		slog.String("handler", "streamItemContentsHandler"),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", user.ID))
	log.Debug("[GoogleReader] Handle /stream/items/contents")

	if err := checkOutputFormat(r); err != nil {
		return nil, response.WrapBadRequest(err)
	}

	if err := r.ParseForm(); err != nil {
		return nil, response.WrapServerError(err)
	}

	modifiers, err := parseStreamFilterFromRequest(r, user)
	if err != nil {
		return nil, response.WrapServerError(err)
	}

	log.Debug("[GoogleReader] Request Modifiers",
		slog.Any("modifiers", modifiers))

	itemIDs, err := parseItemIDsFromRequest(r)
	if err != nil {
		return nil, response.WrapBadRequest(err)
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
		return nil, response.WrapServerError(err)
	}

	result := &streamContentItemsResponse{
		Direction: "ltr",
		ID:        "user/-/state/com.google/reading-list",
		Title:     "Reading List",
		Updated:   time.Now().Unix(),
		Self: []contentHREF{
			{HREF: config.RootURL() + route.Path(h.router, "StreamItemsContents")},
		},
		Author: user.Username,
	}

	streamPrefix := fmt.Sprintf(userStreamPrefix, user.ID)
	userReadingList := streamPrefix + readingListStreamSuffix
	userRead := streamPrefix + readStreamSuffix
	userStarred := streamPrefix + starredStreamSuffix

	labelPrefix := fmt.Sprintf(userLabelPrefix, user.ID)
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
			categories = append(categories, labelPrefix+entry.Feed.Category.Title)
		}
		if entry.Status == model.EntryStatusRead {
			categories = append(categories, userRead)
		}
		if entry.Starred {
			categories = append(categories, userStarred)
		}

		entry.Content = mediaproxy.RewriteDocumentWithAbsoluteProxyURL(
			h.router, entry.Content)
		mediaproxy.ProxifyEnclosures(h.router, entry.Enclosures())

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
				StreamID: feedPrefix + strconv.FormatInt(entry.FeedID, 10),
				Title:    entry.Feed.Title,
				HTMLUrl:  entry.Feed.SiteURL,
			},
			Enclosure: enclosures,
		}
	}

	result.Items = items
	return result, nil
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
		response.BadRequestJSON(w, r, err)
		return
	}

	streams, err := getStreams(r.Form[paramStreamID], userID)
	if err != nil {
		response.BadRequestJSON(w, r, fmt.Errorf(
			"googlereader: invalid data in %s", paramStreamID))
		return
	}

	titles := make([]string, len(streams))
	for i, stream := range streams {
		if stream.Type != LabelStream {
			response.BadRequestJSON(w, r, errors.New(
				"googlereader: only labels are supported"))
			return
		}
		titles[i] = stream.ID
	}

	err = h.store.RemoveAndReplaceCategoriesByName(ctx, userID, titles)
	if err != nil {
		response.ServerErrorJSON(w, r, err)
		return
	}
	response.Text(w, r, "OK")
}

func (h *handler) renameTagHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := request.UserID(r)

	logging.FromContext(ctx).Debug("[GoogleReader] Handle /rename-tag",
		slog.String("handler", "renameTagHandler"),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))

	if err := r.ParseForm(); err != nil {
		response.BadRequestJSON(w, r, err)
		return
	}

	source, err := getStream(r.Form.Get(paramStreamID), userID)
	if err != nil {
		response.BadRequestJSON(w, r, fmt.Errorf(
			"googlereader: invalid data in %s", paramStreamID))
		return
	}

	destination, err := getStream(r.Form.Get(paramDestination), userID)
	if err != nil {
		response.BadRequestJSON(w, r, fmt.Errorf(
			"googlereader: invalid data in %s", paramDestination))
		return
	}

	if source.Type != LabelStream || destination.Type != LabelStream {
		response.BadRequestJSON(w, r, errors.New(
			"googlereader: only labels supported"))
		return
	} else if destination.ID == "" {
		response.BadRequestJSON(w, r, errors.New(
			"googlereader: empty destination name"))
		return
	}

	category, err := h.store.CategoryByTitle(ctx, userID, source.ID)
	if err != nil {
		response.ServerErrorJSON(w, r, err)
		return
	} else if category == nil {
		response.NotFoundJSON(w, r)
		return
	}

	modifyRequest := model.CategoryModificationRequest{
		Title: new(destination.ID),
	}
	lerr := validator.ValidateCategoryModification(ctx, h.store, userID,
		category.ID, &modifyRequest)
	if lerr != nil {
		response.BadRequestJSON(w, r, lerr.Error())
		return
	}

	modifyRequest.Patch(category)
	affected, err := h.store.UpdateCategory(ctx, category)
	if err != nil {
		response.ServerErrorJSON(w, r, err)
		return
	} else if !affected {
		response.NotFoundJSON(w, r)
		return
	}
	response.Text(w, r, "OK")
}

func (h *handler) tagListHandler(w http.ResponseWriter, r *http.Request,
) (*tagsResponse, error) {
	ctx := r.Context()
	userID := request.UserID(r)

	logging.FromContext(ctx).Debug("[GoogleReader] Handle /tags/list",
		slog.String("handler", "tagListHandler"),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))

	if err := checkOutputFormat(r); err != nil {
		return nil, response.WrapBadRequest(err)
	}

	result := &tagsResponse{}
	categories, err := h.store.Categories(ctx, userID)
	if err != nil {
		return nil, response.WrapServerError(err)
	}

	result.Tags = make([]subscriptionCategoryResponse, 0, len(categories)+1)
	result.Tags = append(result.Tags, subscriptionCategoryResponse{
		ID: fmt.Sprintf(userStreamPrefix, userID) + starredStreamSuffix,
	})

	labelPrefix := fmt.Sprintf(userLabelPrefix, userID)
	for _, category := range categories {
		result.Tags = append(result.Tags, subscriptionCategoryResponse{
			ID:    labelPrefix + category.Title,
			Label: category.Title,
			Type:  "folder",
		})
	}
	return result, nil
}

func (h *handler) subscriptionListHandler(w http.ResponseWriter,
	r *http.Request,
) (*subscriptionsResponse, error) {
	ctx := r.Context()
	userID := request.UserID(r)

	logging.FromContext(ctx).Debug("[GoogleReader] Handle /subscription/list",
		slog.String("handler", "subscriptionListHandler"),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))

	if err := checkOutputFormat(r); err != nil {
		return nil, response.WrapBadRequest(err)
	}

	feeds, err := h.store.Feeds(ctx, userID)
	if err != nil {
		return nil, response.WrapServerError(err)
	}

	result := &subscriptionsResponse{
		Subscriptions: make([]subscriptionResponse, len(feeds)),
	}
	labelPrefix := fmt.Sprintf(userLabelPrefix, userID)
	for i, feed := range feeds {
		result.Subscriptions[i] = subscriptionResponse{
			ID:    feedPrefix + strconv.FormatInt(feed.ID, 10),
			Title: feed.Title,
			URL:   feed.FeedURL,
			Categories: []subscriptionCategoryResponse{
				{
					ID:    labelPrefix + feed.Category.Title,
					Label: feed.Category.Title,
					Type:  "folder",
				},
			},
			HTMLURL: feed.SiteURL,
			IconURL: h.feedIconURL(feed),
		}
	}
	return result, nil
}

func (h *handler) serveHandler(w http.ResponseWriter, r *http.Request,
) ([]string, error) {
	logging.FromContext(r.Context()).Debug(
		"[GoogleReader] API endpoint not implemented yet",
		slog.Any("url", r.RequestURI),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))
	return []string{}, nil
}

func (h *handler) userInfoHandler(w http.ResponseWriter, r *http.Request,
) (*userInfoResponse, error) {
	ctx := r.Context()
	logging.FromContext(ctx).Debug("[GoogleReader] Handle /user-info",
		slog.String("handler", "userInfoHandler"),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))

	if err := checkOutputFormat(r); err != nil {
		return nil, response.WrapBadRequest(err)
	}

	user := request.User(r)
	userId := strconv.FormatInt(user.ID, 10)
	userInfo := &userInfoResponse{
		UserID:        userId,
		UserName:      user.Username,
		UserProfileID: userId,
		UserEmail:     user.Username,
	}
	return userInfo, nil
}

func (h *handler) streamItemIDsHandler(w http.ResponseWriter, r *http.Request,
) (*streamIDResponse, error) {
	ctx := r.Context()
	user := request.User(r)

	log := logging.FromContext(ctx).With(
		slog.String("handler", "streamItemIDsHandler"),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", user.ID))
	log.Debug("[GoogleReader] Handle /stream/items/ids")

	if err := checkOutputFormat(r); err != nil {
		return nil, response.WrapBadRequest(err)
	}

	modifiers, err := parseStreamFilterFromRequest(r, user)
	if err != nil {
		return nil, response.WrapServerError(err)
	}
	log.Debug("[GoogleReader] Request Modifiers",
		slog.Any("modifiers", modifiers))

	if len(modifiers.Streams) != 1 {
		return nil, response.WrapServerError(errors.New(
			"googlereader: only one stream type expected"))
	}

	switch modifiers.Streams[0].Type {
	case ReadingListStream:
		return h.handleReadingListStreamHandler(r, modifiers)
	case StarredStream:
		return h.handleStarredStreamHandler(r, modifiers)
	case ReadStream:
		return h.handleReadStreamHandler(r, modifiers)
	case FeedStream:
		return h.handleFeedStreamHandler(r, modifiers)
	default:
	}

	log.Warn("[GoogleReader] Unknown Stream",
		slog.Any("stream_type", modifiers.Streams[0].Type))
	return nil, response.WrapServerError(fmt.Errorf(
		"googlereader: unknown stream type %s", modifiers.Streams[0].Type))
}

func (h *handler) handleReadingListStreamHandler(r *http.Request,
	rm RequestModifiers,
) (*streamIDResponse, error) {
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
		return nil, response.WrapServerError(err)
	}
	return &streamId, nil
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
		return err
	})

	var totalEntries int
	g.Go(func() (err error) {
		totalEntries, err = builder.CountEntries(ctx)
		return err
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

func (h *handler) handleStarredStreamHandler(r *http.Request,
	rm RequestModifiers,
) (*streamIDResponse, error) {
	builder := h.store.NewEntryQueryBuilder(rm.UserID).
		WithoutStatus(model.EntryStatusRemoved).
		WithStarred(true)

	streamId, err := makeStreamIDResp(r.Context(), builder, &rm)
	if err != nil {
		return nil, response.WrapServerError(err)
	}
	return &streamId, nil
}

func (h *handler) handleReadStreamHandler(r *http.Request, rm RequestModifiers,
) (*streamIDResponse, error) {
	builder := h.store.NewEntryQueryBuilder(rm.UserID).
		WithoutStatus(model.EntryStatusRemoved).
		WithStatus(model.EntryStatusRead)

	streamId, err := makeStreamIDResp(r.Context(), builder, &rm)
	if err != nil {
		return nil, response.WrapServerError(err)
	}
	return &streamId, nil
}

func (h *handler) handleFeedStreamHandler(r *http.Request, rm RequestModifiers,
) (*streamIDResponse, error) {
	id, err := strconv.ParseInt(rm.Streams[0].ID, 10, 64)
	if err != nil {
		return nil, response.WrapServerError(err)
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
		return nil, response.WrapServerError(err)
	}
	return &streamId, nil
}

func (h *handler) markAllAsReadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logging.FromContext(ctx).Debug("[GoogleReader] Handle /mark-all-as-read",
		slog.String("handler", "markAllAsReadHandler"),
		slog.String("client_ip", request.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()))

	if err := r.ParseForm(); err != nil {
		response.BadRequestJSON(w, r, err)
		return
	}

	userID := request.UserID(r)
	stream, err := getStream(r.Form.Get(paramStreamID), userID)
	if err != nil {
		response.BadRequestJSON(w, r, err)
		return
	}

	var before time.Time
	if timestampString := r.Form.Get(paramTimestamp); timestampString != "" {
		ts, err := strconv.ParseInt(timestampString, 10, 64)
		if err != nil {
			response.BadRequestJSON(w, r, err)
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
			response.BadRequestJSON(w, r, err)
			return
		}
		affected, err := h.store.MarkFeedAsRead(ctx, userID, feedID, before)
		if err != nil {
			response.ServerErrorJSON(w, r, err)
			return
		} else if !affected {
			response.NotFoundJSON(w, r)
			return
		}
	case LabelStream:
		category, err := h.store.CategoryByTitle(ctx, userID, stream.ID)
		if err != nil {
			response.ServerErrorJSON(w, r, err)
			return
		} else if category == nil {
			response.NotFoundJSON(w, r)
			return
		}
		affected, err := h.store.MarkCategoryAsRead(ctx, userID, category.ID,
			before)
		if err != nil {
			response.ServerErrorJSON(w, r, err)
			return
		} else if !affected {
			response.NotFoundJSON(w, r)
			return
		}
	case ReadingListStream:
		err := h.store.MarkAllAsReadBeforeDate(ctx, userID, before)
		if err != nil {
			response.ServerErrorJSON(w, r, err)
			return
		}
	}

	response.Text(w, r, "OK")
}
