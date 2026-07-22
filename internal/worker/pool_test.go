package worker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/storage"
)

func Test_makeItems(t *testing.T) {
	tests := []struct {
		name string
		jobs []model.Job
		want []queueItem
	}{
		{
			name: "one job",
			jobs: []model.Job{{FeedURL: "https://a.com"}},
			want: []queueItem{{Job: &model.Job{FeedURL: "https://a.com"}}},
		},
		{
			name: "2a1b",
			jobs: []model.Job{
				{FeedURL: "https://a.com"},
				{FeedURL: "https://a.com"},
				{FeedURL: "https://b.com"},
			},
			want: []queueItem{
				{Job: &model.Job{FeedURL: "https://a.com"}},
				{Job: &model.Job{FeedURL: "https://b.com"}},
				{Job: &model.Job{FeedURL: "https://a.com"}},
			},
		},
		{
			name: "1a3b",
			jobs: []model.Job{
				{FeedURL: "https://a.com"},
				{FeedURL: "https://b.com"},
				{FeedURL: "https://b.com"},
				{FeedURL: "https://b.com"},
			},
			want: []queueItem{
				{Job: &model.Job{FeedURL: "https://b.com"}},
				{Job: &model.Job{FeedURL: "https://a.com"}},
				{Job: &model.Job{FeedURL: "https://b.com"}},
				{Job: &model.Job{FeedURL: "https://b.com"}},
			},
		},
		{
			name: "1c3b2a",
			jobs: []model.Job{
				{FeedURL: "https://c.com"},
				{FeedURL: "https://b.com"},
				{FeedURL: "https://b.com"},
				{FeedURL: "https://b.com"},
				{FeedURL: "https://a.com"},
				{FeedURL: "https://a.com"},
			},
			want: []queueItem{
				{Job: &model.Job{FeedURL: "https://b.com"}},
				{Job: &model.Job{FeedURL: "https://a.com"}},
				{Job: &model.Job{FeedURL: "https://c.com"}},
				{Job: &model.Job{FeedURL: "https://b.com"}},
				{Job: &model.Job{FeedURL: "https://a.com"}},
				{Job: &model.Job{FeedURL: "https://b.com"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeItems(t.Context(), nil, tt.jobs, nil, nil)
			assert.EqualExportedValues(t, tt.want, got)
		})
	}
}

func TestPushAfterShutdownDiscardsJobs(t *testing.T) {
	require.NoError(t, config.Load(""))

	ctx, cancel := context.WithCancel(t.Context())
	pool := NewPool(ctx, new(storage.Storage), nil)
	cancel()

	done := make(chan struct{})
	go func() {
		pool.Push(t.Context(), []model.Job{{FeedID: 1}, {FeedID: 2}})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Push blocked after Shutdown instead of discarding jobs")
	}
}

func TestShutdownUnblocksPendingPush(t *testing.T) {
	require.NoError(t, config.Load(""))

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	pool := NewPool(ctx, new(storage.Storage), nil)

	pushed := make(chan struct{})
	go func() {
		pool.Push(t.Context(), []model.Job{{FeedID: 1}})
		close(pushed)
	}()

	// Give Push time to block on the unbuffered queue before shutting down.
	time.Sleep(10 * time.Millisecond)
	done := make(chan struct{})
	go func() {
		cancel()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown deadlocked while a Push was pending")
	}

	select {
	case <-pushed:
	case <-time.After(5 * time.Second):
		t.Fatal("Push remained blocked after Shutdown")
	}
}
