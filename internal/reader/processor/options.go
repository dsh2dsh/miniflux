package processor

import "miniflux.app/v2/internal/storage"

type Option func(self *FeedProcessor)

func WithSkipAgedFilter() Option {
	return func(self *FeedProcessor) { self.WithSkipAgedFilter() }
}

func WithUserByID(fn storage.UserByIDFunc) Option {
	return func(self *FeedProcessor) { self.userByIDFunc = fn }
}
