package worker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"miniflux.app/v2/internal/model"
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
			got := makeItems(context.Background(), tt.jobs, nil)
			assert.EqualExportedValues(t, tt.want, got)
		})
	}
}
