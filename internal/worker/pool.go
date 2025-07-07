// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package worker // import "miniflux.app/v2/internal/worker"

import (
	"cmp"
	"context"
	"errors"
	"iter"
	"log/slog"
	"maps"
	"net/url"
	"slices"
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
		ctx:      ctx,
		queue:    make(chan *queueItem),
		store:    store,
		wakeupCh: make(chan struct{}, 1),
	}
	self.g.SetLimit(n)
	return self
}

// Pool handles a pool of workers.
type Pool struct {
	ctx   context.Context
	queue chan *queueItem
	g     errgroup.Group

	store *storage.Storage

	wakeupCh chan struct{}

	mu  sync.Mutex
	err error
}

type queueItem struct {
	*model.Job

	index int
	end   func()

	err       error
	traceStat *storage.TraceStat
}

func NewItem(job *model.Job, index int, end func()) queueItem {
	return queueItem{
		Job:   job,
		index: index,
		end:   end,
	}
}

func (self *Pool) Wakeup() {
	select {
	case self.wakeupCh <- struct{}{}:
	default:
	}
}

func (self *Pool) WakeupSignal() <-chan struct{} { return self.wakeupCh }

func (self *Pool) Err() error {
	self.mu.Lock()
	defer self.mu.Unlock()
	return self.err
}

func (self *Pool) setErr(err error) {
	self.mu.Lock()
	self.err = err
	self.mu.Unlock()
}

// Push send a list of jobs to the queue.
func (self *Pool) Push(ctx context.Context, jobs []model.Job) {
	log := logging.FromContext(ctx).With(slog.Int("jobs", len(jobs)))
	log.Info("worker: created a batch of feeds")

	var wg sync.WaitGroup
	wg.Add(len(jobs))
	items := makeItems(jobs, wg.Done)
	startTime := time.Now()

jobsLoop:
	for i := range items {
		select {
		case <-self.ctx.Done():
			break jobsLoop
		case <-ctx.Done():
			break jobsLoop
		case self.queue <- &items[i]:
		}
	}

	switch {
	case self.ctx.Err() != nil:
		log.Info("worker: batch cancelled by daemon",
			slog.Any("reason", context.Cause(self.ctx)))
		return
	case ctx.Err() != nil:
		log.Info("worker:  batch cancelled by request",
			slog.Any("reason", context.Cause(self.ctx)))
		return
	}

	log.Info("worker: waiting for batch completion")
	wg.Wait()

	traceStat := sumTraceStats(items)
	log = log.With(
		slog.Group("storage",
			slog.Int64("queries", traceStat.Queries),
			slog.Duration("elapsed", traceStat.Elapsed)),
		slog.Duration("elapsed", time.Since(startTime)))

	for i := range items {
		if err := items[i].err; err != nil {
			self.setErr(err)
			log.Info("worker: refreshed a batch of feeds with error",
				slog.Any("error", err))
			return
		}
	}

	self.setErr(nil)
	log.Info("worker: refreshed a batch of feeds")
}

func makeItems(jobs []model.Job, end func()) []queueItem {
	items := make([]queueItem, 0, len(jobs))
	for job := range distributeJobs(jobs) {
		items = append(items, NewItem(job, len(items), end))
	}
	return items
}

func distributeJobs(jobs []model.Job) iter.Seq[*model.Job] {
	perHost := map[string][]*model.Job{}
	for i := range jobs {
		j := &jobs[i]
		if u, err := url.Parse(j.FeedURL); err != nil {
			perHost[j.FeedURL] = append(perHost[j.FeedURL], j)
		} else {
			perHost[u.Host] = append(perHost[u.Host], j)
		}
	}

	hosts := slices.SortedFunc(maps.Keys(perHost), func(a, b string) int {
		return cmp.Compare(len(perHost[b]), len(perHost[a]))
	})

	return func(yield func(job *model.Job) bool) {
		for len(hosts) > 0 {
			var deleted bool
			for i, host := range hosts {
				hostJobs := perHost[host]
				j := hostJobs[len(hostJobs)-1]
				if !yield(j) {
					return
				}
				if len(hostJobs) > 1 {
					perHost[host] = hostJobs[:len(hostJobs)-1]
				} else {
					hosts[i] = ""
					deleted = true
				}
			}
			if deleted {
				hosts = slices.DeleteFunc(hosts,
					func(host string) bool { return host == "" })
			}
		}
	}
}

func (self *Pool) Run() error {
	log := logging.FromContext(self.ctx)
	log.Info("worker pool started")

forLoop:
	for {
		select {
		case <-self.ctx.Done():
			break forLoop
		case job := <-self.queue:
			self.g.Go(func() error {
				err := self.refreshFeed(job)
				job.end()
				log = log.With(slog.Int("job", job.index))
				if err != nil {
					log.Info("worker: job completed with error", slog.Any("error", err))
				} else {
					log.Info("worker: job completed")
				}
				return nil
			})
		}
	}

	log.Info("worker: waiting all jobs completed",
		slog.Any("reason", context.Cause(self.ctx)))
	self.Wakeup()
	if err := self.g.Wait(); err != nil {
		log.Error("worker pool stopped with error", slog.Any("error", err))
	} else {
		log.Info("worker pool stopped")
	}
	return nil
}

func (self *Pool) refreshFeed(job *queueItem) error {
	log := logging.FromContext(self.ctx).With(slog.Int("job", job.index))
	ctx := storage.WithTraceStat(logging.WithLogger(self.ctx, log))

	log = log.With(
		slog.Int64("user_id", job.UserID),
		slog.Int64("feed_id", job.FeedID))
	log.Debug("worker: job received")

	startTime := time.Now()
	err := handler.RefreshFeed(ctx, self.store, job.UserID, job.FeedID, false)
	if err != nil && !errors.Is(err, handler.ErrBadFeed) {
		job.err = err
		log.Error("worker: error refreshing feed", slog.Any("error", err))
	}

	if config.Opts.HasMetricsCollector() {
		status := "success"
		if err != nil {
			status = "error"
		}
		metric.BackgroundFeedRefreshDuration.
			WithLabelValues(status).
			Observe(time.Since(startTime).Seconds())
	}

	if t := storage.TraceStatFrom(ctx); t != nil {
		job.traceStat = t
	}
	return err
}

func sumTraceStats(items []queueItem) storage.TraceStat {
	var traceStat storage.TraceStat
	for i := range items {
		item := &items[i]
		if item.traceStat != nil {
			traceStat.Add(item.traceStat)
		}
	}
	return traceStat
}
