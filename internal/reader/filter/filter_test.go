// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package filter // import "miniflux.app/v2/internal/reader/filter"

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/model"
)

func TestBlockingEntries(t *testing.T) {
	tests := []struct {
		name     string
		feed     model.Feed
		user     model.User
		expected int
		err      bool
	}{
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					BlockFilterEntryRules: "Any=(?i)example",
				},
				Entries: model.Entries{{URL: "https://example.com"}},
			},
		},
		{ // invalid regex
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					BlockFilterEntryRules: "Any=[a-z",
				},
				Entries: model.Entries{{URL: "https://example.com"}},
			},
			err: true,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					BlockFilterEntryRules: "Any=(?i)example",
				},
				Entries: model.Entries{{URL: "https://different.com"}},
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					BlockFilterEntryRules: "Any=(?i)example",
				},
				Entries: model.Entries{{Title: "Some Example"}},
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					BlockFilterEntryRules: "Any=(?i)example",
				},
				Entries: model.Entries{{Title: "Something different"}},
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					BlockFilterEntryRules: "Any=(?i)example",
				},
				Entries: model.Entries{
					{
						Title: "Something different",
						Tags:  []string{"example", "something else"},
					},
				},
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					BlockFilterEntryRules: "Any=(?i)example",
				},
				Entries: model.Entries{
					{
						Title: "Example",
						Tags:  []string{"example", "something else"},
					},
				},
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					BlockFilterEntryRules: "Any=(?i)example",
				},
				Entries: model.Entries{
					{
						Title: "Example",
						Tags:  []string{"something different", "something else"},
					},
				},
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					BlockFilterEntryRules: "Any=(?i)example",
				},
				Entries: model.Entries{
					{
						Title: "Something different",
						Tags:  []string{"something different", "something else"},
					},
				},
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					BlockFilterEntryRules: "Any=(?i)example",
				},
				Entries: model.Entries{
					{Title: "Something different", Author: "Example"},
				},
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					BlockFilterEntryRules: "Any=(?i)example",
				},
				Entries: model.Entries{
					{Title: "Something different", Author: "Something different"},
				},
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries:  model.Entries{{Title: "No rule defined"}},
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{URL: "https://example.com", Title: "Some Example"},
				},
			},
			user: model.User{
				BlockFilterEntryRules: "EntryURL=(?i)example\nEntryTitle=(?i)Test",
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{URL: "https://different.com", Title: "Some Test"},
				},
			},
			user: model.User{
				BlockFilterEntryRules: "EntryURL=(?i)example\nEntryTitle=(?i)Test",
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{URL: "https://different.com", Title: "Some Example"},
				},
			},
			user: model.User{
				BlockFilterEntryRules: "EntryURL=(?i)example\nEntryTitle=(?i)Test",
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{CommentsURL: "https://example.com", Content: "Some Example"},
				},
			},
			user: model.User{
				BlockFilterEntryRules: "EntryCommentsURL=(?i)example\nEntryContent=(?i)Test",
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{CommentsURL: "https://different.com", Content: "Some Test"},
				},
			},
			user: model.User{
				BlockFilterEntryRules: "EntryCommentsURL=(?i)example\nEntryContent=(?i)Test",
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{CommentsURL: "https://different.com", Content: "Some Example"},
				},
			},
			user: model.User{
				BlockFilterEntryRules: "EntryCommentsURL=(?i)example\nEntryContent=(?i)Test",
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Author: "Example", Tags: []string{"example", "something else"}},
				},
			},
			user: model.User{
				BlockFilterEntryRules: "EntryAuthor=(?i)example\nEntryTag=(?i)Test",
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Author: "Different", Tags: []string{"example", "something else"}},
				},
			},
			user: model.User{
				BlockFilterEntryRules: "EntryAuthor=(?i)example\nEntryTag=(?i)example",
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Author: "Different", Tags: []string{"example", "something else"}},
				},
			},
			user: model.User{
				BlockFilterEntryRules: "EntryAuthor=(?i)example\nEntryTag=(?i)Test",
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Author: "Different", Tags: []string{"example", "test"}},
				},
			},
			user: model.User{
				BlockFilterEntryRules: "EntryAuthor\nEntryTag=(?i)Test",
			},
			err: true,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Date: time.Date(2024, 3, 14, 0, 0, 0, 0, time.UTC)},
				},
			},
			user: model.User{
				BlockFilterEntryRules: "EntryDate=before:2024-03-15",
			},
		},
		// Test max-age filter
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
			user: model.User{BlockFilterEntryRules: "EntryDate=max-age:30d"},
			// Entry from Jan 1, 2024 is definitely older than 30 days
		},
		// Invalid duration format
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
			user:     model.User{BlockFilterEntryRules: "EntryDate=max-age:invalid"},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Date: time.Date(2024, 3, 14, 0, 0, 0, 0, time.UTC)},
				},
			},
			user: model.User{BlockFilterEntryRules: "UnknownRuleType=test"},
			err:  true,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					BlockFilterEntryRules: "EntryURL=(?i)example\nEntryTitle=(?i)Test",
				},
				Entries: model.Entries{
					{URL: "https://example.com", Title: "Some Example"},
				},
			},
		},
		// Test cases for merged user and feed BlockFilterEntryRules
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{BlockFilterEntryRules: "EntryURL=(?i)website"},
				Entries: model.Entries{
					{URL: "https://example.com", Title: "Some Title"},
				},
			},
			user: model.User{
				BlockFilterEntryRules: "   EntryTitle=(?i)title   ",
			},
			// User rule matches
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{BlockFilterEntryRules: "EntryURL=(?i)example"},
				Entries: model.Entries{
					{URL: "https://example.com", Title: "Some Other"},
				},
			},
			user: model.User{BlockFilterEntryRules: "EntryTitle=(?i)title"},
			// Feed rule matches
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{BlockFilterEntryRules: "EntryURL=(?i)example"},
				Entries: model.Entries{
					{URL: "https://different.com", Title: "Some Other"},
				},
			},
			user:     model.User{BlockFilterEntryRules: "EntryTitle=(?i)title"},
			expected: 1,
			// Neither rule matches
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{BlockFilterEntryRules: "EntryURL=(?i)example"},
				Entries: model.Entries{
					{URL: "https://example.com", Title: "Some Title"},
				},
			},
			user: model.User{BlockFilterEntryRules: "EntryTitle=(?i)title"},
			// Both rules would match
		},
		// Test multiple rules with \r\n separators
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					BlockFilterEntryRules: "EntryURL=(?i)example\r\nEntryTitle=(?i)Test",
				},
				Entries: model.Entries{
					{URL: "https://example.com", Title: "Some Example"},
				},
			},
			user: model.User{BlockFilterEntryRules: "EntryTitle=(?i)title"},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					BlockFilterEntryRules: "EntryURL=(?i)example\r\nEntryTitle=(?i)Test",
				},
				Entries: model.Entries{
					{URL: "https://example.com", Title: "Some Test"},
				},
			},
			user: model.User{BlockFilterEntryRules: "EntryTitle=(?i)title"},
		},
		{
			name: "category rule matched",
			feed: model.Feed{
				Category: &model.Category{
					Extra: model.CategoryExtra{
						BlockFilter: "   EntryTitle=(?i)title   ",
					},
				},
				Entries: model.Entries{
					{URL: "https://example.com", Title: "Some Title"},
				},
			},
		},
	}

	require.NoError(t, config.Load(""))
	for i, tt := range tests {
		name := tt.name
		if name == "" {
			name = "test " + strconv.Itoa(i)
		}
		t.Run(name, func(t *testing.T) {
			user, feed := tt.user, tt.feed
			if tt.err {
				require.Error(t, DeleteEntries(t.Context(), &user, &feed))
				return
			}
			require.NoError(t, DeleteEntries(t.Context(), &user, &feed))
			assert.Len(t, feed.Entries, tt.expected)
		})
	}
}

