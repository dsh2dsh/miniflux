package ui

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/ui/session"
	"miniflux.app/v2/internal/ui/view"
)

func (h *handler) View(r *http.Request) *View {
	s := session.New(h.store, r)
	self := &View{
		View:  view.New(h.tpl, r, s),
		store: h.store,
		user:  request.User(r),
	}
	self.g, self.ctx = errgroup.WithContext(r.Context())
	return self.init()
}

type View struct {
	*view.View

	store *storage.Storage
	user  *model.User

	g   *errgroup.Group
	ctx context.Context

	countUnread     int
	countErrorFeeds int
	hasSaveEntry    bool
}

func (self *View) init() *View {
	self.Go(func(ctx context.Context) (err error) {
		self.countUnread, err = self.store.NewEntryQueryBuilder(self.UserID()).
			WithStatus(model.EntryStatusUnread).
			WithGloballyVisible().
			CountEntries(ctx)
		return
	})

	self.Go(func(ctx context.Context) error {
		self.countErrorFeeds = self.store.CountUserFeedsWithErrors(ctx,
			self.UserID())
		return nil
	})
	return self
}

func (self *View) Go(fn func(ctx context.Context) error) {
	self.g.Go(func() error { return fn(self.ctx) })
}

func (self *View) WithSaveEntry() *View {
	self.hasSaveEntry = self.user.HasSaveEntry()
	return self
}

func (self *View) wait() error {
	if err := self.g.Wait(); err != nil {
		return fmt.Errorf("group error: %w", err)
	}
	return nil
}

func (self *View) Wait() error {
	if err := self.wait(); err != nil {
		return err
	}

	self.Set("user", self.user).
		Set("countUnread", self.countUnread).
		Set("countErrorFeeds", self.countErrorFeeds).
		Set("hasSaveEntry", self.hasSaveEntry)
	return nil
}

func (self *View) CountErrorFeed() int { return self.countErrorFeeds }
func (self *View) CountUnread() int    { return self.countUnread }
func (self *View) HasSaveEntry() bool  { return self.hasSaveEntry }
func (self *View) User() *model.User   { return self.user }
func (self *View) UserID() int64       { return self.user.ID }

func (self *View) WaitEntriesCount(query *storage.EntryQueryBuilder,
) (model.Entries, int, error) {
	var entries model.Entries
	self.Go(func(ctx context.Context) (err error) {
		entries, err = query.GetEntries(ctx)
		return
	})

	var count int
	self.Go(func(ctx context.Context) (err error) {
		count, err = query.CountEntries(ctx)
		return
	})
	return entries, count, self.Wait()
}
