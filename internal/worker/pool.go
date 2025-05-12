// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package worker // import "miniflux.app/v2/internal/worker"

import (
	"context"
	"log/slog"
	"math/rand/v2"
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
		queue: make(chan model.Job),
		store: store,
	}
	self.g.SetLimit(n)
	return self
}

// Pool handles a pool of workers.
type Pool struct {
	ctx   context.Context
	queue chan model.Job
	g     errgroup.Group

	store *storage.Storage
}

// Push send a list of jobs to the queue.
func (self *Pool) Push(ctx context.Context, jobs []model.Job) {
	rand.Shuffle(len(jobs), func(i, j int) {
		jobs[i], jobs[j] = jobs[j], jobs[i]
	})

	for i, job := range jobs {
		job.SetIndex(i)
		select {
		case <-self.ctx.Done():
			return
		case <-ctx.Done():
			return
		case self.queue <- job:
		}
	}
	logging.FromContext(ctx).Info("worker: sent a batch of feeds to the queue",
		slog.Int("nb_jobs", len(jobs)))
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
				self.refreshFeed(job)
				return nil
			})
		}
	}
}

func (self *Pool) refreshFeed(job model.Job) {
	log := logging.FromContext(self.ctx).With(slog.Int("job", job.Index()))
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
