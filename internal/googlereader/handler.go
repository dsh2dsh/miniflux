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

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/integration"
	"miniflux.app/v2/internal/mediaproxy"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/proxyrotator"
	"miniflux.app/v2/internal/reader/fetcher"
	mff "miniflux.app/v2/internal/reader/handler"
	mfs "miniflux.app/v2/internal/reader/subscription"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/validator"

	"github.com/gorilla/mux"
)

type handler struct {
	store  *storage.Storage
	router *mux.Router
}

var (
	errEmptyFeedTitle   = errors.New("googlereader: empty feed title")
	errFeedNotFound     = errors.New("googlereader: feed not found")
	errCategoryNotFound = errors.New("googlereader: category not found")
)

// Serve handles Google Reader API calls.
func Serve(router *mux.Router, store *storage.Storage) {
	handler := &handler{store, router}
	router.HandleFunc("/accounts/ClientLogin", handler.clientLoginHandler).Methods(http.MethodPost).Name("ClientLogin")

	middleware := newMiddleware(store)
	sr := router.PathPrefix("/reader/api/0").Subrouter()
	sr.Use(middleware.handleCORS)
	sr.Use(middleware.apiKeyAuth)
	sr.Methods(http.MethodOptions)
	sr.HandleFunc("/token", handler.tokenHandler).Methods(http.MethodGet).Name("Token")
	sr.HandleFunc("/edit-tag", handler.editTagHandler).Methods(http.MethodPost).Name("EditTag")
	sr.HandleFunc("/rename-tag", handler.renameTagHandler).Methods(http.MethodPost).Name("Rename Tag")
	sr.HandleFunc("/disable-tag", handler.disableTagHandler).Methods(http.MethodPost).Name("Disable Tag")
	sr.HandleFunc("/tag/list", handler.tagListHandler).Methods(http.MethodGet).Name("TagList")
	sr.HandleFunc("/user-info", handler.userInfoHandler).Methods(http.MethodGet).Name("UserInfo")
	sr.HandleFunc("/subscription/list", handler.subscriptionListHandler).Methods(http.MethodGet).Name("SubscriptonList")
	sr.HandleFunc("/subscription/edit", handler.editSubscriptionHandler).Methods(http.MethodPost).Name("SubscriptionEdit")
	sr.HandleFunc("/subscription/quickadd", handler.quickAddHandler).Methods(http.MethodPost).Name("QuickAdd")
	sr.HandleFunc("/stream/items/ids", handler.streamItemIDsHandler).Methods(http.MethodGet).Name("StreamItemIDs")
	sr.HandleFunc("/stream/items/contents", handler.streamItemContentsHandler).Methods(http.MethodPost).Name("StreamItemsContents")
	sr.HandleFunc("/mark-all-as-read", handler.markAllAsReadHandler).Methods(http.MethodPost).Name("MarkAllAsRead")
	sr.PathPrefix("/").HandlerFunc(handler.serveHandler).Methods(http.MethodPost, http.MethodGet).Name("GoogleReaderApiEndpoint")
}

