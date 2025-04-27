package storage

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

var (
	poolAcquireCountGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "miniflux",
		Name:      "pgx_acquire_count",
		Help:      "The cumulative count of successful acquires from the pool",
	})

	poolAcquireDurationGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "miniflux",
		Name:      "pgx_acquire_duration",
		Help:      "The total duration of all successful acquires from the pool",
	})

	poolAcquiredConnsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "miniflux",
		Name:      "pgx_acquired_conns",
		Help:      "The number of currently acquired connections in the pool",
	})

	poolCanceledAcquireCountGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "miniflux",
		Name:      "pgx_canceled_acquire_count",
		Help:      "The cumulative count of acquires from the pool that were canceled by a context",
	})

	poolConstructingConnsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "miniflux",
		Name:      "pgx_constructing_conns",
		Help:      "The number of conns with construction in progress in the pool",
	})

	poolEmptyAcquireCountGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "miniflux",
		Name:      "pgx_empty_acquire_count",
		Help:      "The cumulative count of successful acquires from the pool that waited for a resource to be released or constructed because the pool was empty",
	})

	poolEmptyAcquireWaitTimeGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "miniflux",
		Name:      "pgx_empty_acquire_wait_time",
		Help:      "The cumulative time waited for successful acquires from the pool for a resource to be released or constructed because the pool was empty",
	})

	poolIdleConnsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "miniflux",
		Name:      "pgx_idle_conns",
		Help:      "The number of currently idle conns in the pool",
	})

	poolMaxConnsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "miniflux",
		Name:      "pgx_max_conns",
		Help:      "The maximum size of the pool",
	})

	poolMaxIdleDestroyCountGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "miniflux",
		Name:      "pgx_max_idle_destroy_count",
		Help:      "The cumulative count of connections destroyed because they exceeded MaxConnIdleTime",
	})

	poolMaxLifetimeDestroyCountGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "miniflux",
		Name:      "pgx_max_lifetime_destroy_count",
		Help:      "The cumulative count of connections destroyed because they exceeded MaxConnLifetime",
	})

	poolNewConnsCountGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "miniflux",
		Name:      "pgx_new_conns_count",
		Help:      "The cumulative count of new connections opened",
	})

	poolTotalConnsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "miniflux",
		Name:      "pgx_total_conns",
		Help:      "The total number of resources currently in the pool",
	})

	usersGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "miniflux",
		Name:      "users",
		Help:      "Number of users",
	})

	brokenFeedsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "miniflux",
		Name:      "broken_feeds",
		Help:      "Number of broken feeds",
	})

	feedsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "miniflux",
		Name:      "feeds",
		Help:      "Number of feeds by status",
	}, []string{"status"})

	entriesGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "miniflux",
		Name:      "entries",
		Help:      "Number of entries by status",
	}, []string{"status"})
)

func (s *Storage) RegisterMetricts() {
	prometheus.MustRegister(
		poolAcquireCountGauge,
		poolAcquireDurationGauge,
		poolAcquiredConnsGauge,
		poolCanceledAcquireCountGauge,
		poolConstructingConnsGauge,
		poolEmptyAcquireCountGauge,
		poolEmptyAcquireWaitTimeGauge,
		poolIdleConnsGauge,
		poolMaxConnsGauge,
		poolMaxIdleDestroyCountGauge,
		poolMaxLifetimeDestroyCountGauge,
		poolNewConnsCountGauge,
		poolTotalConnsGauge,
		usersGauge,
		brokenFeedsGauge,
		feedsGauge,
		entriesGauge)
}

func (s *Storage) Metrics(ctx context.Context, fromDB bool) error {
	if fromDB {
		if err := s.metricsFromDB(ctx); err != nil {
			return err
		}
	}

	stat := s.db.Stat()
	poolAcquireCountGauge.Set(float64(stat.AcquireCount()))
	poolAcquireDurationGauge.Set(float64(stat.AcquireDuration()))
	poolAcquiredConnsGauge.Set(float64(stat.AcquiredConns()))
	poolCanceledAcquireCountGauge.Set(float64(stat.CanceledAcquireCount()))
	poolConstructingConnsGauge.Set(float64(stat.ConstructingConns()))
	poolEmptyAcquireCountGauge.Set(float64(stat.EmptyAcquireCount()))
	poolEmptyAcquireWaitTimeGauge.Set(float64(stat.EmptyAcquireWaitTime()))
	poolIdleConnsGauge.Set(float64(stat.IdleConns()))
	poolMaxConnsGauge.Set(float64(stat.MaxConns()))
	poolMaxIdleDestroyCountGauge.Set(float64(stat.MaxIdleDestroyCount()))
	poolMaxLifetimeDestroyCountGauge.Set(float64(stat.MaxLifetimeDestroyCount()))
	poolNewConnsCountGauge.Set(float64(stat.NewConnsCount()))
	poolTotalConnsGauge.Set(float64(stat.TotalConns()))
	return nil
}

func (s *Storage) metricsFromDB(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return s.updateUsersGauge(ctx) })
	g.Go(func() error { return s.updateBrokenFeedsGauge(ctx) })
	g.Go(func() error { return s.updateFeedsGauge(ctx) })

	if err := s.updateEntriesGauge(ctx); err != nil {
		_ = g.Wait()
		return err
	}
	return g.Wait() //nolint:wrapcheck // already wrapped
}

func (s *Storage) updateUsersGauge(ctx context.Context) error {
	usersCount, err := s.CountUsers(ctx)
	if err != nil {
		return err
	}
	usersGauge.Set(float64(usersCount))
	return nil
}

func (s *Storage) updateBrokenFeedsGauge(ctx context.Context) error {
	feedsCount, err := s.CountAllFeedsWithErrors(ctx)
	if err != nil {
		return err
	}
	brokenFeedsGauge.Set(float64(feedsCount))
	return nil
}

func (s *Storage) updateFeedsGauge(ctx context.Context) error {
	feedsCount, err := s.CountAllFeeds(ctx)
	if err != nil {
		return err
	}
	for status, count := range feedsCount {
		feedsGauge.WithLabelValues(status).Set(float64(count))
	}
	return nil
}

func (s *Storage) updateEntriesGauge(ctx context.Context) error {
	entriesCount, err := s.CountAllEntries(ctx)
	if err != nil {
		return err
	}
	for status, count := range entriesCount {
		entriesGauge.WithLabelValues(status).Set(float64(count))
	}
	return nil
}
