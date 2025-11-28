package worker

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/singleflight"

	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/storage"
)

type userCache struct {
	store *storage.Storage

	mu sync.RWMutex
	sg singleflight.Group

	users map[int64]*model.User
	hit   atomic.Uint64
	miss  uint64
}

func (self *userCache) Init(store *storage.Storage) *userCache {
	self.store = store
	self.users = map[int64]*model.User{}
	return self
}

func (self *userCache) UserByID(ctx context.Context, id int64,
) (*model.User, error) {
	if u := self.userFromMap(id); u != nil {
		return u, nil
	}

	userID := strconv.FormatInt(id, 10)
	v, err, shared := self.sg.Do(userID, func() (any, error) {
		if u := self.userFromMap(id); u != nil {
			return u, nil
		}

		u, err := self.store.UserByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return self.rememberUser(u), nil
	})
	if err != nil {
		return nil, fmt.Errorf("worker: fetch user id=%v to cache: %w", id, err)
	}

	if shared {
		self.hit.Add(1)
	}
	return v.(*model.User), nil
}

func (self *userCache) userFromMap(id int64) *model.User {
	self.mu.RLock()
	defer self.mu.RUnlock()
	if u, ok := self.users[id]; ok {
		self.hit.Add(1)
		return u
	}
	return nil
}

func (self *userCache) rememberUser(u *model.User) *model.User {
	self.mu.Lock()
	self.miss++
	self.users[u.ID] = u
	self.mu.Unlock()
	return u
}

func (self *userCache) Stats() (hit, miss uint64) {
	return self.hit.Load(), self.miss
}