func checkAndSimplifyTags(addTags []Stream, removeTags []Stream) (map[StreamType]bool, error) {
	tags := make(map[StreamType]bool)
	for _, s := range addTags {
		switch s.Type {
		case ReadStream:
			if _, ok := tags[KeptUnreadStream]; ok {
				return nil, fmt.Errorf("googlereader: %s and %s should not be supplied simultaneously", KeptUnread, Read)
			}
			tags[ReadStream] = true
		case KeptUnreadStream:
			if _, ok := tags[ReadStream]; ok {
				return nil, fmt.Errorf("googlereader: %s and %s should not be supplied simultaneously", KeptUnread, Read)
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
				return nil, fmt.Errorf("googlereader: %s and %s should not be supplied simultaneously", KeptUnread, Read)
			}
			tags[ReadStream] = false
		case KeptUnreadStream:
			if _, ok := tags[ReadStream]; ok {
				return nil, fmt.Errorf("googlereader: %s and %s should not be supplied simultaneously", KeptUnread, Read)
			}
			tags[ReadStream] = true
		case StarredStream:
			if _, ok := tags[StarredStream]; ok {
				return nil, fmt.Errorf("googlereader: %s should not be supplied for add and remove simultaneously", Starred)
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

func (h *handler) clientLoginHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := request.ClientIP(r)

	slog.Debug("[GoogleReader] Handle /accounts/ClientLogin",
		slog.String("handler", "clientLoginHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
	)

	if err := r.ParseForm(); err != nil {
		slog.Warn("[GoogleReader] Could not parse request form data",
			slog.Bool("authentication_failed", true),
			slog.String("client_ip", clientIP),
			slog.String("user_agent", r.UserAgent()),
			slog.Any("error", err),
		)
		json.Unauthorized(w, r)
		return
	}

	username := r.Form.Get("Email")
	password := r.Form.Get("Passwd")
	output := r.Form.Get("output")

	if username == "" || password == "" {
		slog.Warn("[GoogleReader] Empty username or password",
			slog.Bool("authentication_failed", true),
			slog.String("client_ip", clientIP),
			slog.String("user_agent", r.UserAgent()),
		)
		json.Unauthorized(w, r)
		return
	}

	user, err := h.store.UserByUsername(r.Context(), username)
	if err != nil {
		slog.Warn("[GoogleReader] Invalid username or password",
			slog.Bool("authentication_failed", true),
			slog.String("client_ip", clientIP),
			slog.String("user_agent", r.UserAgent()),
			slog.String("username", username),
			slog.Any("error", err),
		)
		json.Unauthorized(w, r)
		return
	} else if user == nil || !user.Integration().GoogleReaderEnabled {
		slog.Warn("[GoogleReader] Invalid username or password",
			slog.Bool("authentication_failed", true),
			slog.String("client_ip", clientIP),
			slog.String("user_agent", r.UserAgent()),
			slog.String("username", username),
			slog.Any("error", errors.New("googlereader: unable to find user")),
		)
		json.Unauthorized(w, r)
		return
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Integration().GoogleReaderPassword),
		[]byte(password))
	if err != nil {
		slog.Warn("[GoogleReader] Invalid username or password",
			slog.Bool("authentication_failed", true),
			slog.String("client_ip", clientIP),
			slog.String("user_agent", r.UserAgent()),
			slog.String("username", username),
			slog.Any("error", err),
		)
		json.Unauthorized(w, r)
		return
	}

	slog.Info("[GoogleReader] User authenticated successfully",
		slog.Bool("authentication_successful", true),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.String("username", username),
	)

	err = h.store.SetLastLogin(r.Context(), user.ID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	token := getAuthToken(user.Username, user.Integration().GoogleReaderPassword)
	slog.Debug("[GoogleReader] Created token",
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.String("username", username),
	)

	result := login{SID: token, LSID: token, Auth: token}
	if output == "json" {
		json.OK(w, r, result)
		return
	}

	builder := response.New(w, r)
	builder.WithHeader("Content-Type", "text/plain; charset=UTF-8")
	builder.WithBody(result.String())
	builder.Write()
}

func (h *handler) tokenHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := request.ClientIP(r)

	slog.Debug("[GoogleReader] Handle /token",
		slog.String("handler", "tokenHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
	)

	if !request.IsAuthenticated(r) {
		slog.Warn("[GoogleReader] User is not authenticated",
			slog.String("client_ip", clientIP),
			slog.String("user_agent", r.UserAgent()),
		)
		json.Unauthorized(w, r)
		return
	}

	token := request.GoolgeReaderToken(r)
	if token == "" {
		slog.Warn("[GoogleReader] User does not have token",
			slog.String("client_ip", clientIP),
			slog.String("user_agent", r.UserAgent()),
			slog.Int64("user_id", request.UserID(r)),
		)
		json.Unauthorized(w, r)
		return
	}

	slog.Debug("[GoogleReader] Token handler",
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", request.UserID(r)),
		slog.String("token", token),
	)

	w.Header().Add("Content-Type", "text/plain; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(token)); err != nil {
		json.ServerError(w, r, err)
	}
}

func (h *handler) editTagHandler(w http.ResponseWriter, r *http.Request) {
	user := request.User(r)
	userID := request.UserID(r)
	clientIP := request.ClientIP(r)

	slog.Debug("[GoogleReader] Handle /edit-tag",
		slog.String("handler", "editTagHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", userID),
	)

	if err := r.ParseForm(); err != nil {
		json.ServerError(w, r, err)
		return
	}

	addTags, err := getStreams(r.PostForm[ParamTagsAdd], userID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	removeTags, err := getStreams(r.PostForm[ParamTagsRemove], userID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	if len(addTags) == 0 && len(removeTags) == 0 {
		err = errors.New("googlreader: add or/and remove tags should be supplied")
		json.ServerError(w, r, err)
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

	slog.Debug("[GoogleReader] Edited tags",
		slog.String("handler", "editTagHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", userID),
		slog.Any("item_ids", itemIDs),
		slog.Any("tags", tags),
	)

	builder := h.store.NewEntryQueryBuilder(userID)
	builder.WithEntryIDs(itemIDs)
	builder.WithoutStatus(model.EntryStatusRemoved)

	entries, err := builder.GetEntries(r.Context())
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	n := 0
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
	if len(readEntryIDs) > 0 {
		err = h.store.SetEntriesStatus(r.Context(), userID, readEntryIDs,
			model.EntryStatusRead)
		if err != nil {
			json.ServerError(w, r, err)
			return
		}
	}

	if len(unreadEntryIDs) > 0 {
		err = h.store.SetEntriesStatus(r.Context(), userID, unreadEntryIDs,
			model.EntryStatusUnread)
		if err != nil {
			json.ServerError(w, r, err)
			return
		}
	}

	if len(unstarredEntryIDs) > 0 {
		err = h.store.SetEntriesBookmarkedState(r.Context(), userID,
			unstarredEntryIDs, false)
		if err != nil {
			json.ServerError(w, r, err)
			return
		}
	}

	if len(starredEntryIDs) > 0 {
		err = h.store.SetEntriesBookmarkedState(r.Context(), userID,
			starredEntryIDs, true)
		if err != nil {
			json.ServerError(w, r, err)
			return
		}
	}

	if len(entries) > 0 {
		for _, entry := range entries {
			e := entry
			integration.SendEntry(e, user)
		}
	}

	OK(w, r)
}

func (h *handler) quickAddHandler(w http.ResponseWriter, r *http.Request) {
	user := request.User(r)
	userID := request.UserID(r)
	clientIP := request.ClientIP(r)

	slog.Debug("[GoogleReader] Handle /subscription/quickadd",
		slog.String("handler", "quickAddHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", userID),
	)

	err := r.ParseForm()
	if err != nil {
		json.BadRequest(w, r, err)
		return
	}

	feedURL := r.Form.Get(ParamQuickAdd)
	if !validator.IsValidURL(feedURL) {
		json.BadRequest(w, r, fmt.Errorf("googlereader: invalid URL: %s", feedURL))
		return
	}

	requestBuilder := fetcher.NewRequestBuilder()
	requestBuilder.WithTimeout(config.Opts.HTTPClientTimeout())
	requestBuilder.WithProxyRotator(proxyrotator.ProxyRotatorInstance)

	subscriptions, localizedError := mfs.
		NewSubscriptionFinder(requestBuilder).
		FindSubscriptions(r.Context(), feedURL,
			user.Integration().RSSBridgeURLIfEnabled(),
			user.Integration().RSSBridgeTokenIfEnabled())
	if localizedError != nil {
		json.ServerError(w, r, localizedError)
		return
	}

	if len(subscriptions) == 0 {
		json.OK(w, r, quickAddResponse{
			NumResults: 0,
		})
		return
	}

	toSubscribe := Stream{FeedStream, subscriptions[0].URL}
	category := Stream{NoStream, ""}
	newFeed, err := subscribe(r.Context(), toSubscribe, category, "", h.store,
		userID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	slog.Debug("[GoogleReader] Added a new feed",
		slog.String("handler", "quickAddHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", userID),
		slog.String("feed_url", newFeed.FeedURL),
	)

	json.OK(w, r, quickAddResponse{
		NumResults: 1,
		Query:      newFeed.FeedURL,
		StreamID:   fmt.Sprintf(FeedPrefix+"%d", newFeed.ID),
		StreamName: newFeed.Title,
	})
}

func getFeed(ctx context.Context, stream Stream, store *storage.Storage,
	userID int64,
) (*model.Feed, error) {
	feedID, err := strconv.ParseInt(stream.ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("googlereader: %w", err)
	}
	return store.FeedByID(ctx, userID, feedID)
}

func getOrCreateCategory(ctx context.Context, streamCategory Stream,
	store *storage.Storage, userID int64,
) (*model.Category, error) {
	switch {
	case streamCategory.ID == "":
		return store.FirstCategory(ctx, userID)
	case store.CategoryTitleExists(ctx, userID, streamCategory.ID):
		return store.CategoryByTitle(ctx, userID, streamCategory.ID)
	default:
		return store.CreateCategory(ctx, userID, &model.CategoryCreationRequest{
			Title: streamCategory.ID,
		})
	}
}

func subscribe(ctx context.Context, newFeed Stream, category Stream,
	title string, store *storage.Storage, userID int64,
) (*model.Feed, error) {
	destCategory, err := getOrCreateCategory(ctx, category, store, userID)
	if err != nil {
		return nil, err
	}

	feedRequest := model.FeedCreationRequest{
		FeedURL:    newFeed.ID,
		CategoryID: destCategory.ID,
	}
	verr := validator.ValidateFeedCreation(ctx, store, userID, &feedRequest)
	if verr != nil {
		return nil, verr.Error()
	}

	created, localizedError := mff.CreateFeed(ctx, store, userID, &feedRequest)
	if localizedError != nil {
		return nil, localizedError
	}

	if title != "" {
		feedModification := model.FeedModificationRequest{
			Title: &title,
		}
		feedModification.Patch(created)
		if err := store.UpdateFeed(ctx, created); err != nil {
			return nil, err
		}
	}

	return created, nil
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
	slog.Debug("[GoogleReader] Renaming feed",
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

	feedModification := model.FeedModificationRequest{Title: &title}
	feedModification.Patch(feed)
	return store.UpdateFeed(ctx, feed)
}

func move(ctx context.Context, feedStream Stream, labelStream Stream,
	store *storage.Storage, userID int64,
) error {
	slog.Debug("[GoogleReader] Moving feed",
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

	feedModification := model.FeedModificationRequest{CategoryID: &category.ID}
	feedModification.Patch(feed)
	return store.UpdateFeed(ctx, feed)
}

func (h *handler) feedIconURL(f *model.Feed) string {
	if f.Icon != nil && f.Icon.ExternalIconID != "" {
		return config.Opts.RootURL() + route.Path(h.router, "feedIcon", "externalIconID", f.Icon.ExternalIconID)
	} else {
		return ""
	}
}

func (h *handler) editSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	clientIP := request.ClientIP(r)

	slog.Debug("[GoogleReader] Handle /subscription/edit",
		slog.String("handler", "editSubscriptionHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", userID),
	)

	if err := r.ParseForm(); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	streamIds, err := getStreams(r.Form[ParamStreamID], userID)
	if err != nil || len(streamIds) == 0 {
		json.BadRequest(w, r, errors.New("googlereader: no valid stream IDs provided"))
		return
	}

	newLabel, err := getStream(r.Form.Get(ParamTagsAdd), userID)
	if err != nil {
		json.BadRequest(w, r, fmt.Errorf("googlereader: invalid data in %s", ParamTagsAdd))
		return
	}

	title := r.Form.Get(ParamTitle)
	action := r.Form.Get(ParamSubscribeAction)

	switch action {
	case "subscribe":
		_, err := subscribe(r.Context(), streamIds[0], newLabel, title, h.store, userID)
		if err != nil {
			json.ServerError(w, r, err)
			return
		}
	case "unsubscribe":
		err := unsubscribe(r.Context(), streamIds, h.store, userID)
		if err != nil {
			json.ServerError(w, r, err)
			return
		}
	case "edit":
		if title != "" {
			err := rename(r.Context(), streamIds[0], title, h.store, userID)
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

		if r.Form.Has(ParamTagsAdd) {
			if newLabel.Type != LabelStream {
				json.BadRequest(w, r, errors.New("destination must be a label"))
				return
			}

			err := move(r.Context(), streamIds[0], newLabel, h.store, userID)
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

	OK(w, r)
}

func (h *handler) streamItemContentsHandler(w http.ResponseWriter,
	r *http.Request,
) {
	userID, clientIP := request.UserID(r), request.ClientIP(r)

	slog.Debug("[GoogleReader] Handle /stream/items/contents",
		slog.String("handler", "streamItemContentsHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", userID),
	)

	if err := checkOutputFormat(r); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	if err := r.ParseForm(); err != nil {
		json.ServerError(w, r, err)
		return
	}

	user, err := h.store.UserByID(r.Context(), userID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	requestModifiers, err := parseStreamFilterFromRequest(r, user)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	slog.Debug("[GoogleReader] Request Modifiers",
		slog.String("handler", "streamItemContentsHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.Any("modifiers", requestModifiers),
	)

	userReadingList := fmt.Sprintf(UserStreamPrefix, userID) + ReadingList
	userRead := fmt.Sprintf(UserStreamPrefix, userID) + Read
	userStarred := fmt.Sprintf(UserStreamPrefix, userID) + Starred

	itemIDs, err := parseItemIDsFromRequest(r)
	if err != nil {
		json.BadRequest(w, r, err)
		return
	}

	slog.Debug("[GoogleReader] Fetching item contents",
		slog.String("handler", "streamItemContentsHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", userID),
		slog.Any("item_ids", itemIDs),
	)

	builder := h.store.NewEntryQueryBuilder(userID).
		WithContent().
		WithEnclosures().
		WithoutStatus(model.EntryStatusRemoved).
		WithEntryIDs(itemIDs).
		WithSorting(model.DefaultSortingOrder, requestModifiers.SortDirection)

	entries, err := builder.GetEntries(r.Context())
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	result := streamContentItems{
		Direction: "ltr",
		ID:        "user/-/state/com.google/reading-list",
		Title:     "Reading List",
		Updated:   time.Now().Unix(),
		Self: []contentHREF{
			{HREF: config.Opts.RootURL() + route.Path(h.router, "StreamItemsContents")},
		},
		Author: user.Username,
	}

	contentItems := make([]contentItem, len(entries))
	for i, entry := range entries {
		enclosures := make([]contentItemEnclosure, 0, len(entry.Enclosures))
		for _, enclosure := range entry.Enclosures {
			enclosures = append(enclosures,
				contentItemEnclosure{URL: enclosure.URL, Type: enclosure.MimeType})
		}

		categories := []string{userReadingList}
		if entry.Feed.Category.Title != "" {
			categories = append(categories,
				fmt.Sprintf(UserLabelPrefix, userID)+entry.Feed.Category.Title)
		}
		if entry.Status == model.EntryStatusRead {
			categories = append(categories, userRead)
		}
		if entry.Starred {
			categories = append(categories, userStarred)
		}

		entry.Content = mediaproxy.RewriteDocumentWithAbsoluteProxyURL(
			h.router, entry.Content)
		entry.Enclosures.ProxifyEnclosureURL(h.router)

		contentItems[i] = contentItem{
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
			Content:       contentItemContent{Direction: "ltr", Content: entry.Content},
			Summary:       contentItemContent{Direction: "ltr", Content: entry.Content},
			Origin: contentItemOrigin{
				StreamID: fmt.Sprintf("feed/%d", entry.FeedID),
				Title:    entry.Feed.Title,
				HTMLUrl:  entry.Feed.SiteURL,
			},
			Enclosure: enclosures,
		}
	}

	result.Items = contentItems
	json.OK(w, r, result)
}

func (h *handler) disableTagHandler(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	clientIP := request.ClientIP(r)

	slog.Debug("[GoogleReader] Handle /disable-tags",
		slog.String("handler", "disableTagHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", userID),
	)

	err := r.ParseForm()
	if err != nil {
		json.BadRequest(w, r, err)
		return
	}

	streams, err := getStreams(r.Form[ParamStreamID], userID)
	if err != nil {
		json.BadRequest(w, r, fmt.Errorf("googlereader: invalid data in %s", ParamStreamID))
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

	err = h.store.RemoveAndReplaceCategoriesByName(r.Context(), userID, titles)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	OK(w, r)
}

func (h *handler) renameTagHandler(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	clientIP := request.ClientIP(r)

	slog.Debug("[GoogleReader] Handle /rename-tag",
		slog.String("handler", "renameTagHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
	)

	err := r.ParseForm()
	if err != nil {
		json.BadRequest(w, r, err)
		return
	}

	source, err := getStream(r.Form.Get(ParamStreamID), userID)
	if err != nil {
		json.BadRequest(w, r, fmt.Errorf("googlereader: invalid data in %s", ParamStreamID))
		return
	}

	destination, err := getStream(r.Form.Get(ParamDestination), userID)
	if err != nil {
		json.BadRequest(w, r, fmt.Errorf("googlereader: invalid data in %s", ParamDestination))
		return
	}

	if source.Type != LabelStream || destination.Type != LabelStream {
		json.BadRequest(w, r, errors.New("googlereader: only labels supported"))
		return
	}

	if destination.ID == "" {
		json.BadRequest(w, r, errors.New("googlereader: empty destination name"))
		return
	}

	category, err := h.store.CategoryByTitle(r.Context(), userID, source.ID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	if category == nil {
		json.NotFound(w, r)
		return
	}

	categoryModificationRequest := model.CategoryModificationRequest{
		Title: model.SetOptionalField(destination.ID),
	}

	validationError := validator.ValidateCategoryModification(r.Context(),
		h.store, userID, category.ID, &categoryModificationRequest)
	if validationError != nil {
		json.BadRequest(w, r, validationError.Error())
		return
	}

	categoryModificationRequest.Patch(category)

	affected, err := h.store.UpdateCategory(r.Context(), category)
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if !affected {
		json.NotFound(w, r)
		return
	}

	OK(w, r)
}

func (h *handler) tagListHandler(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	clientIP := request.ClientIP(r)

	slog.Debug("[GoogleReader] Handle /tags/list",
		slog.String("handler", "tagListHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
	)

	if err := checkOutputFormat(r); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	var result tagsResponse
	categories, err := h.store.Categories(r.Context(), userID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	result.Tags = make([]subscriptionCategory, 0)
	result.Tags = append(result.Tags, subscriptionCategory{
		ID: fmt.Sprintf(UserStreamPrefix, userID) + Starred,
	})
	for _, category := range categories {
		result.Tags = append(result.Tags, subscriptionCategory{
			ID:    fmt.Sprintf(UserLabelPrefix, userID) + category.Title,
			Label: category.Title,
			Type:  "folder",
		})
	}
	json.OK(w, r, result)
}

func (h *handler) subscriptionListHandler(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	clientIP := request.ClientIP(r)

	slog.Debug("[GoogleReader] Handle /subscription/list",
		slog.String("handler", "subscriptionListHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
	)

	if err := checkOutputFormat(r); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	var result subscriptionsResponse
	feeds, err := h.store.Feeds(r.Context(), userID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	result.Subscriptions = make([]subscription, 0)
	for _, feed := range feeds {
		result.Subscriptions = append(result.Subscriptions, subscription{
			ID:         fmt.Sprintf(FeedPrefix+"%d", feed.ID),
			Title:      feed.Title,
			URL:        feed.FeedURL,
			Categories: []subscriptionCategory{{fmt.Sprintf(UserLabelPrefix, userID) + feed.Category.Title, feed.Category.Title, "folder"}},
			HTMLURL:    feed.SiteURL,
			IconURL:    h.feedIconURL(feed),
		})
	}
	json.OK(w, r, result)
}

func (h *handler) serveHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := request.ClientIP(r)

	slog.Debug("[GoogleReader] API endpoint not implemented yet",
		slog.Any("url", r.RequestURI),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
	)

	json.OK(w, r, []string{})
}

func (h *handler) userInfoHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := request.ClientIP(r)

	slog.Debug("[GoogleReader] Handle /user-info",
		slog.String("handler", "userInfoHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
	)

	if err := checkOutputFormat(r); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	user, err := h.store.UserByID(r.Context(), request.UserID(r))
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	userId := strconv.FormatInt(user.ID, 10)
	userInfo := userInfo{
		UserID:        userId,
		UserName:      user.Username,
		UserProfileID: userId,
		UserEmail:     user.Username,
	}
	json.OK(w, r, userInfo)
}

func (h *handler) streamItemIDsHandler(w http.ResponseWriter, r *http.Request) {
	userID, clientIP := request.UserID(r), request.ClientIP(r)

	slog.Debug("[GoogleReader] Handle /stream/items/ids",
		slog.String("handler", "streamItemIDsHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.Int64("user_id", userID),
	)

	if err := checkOutputFormat(r); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	user, err := h.store.UserByID(r.Context(), userID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	rm, err := parseStreamFilterFromRequest(r, user)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	slog.Debug("[GoogleReader] Request Modifiers",
		slog.String("handler", "streamItemIDsHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.Any("modifiers", rm),
	)

	if len(rm.Streams) != 1 {
		json.ServerError(w, r, errors.New(
			"googlereader: only one stream type expected"))
		return
	}

	switch rm.Streams[0].Type {
	case ReadingListStream:
		h.handleReadingListStreamHandler(w, r, rm)
	case StarredStream:
		h.handleStarredStreamHandler(w, r, rm)
	case ReadStream:
		h.handleReadStreamHandler(w, r, rm)
	case FeedStream:
		h.handleFeedStreamHandler(w, r, rm)
	default:
		slog.Warn("[GoogleReader] Unknown Stream",
			slog.String("handler", "streamItemIDsHandler"),
			slog.String("client_ip", clientIP),
			slog.String("user_agent", r.UserAgent()),
			slog.Any("stream_type", rm.Streams[0].Type),
		)
		json.ServerError(w, r, fmt.Errorf(
			"googlereader: unknown stream type %s", rm.Streams[0].Type))
	}
}

func (h *handler) handleReadingListStreamHandler(w http.ResponseWriter,
	r *http.Request, rm RequestModifiers,
) {
	clientIP := request.ClientIP(r)

	slog.Debug("[GoogleReader] Handle ReadingListStream",
		slog.String("handler", "handleReadingListStreamHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
	)

	builder := h.store.NewEntryQueryBuilder(rm.UserID)
	for _, s := range rm.ExcludeTargets {
		switch s.Type {
		case ReadStream:
			builder.WithStatus(model.EntryStatusUnread)
		default:
			slog.Warn("[GoogleReader] Unknown ExcludeTargets filter type",
				slog.String("handler", "handleReadingListStreamHandler"),
				slog.String("client_ip", clientIP),
				slog.String("user_agent", r.UserAgent()),
				slog.Any("filter_type", s.Type),
			)
		}
	}

	builder.WithoutStatus(model.EntryStatusRemoved)
	streamId, err := makeStreamIDResp(r.Context(), builder, &rm)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.OK(w, r, streamId)
}

func makeStreamIDResp(ctx context.Context, builder *storage.EntryQueryBuilder,
	rm *RequestModifiers,
) (streamIDResponse, error) {
	builder.WithLimit(rm.Count).WithOffset(rm.Offset).
		WithSorting(model.DefaultSortingOrder, rm.SortDirection)
	if rm.StartTime > 0 {
		builder.AfterPublishedDate(time.Unix(rm.StartTime, 0))
	}
	if rm.StopTime > 0 {
		builder.BeforePublishedDate(time.Unix(rm.StopTime, 0))
	}

	rawEntryIDs, err := builder.GetEntryIDs(ctx)
	if err != nil {
		return streamIDResponse{}, err
	}

	itemRefs := make([]itemRef, len(rawEntryIDs))
	for i, entryID := range rawEntryIDs {
		itemRefs[i] = itemRef{ID: strconv.FormatInt(entryID, 10)}
	}

	totalEntries, err := builder.CountEntries(ctx)
	if err != nil {
		return streamIDResponse{}, err
	}

	continuation := 0
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
	feedID, err := strconv.ParseInt(rm.Streams[0].ID, 10, 64)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	builder := h.store.NewEntryQueryBuilder(rm.UserID).
		WithoutStatus(model.EntryStatusRemoved).
		WithFeedID(feedID)

	if len(rm.ExcludeTargets) > 0 {
		for _, s := range rm.ExcludeTargets {
			if s.Type == ReadStream {
				builder.WithoutStatus(model.EntryStatusRead)
			}
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
	userID := request.UserID(r)
	clientIP := request.ClientIP(r)

	slog.Debug("[GoogleReader] Handle /mark-all-as-read",
		slog.String("handler", "markAllAsReadHandler"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
	)

	if err := r.ParseForm(); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	stream, err := getStream(r.Form.Get(ParamStreamID), userID)
	if err != nil {
		json.BadRequest(w, r, err)
		return
	}

	var before time.Time
	if timestampParamValue := r.Form.Get(ParamTimestamp); timestampParamValue != "" {
		timestampParsedValue, err := strconv.ParseInt(timestampParamValue, 10, 64)
		if err != nil {
			json.BadRequest(w, r, err)
			return
		}

		if timestampParsedValue > 0 {
			// It's unclear if the timestamp is in seconds or microseconds, so we try both using a naive approach.
			if len(timestampParamValue) >= 16 {
				before = time.UnixMicro(timestampParsedValue)
			} else {
				before = time.Unix(timestampParsedValue, 0)
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
		affected, err := h.store.MarkFeedAsRead(r.Context(), userID, feedID, before)
		if err != nil {
			json.ServerError(w, r, err)
			return
		} else if !affected {
			json.NotFound(w, r)
			return
		}
	case LabelStream:
		category, err := h.store.CategoryByTitle(r.Context(), userID, stream.ID)
		if err != nil {
			json.ServerError(w, r, err)
			return
		} else if category == nil {
			json.NotFound(w, r)
			return
		}
		affected, err := h.store.MarkCategoryAsRead(r.Context(), userID, category.ID, before)
		if err != nil {
			json.ServerError(w, r, err)
			return
		} else if !affected {
			json.NotFound(w, r)
			return
		}
	case ReadingListStream:
		err := h.store.MarkAllAsReadBeforeDate(r.Context(), userID, before)
		if err != nil {
			json.ServerError(w, r, err)
			return
		}
	}
	OK(w, r)
}
