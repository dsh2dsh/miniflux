package handler

import "miniflux.app/v2/internal/storage"

type Option func(r *Refresh)

func WithForceUpdate(v bool) Option {
	return func(self *Refresh) { self.force = v }
}

func WithUserByID(fn storage.UserByIDFunc) Option {
	return func(self *Refresh) { self.userByIDFunc = fn }
}
