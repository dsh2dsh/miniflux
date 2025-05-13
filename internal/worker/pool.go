// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package worker // import "miniflux.app/v2/internal/worker"

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/metric"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/handler"
	"miniflux.app/v2/internal/storage"
)

// NewPool creates a pool of background workers.
func NewPool(ctx context.Context, store *storage.Storage, n int) *Pool {
	self := &Pool{
		ctx:   ctx,
		queue: make(chan queueItem),
		store: store,
	}
	self.g.SetLimit(n)
	return self
}

// Pool handles a pool of workers.
type Pool struct {
	ctx   context.Context
	queue chan queueItem
	g     errgroup.Group

	store *storage.Storage
}

type queueItem struct {
	*model.Job

	index int
	begin func()
	end   func()
}

func NewItem(job *model.Job, index int, begin, end func()) queueItem {
	return queueItem{
		Job:   job,
		index: index,
		begin: begin,
		end:   end,
	}
}

// Push send a list of jobs to the queue.
func (self *Pool) Push(ctx context.Context, jobs []model.Job) {
	log := logging.FromContext(ctx).With(slog.Int("jobs", len(jobs)))
	log.Info("worker: created a batch of feeds")

	rand.Shuffle(len(jobs), func(i, j int) {
		jobs[i], jobs[j] = jobs[j], jobs[i]
	})

	var wg sync.WaitGroup
	beginJob := func() { wg.Add(1) }
	startTime := time.Now()

jobsLoop:
	for i := range jobs {
		select {
		case <-self.ctx.Done():
			break jobsLoop
		case <-ctx.Done():
			break jobsLoop
		case self.queue <- NewItem(&jobs[i], i, beginJob, wg.Done):
		}
	}

	wg.Wait()
	log.Info("worker: refreshed a batch of feeds",
		slog.Duration("elapsed", time.Since(startTime)))
}

func (self *Pool) Run() error {
	log := logging.FromContext(self.ctx)
	log.Info("worker pool started")
	for {
		select {
		case <-self.ctx.Done():
			log.Info("worker pool stopped")
			return nil
		case job := <-self.queue:
			self.g.Go(func() error {
				job.begin()
				self.refreshFeed(job)
				job.end()
				return nil
			})
		}
	}
}

func (self *Pool) refreshFeed(job queueItem) {
	log := logging.FromContext(self.ctx).With(slog.Int("job", job.index))
	log.Debug("worker: job received",
		slog.Int64("user_id", job.UserID), slog.Int64("feed_id", job.FeedID))

	startTime := time.Now()
	err := handler.RefreshFeed(logging.WithLogger(self.ctx, log),
		self.store, job.UserID, job.FeedID, false)

	if config.Opts.HasMetricsCollector() {
		status := "success"
		if err != nil {
			status = "error"
		}
		metric.BackgroundFeedRefreshDuration.
			WithLabelValues(status).
			Observe(time.Since(startTime).Seconds())
	}
}
