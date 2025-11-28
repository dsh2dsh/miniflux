package worker

import (
	"context"

	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/storage"
)

type queueItem struct {
	*model.Job

	store *storage.Storage
	ctx   context.Context
	index int
	users *userCache
	end   func()

	err       error
	refreshed *model.FeedRefreshed
	traceStat *storage.TraceStat
}

func makeItems(ctx context.Context, store *storage.Storage, jobs []model.Job,
	users *userCache, end func(),
) []queueItem {
	items := make([]queueItem, 0, len(jobs))
	for job := range distributeJobs(jobs) {
		items = append(items, queueItem{
			Job: job,

			store: store,
			ctx:   ctx,
			index: len(items),
			users: users,
			end:   end,
		})
	}
	return items
}

func (self *queueItem) Id() int { return self.index + 1 }