func TestAllowEntries(t *testing.T) {
	tests := []struct {
		feed     model.Feed
		user     model.User
		expected int
		err      bool
	}{
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{KeepFilterEntryRules: "Any=(?i)example"},
				Entries:  model.Entries{{Title: "https://example.com"}},
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{KeepFilterEntryRules: "Any=[a-z"},
				Entries:  model.Entries{{Title: "https://example.com"}},
			},
			err: true,
			// invalid regex
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{KeepFilterEntryRules: "Any=(?i)example"},
				Entries:  model.Entries{{Title: "https://different.com"}},
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{KeepFilterEntryRules: "Any=(?i)example"},
				Entries:  model.Entries{{Title: "Some Example"}},
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{KeepFilterEntryRules: "Any=(?i)example"},
				Entries:  model.Entries{{Title: "Something different"}},
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries:  model.Entries{{Title: "No rule defined"}},
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{KeepFilterEntryRules: "Any=(?i)example"},
				Entries: model.Entries{
					{
						Title: "Something different",
						Tags:  []string{"example", "something else"},
					},
				},
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{KeepFilterEntryRules: "Any=(?i)example"},
				Entries: model.Entries{
					{
						Title: "Example",
						Tags:  []string{"example", "something else"},
					},
				},
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{KeepFilterEntryRules: "Any=(?i)example"},
				Entries: model.Entries{
					{
						Title: "Example",
						Tags:  []string{"something different", "something else"},
					},
				},
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{KeepFilterEntryRules: "Any=(?i)example"},
				Entries: model.Entries{
					{
						Title: "Something more",
						Tags:  []string{"something different", "something else"},
					},
				},
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{KeepFilterEntryRules: "Any=(?i)example"},
				Entries: model.Entries{
					{Title: "Something different", Author: "Example"},
				},
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{KeepFilterEntryRules: "Any=(?i)example"},
				Entries: model.Entries{
					{Title: "Something different", Author: "Something different"},
				},
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{URL: "https://example.com", Title: "Some Example"},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryURL=(?i)example\nEntryTitle=(?i)Test",
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{URL: "https://different.com", Title: "Some Test"},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryURL=(?i)example\nEntryTitle=(?i)Test",
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{URL: "https://different.com", Title: "Some Example"},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryURL=(?i)example\nEntryTitle=(?i)Test",
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{CommentsURL: "https://example.com", Content: "Some Example"},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryCommentsURL=(?i)example\nEntryContent=(?i)Test",
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{CommentsURL: "https://different.com", Content: "Some Test"},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryCommentsURL=(?i)example\nEntryContent=(?i)Test",
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{CommentsURL: "https://different.com", Content: "Some Example"},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryCommentsURL=(?i)example\nEntryContent=(?i)Test",
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{
						Author: "Example",
						Tags:   []string{"example", "something else"},
					},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryAuthor=(?i)example\nEntryTag=(?i)Test",
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{
						Author: "Different",
						Tags:   []string{"example", "something else"},
					},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryAuthor=(?i)example\nEntryTag=(?i)example",
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{
						Author: "Different",
						Tags:   []string{"example", "something else"},
					},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryAuthor=(?i)example\nEntryTag=(?i)Test",
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{
						Author: "Different",
						Tags:   []string{"example", "some test"},
					},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryAuthor\nEntryTag=(?i)Test",
			},
			err: true,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries:  model.Entries{{Date: time.Now().Add(24 * time.Hour)}},
			},
			user:     model.User{KeepFilterEntryRules: "EntryDate=future"},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries:  model.Entries{{Date: time.Now().Add(-24 * time.Hour)}},
			},
			user: model.User{KeepFilterEntryRules: "EntryDate=future"},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Date: time.Date(2024, 3, 14, 0, 0, 0, 0, time.UTC)},
				},
			},
			user:     model.User{KeepFilterEntryRules: "EntryDate=before:2024-03-15"},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Date: time.Date(2024, 3, 14, 0, 0, 0, 0, time.UTC)},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryDate=before:invalid-date",
			},
			// invalid date format
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Date: time.Date(2024, 3, 16, 0, 0, 0, 0, time.UTC)},
				},
			},
			user:     model.User{KeepFilterEntryRules: "EntryDate=after:2024-03-15"},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Date: time.Date(2024, 3, 16, 0, 0, 0, 0, time.UTC)},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryDate=after:invalid-date",
			},
			// invalid date format
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Date: time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryDate=between:2024-03-01,2024-03-15",
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Date: time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryDate=between:2024-03-01,2024-03-15",
			},
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Date: time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryDate=between:invalid-date,2024-03-15",
			},
			// invalid date format
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Date: time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryDate=between:2024-03-15,invalid-date",
			},
			// invalid date format
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Date: time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryDate=between:2024-03-15",
			},
			// missing second date in range
		},
		// Test max-age filter
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
			user:     model.User{KeepFilterEntryRules: "EntryDate=max-age:30d"},
			expected: 1,
			// Entry from Jan 1, 2024 is definitely older than 30 days
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
			user: model.User{KeepFilterEntryRules: "EntryDate=max-age:invalid"},
			// Invalid duration format
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Date: time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)},
				},
			},
			user: model.User{KeepFilterEntryRules: "EntryDate=abcd"},
			// no colon in rule value
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Entries: model.Entries{
					{Date: time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)},
				},
			},
			user: model.User{
				KeepFilterEntryRules: "EntryDate=unknown:2024-03-15",
			},
			// unknown rule type
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					KeepFilterEntryRules: "EntryURL=(?i)example\nEntryTitle=(?i)Test",
				},
				Entries: model.Entries{
					{URL: "https://example.com", Title: "Some Example"},
				},
			},
			expected: 1,
		},
		// Test cases for merged user and feed KeepFilterEntryRules
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{KeepFilterEntryRules: "EntryURL=(?i)website"},
				Entries: model.Entries{
					{URL: "https://example.com", Title: "Some Title"},
				},
			},
			user:     model.User{KeepFilterEntryRules: "EntryTitle=(?i)title"},
			expected: 1,
			// User rule matches
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{KeepFilterEntryRules: "EntryURL=(?i)example"},
				Entries: model.Entries{
					{URL: "https://example.com", Title: "Some Other"},
				},
			},
			user:     model.User{KeepFilterEntryRules: "EntryTitle=(?i)title"},
			expected: 1,
			// Feed rule matches
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{KeepFilterEntryRules: "EntryURL=(?i)example"},
				Entries: model.Entries{
					{URL: "https://different.com", Title: "Some Other"},
				},
			},
			user: model.User{KeepFilterEntryRules: "EntryTitle=(?i)title"},
			// Neither rule matches
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra:    model.FeedExtra{KeepFilterEntryRules: "EntryURL=(?i)example"},
				Entries: model.Entries{
					{URL: "https://example.com", Title: "Some Title"},
				},
			},
			user:     model.User{KeepFilterEntryRules: "EntryTitle=(?i)title"},
			expected: 1,
			// Both rules would match
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					KeepFilterEntryRules: "EntryURL=(?i)example\r\nEntryTitle=(?i)Test",
				},
				Entries: model.Entries{
					{URL: "https://example.com", Title: "Some Title"},
				},
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					KeepFilterEntryRules: "EntryURL=(?i)example\r\nEntryTitle=(?i)Test",
				},
				Entries: model.Entries{
					{URL: "https://different.com", Title: "Some Test"},
				},
			},
			expected: 1,
		},
		{
			feed: model.Feed{
				Category: &model.Category{},
				Extra: model.FeedExtra{
					KeepFilterEntryRules: "EntryURL=(?i)example\r\nEntryTitle=(?i)Test",
				},
				Entries: model.Entries{
					{URL: "https://different.com", Title: "Some Example"},
				},
			},
		},
	}

	require.NoError(t, config.Load(""))
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			user, feed := tt.user, tt.feed
			if tt.err {
				require.Error(t, DeleteEntries(t.Context(), &user, &feed))
				return
			}
			require.NoError(t, DeleteEntries(t.Context(), &user, &feed))
			assert.Len(t, feed.Entries, tt.expected)
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		err      bool
	}{
		{
			input:    "30d",
			expected: 30 * 24 * time.Hour,
		},
		{
			input:    "1h",
			expected: time.Hour,
		},
		{
			input:    "2m",
			expected: 2 * time.Minute,
		},
		{
			input: "invalid",
			err:   true,
		},
		// Invalid unit
		{
			input: "5x",
			err:   true,
		},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			result, err := parseDuration(tt.input)
			if tt.err {
				require.Error(t, err)
				return
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaxAgeFilter(t *testing.T) {
	oldFeed := model.Feed{
		Category: &model.Category{},
		Entries: model.Entries{
			{
				Title: "Old Entry",
				Date:  time.Now().Add(-48 * time.Hour), // 48 hours ago
			},
		},
	}

	newFeed := model.Feed{
		Category: &model.Category{},
		Entries: model.Entries{
			{
				Title: "New Entry",
				Date:  time.Now().Add(-30 * time.Minute), // 30 minutes ago
			},
		},
	}

	// Test blocking old entries
	user := model.User{BlockFilterEntryRules: "EntryDate=max-age:1d"}
	require.NoError(t, config.Load(""))

	// Old entry should be blocked (48 hours > 1 day is true)
	require.NoError(t, DeleteEntries(t.Context(), &user, &oldFeed))
	assert.Empty(t, oldFeed.Entries)

	// New entry should not be blocked
	require.NoError(t, DeleteEntries(t.Context(), &user, &newFeed))
	assert.Len(t, newFeed.Entries, 1)
}

func TestBlockedGlobally(t *testing.T) {
	require.NoError(t, config.Load(""))
	var user model.User
	feed := model.Feed{
		Category: &model.Category{},
		Entries: model.Entries{
			{Date: time.Date(2020, 5, 1, 0o5, 0o5, 0o5, 0o5, time.UTC)},
		},
	}
	require.NoError(t, DeleteEntries(t.Context(), &user, &feed))
	assert.Len(t, feed.Entries, 1,
		"Expected no entries to be blocked globally when max-age is not set")

	t.Setenv("FILTER_ENTRY_MAX_AGE_DAYS", "30")
	require.NoError(t, config.Load(""))

	feed.Entries = model.Entries{
		{Date: time.Date(2020, 5, 1, 0o5, 0o5, 0o5, 0o5, time.UTC)},
	}
	require.NoError(t, DeleteEntries(t.Context(), &user, &feed))
	assert.Empty(t, feed.Entries,
		"Expected entries to be blocked globally when max-age is set")

	feed.Entries = model.Entries{{Date: time.Now().Add(-2 * time.Hour)}}
	require.NoError(t, DeleteEntries(t.Context(), &user, &feed))
	assert.Len(t, feed.Entries, 1,
		"Expected entries not to be blocked globally when they are within the max-age limit")
}

func TestBlockedEntryWithGlobalMaxAge(t *testing.T) {
	t.Setenv("FILTER_ENTRY_MAX_AGE_DAYS", "30")
	require.NoError(t, config.Load(""))

	var user model.User
	feed := model.Feed{
		Category: &model.Category{},
		Entries: model.Entries{
			{
				Title: "Test Entry",
				Date:  time.Now().Add(-31 * 24 * time.Hour),
				// 31 days old
			},
		},
	}
	require.NoError(t, DeleteEntries(t.Context(), &user, &feed))
	assert.Empty(t, feed.Entries)
}

func TestBlockedEntryWithDefaultGlobalMaxAge(t *testing.T) {
	require.NoError(t, config.Load(""))
	var user model.User
	feed := model.Feed{
		Category: &model.Category{},
		Entries: model.Entries{
			// 31 days old
			{Title: "Test Entry", Date: time.Now().Add(-31 * 24 * time.Hour)},
		},
	}
	require.NoError(t, DeleteEntries(t.Context(), &user, &feed))
	assert.Len(t, feed.Entries, 1)
}
