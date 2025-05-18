package ui

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/ui/session"
	"miniflux.app/v2/internal/ui/view"
)

func (h *handler) View(r *http.Request) *View {
	s := session.New(h.store, request.SessionID(r))
	self := &View{
		View:    view.New(h.tpl, r, s),
		session: s,
		store:   h.store,
		userID:  request.UserID(r),
	}
	self.g, self.ctx = errgroup.WithContext(r.Context())
	return self.init()
}

type View struct {
	*view.View

	session *session.Session
	store   *storage.Storage
	userID  int64
	user    *model.User

	g   *errgroup.Group
	ctx context.Context

	startTime            time.Time
	countUnreadElapsed   time.Duration
	preProcessingElapsed time.Duration
	renderingElapsed     time.Duration

	countUnread     int
	countErrorFeeds int
	hasSaveEntry    bool
}

func (self *View) init() *View {
	self.startTime = time.Now()

	self.Go(func(ctx context.Context) (err error) {
		self.user, err = self.store.UserByID(ctx, self.userID)
		return
	})

	startCountUnread := time.Now()
	self.Go(func(ctx context.Context) (err error) {
		self.countUnread, err = self.store.NewEntryQueryBuilder(self.userID).
			WithStatus(model.EntryStatusUnread).
			WithGloballyVisible().
			CountEntries(ctx)
		self.countUnreadElapsed = time.Since(startCountUnread)
		return
	})

	self.Go(func(ctx context.Context) error {
		self.countErrorFeeds = self.store.CountUserFeedsWithErrors(ctx, self.userID)
		return nil
	})
	return self
}

func (self *View) Go(fn func(ctx context.Context) error) {
	self.g.Go(func() error { return fn(self.ctx) })
}

func (self *View) WithSaveEntry() *View {
	self.Go(func(ctx context.Context) error {
		self.hasSaveEntry = self.store.HasSaveEntry(ctx, self.userID)
		return nil
	})
	return self
}

func (self *View) Wait() error {
	if err := self.g.Wait(); err != nil {
		return fmt.Errorf("group error: %w", err)
	}

	self.Set("user", self.user).
		Set("countUnread", self.countUnread).
		Set("countErrorFeeds", self.countErrorFeeds).
		Set("hasSaveEntry", self.hasSaveEntry)
	return nil
}

func (self *View) Render(templateName string) []byte {
	self.preProcessingElapsed = time.Since(self.startTime)
	startTime := time.Now()
	b := self.View.Render(templateName)
	self.renderingElapsed = time.Since(startTime)
	return b
}

func (self *View) CountUnread() int          { return self.countUnread }
func (self *View) CountErrorFeed() int       { return self.countErrorFeeds }
func (self *View) HasSaveEntry() bool        { return self.hasSaveEntry }
func (self *View) User() *model.User         { return self.user }
func (self *View) Session() *session.Session { return self.session }

func (self *View) CountUnreadElapsed() time.Duration {
	return self.countUnreadElapsed
}

func (self *View) PreProcessingElapsed() time.Duration {
	return self.preProcessingElapsed
}

func (self *View) RenderingElapsed() time.Duration {
	return self.renderingElapsed
}
