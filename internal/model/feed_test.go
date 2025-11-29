// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import (
	"os"
	"strconv"
	"testing"
	"time"

	"miniflux.app/v2/internal/config"
)

const noRefreshDelay = 0

func TestFeedCategorySetter(t *testing.T) {
	feed := &Feed{}
	feed.WithCategoryID(int64(123))

	if feed.Category == nil {
		t.Fatal(`The category field should not be null`)
	}

	if feed.Category.ID != int64(123) {
		t.Error(`The category ID must be set`)
	}
}

func TestFeedErrorCounter(t *testing.T) {
	feed := &Feed{}
	feed.WithTranslatedErrorMessage("Some Error")

	if feed.ParsingErrorMsg != "Some Error" {
		t.Error(`The error message must be set`)
	}

	if feed.ParsingErrorCount != 1 {
		t.Error(`The error counter must be set to 1`)
	}

	feed.ResetErrorCounter()

	if feed.ParsingErrorMsg != "" {
		t.Error(`The error message must be removed`)
	}

	if feed.ParsingErrorCount != 0 {
		t.Error(`The error counter must be set to 0`)
	}
}

func TestFeedCheckedNow(t *testing.T) {
	feed := &Feed{}
	feed.FeedURL = "https://example.org/feed"
	feed.CheckedNow()

	if feed.SiteURL != feed.FeedURL {
		t.Error(`The site URL must not be empty`)
	}

	if feed.CheckedAt.IsZero() {
		t.Error(`The checked date must be set`)
	}
}

func checkTargetInterval(t *testing.T, feed *Feed, targetInterval int, timeBefore time.Time, message string) {
	if feed.NextCheckAt.Before(timeBefore.Add(time.Minute * time.Duration(targetInterval))) {
		t.Errorf(`The next_check_at should be after timeBefore + %s`, message)
	}
	if feed.NextCheckAt.After(time.Now().Add(time.Minute * time.Duration(targetInterval))) {
		t.Errorf(`The next_check_at should be before now + %s`, message)
	}
}

func TestFeedScheduleNextCheckRoundRobinDefault(t *testing.T) {
	os.Clearenv()

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	timeBefore := time.Now()
	feed := &Feed{}
	feed.ScheduleNextCheck(noRefreshDelay)

	if feed.NextCheckAt.IsZero() {
		t.Error(`The next_check_at must be set`)
	}

	targetInterval := config.SchedulerRoundRobinMinInterval()
	checkTargetInterval(t, feed, targetInterval, timeBefore, "TestFeedScheduleNextCheckRoundRobinDefault")
}

func TestFeedScheduleNextCheckRoundRobinWithRefreshDelayAboveMinInterval(t *testing.T) {
	os.Clearenv()

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	timeBefore := time.Now()
	feed := &Feed{}

	feed.ScheduleNextCheck(config.SchedulerRoundRobinMinInterval() + 30)

	if feed.NextCheckAt.IsZero() {
		t.Error(`The next_check_at must be set`)
	}

	expectedInterval := config.SchedulerRoundRobinMinInterval() + 30
	checkTargetInterval(t, feed, expectedInterval, timeBefore, "TestFeedScheduleNextCheckRoundRobinWithRefreshDelayAboveMinInterval")
}

func TestFeedScheduleNextCheckRoundRobinWithRefreshDelayBelowMinInterval(t *testing.T) {
	os.Clearenv()

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	timeBefore := time.Now()
	feed := &Feed{}

	feed.ScheduleNextCheck(config.SchedulerRoundRobinMinInterval() - 30)

	if feed.NextCheckAt.IsZero() {
		t.Error(`The next_check_at must be set`)
	}

	expectedInterval := config.SchedulerRoundRobinMinInterval()
	checkTargetInterval(t, feed, expectedInterval, timeBefore, "TestFeedScheduleNextCheckRoundRobinWithRefreshDelayBelowMinInterval")
}

func TestFeedScheduleNextCheckRoundRobinWithRefreshDelayAboveMaxInterval(t *testing.T) {
	os.Clearenv()

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	timeBefore := time.Now()
	feed := &Feed{}

	feed.ScheduleNextCheck(config.SchedulerRoundRobinMaxInterval() + 30)

	if feed.NextCheckAt.IsZero() {
		t.Error(`The next_check_at must be set`)
	}

	expectedInterval := config.SchedulerRoundRobinMaxInterval()
	checkTargetInterval(t, feed, expectedInterval, timeBefore, "TestFeedScheduleNextCheckRoundRobinWithRefreshDelayAboveMaxInterval")
}

func TestFeedScheduleNextCheckRoundRobinMinInterval(t *testing.T) {
	minInterval := 1
	os.Clearenv()
	t.Setenv("POLLING_SCHEDULER", "round_robin")
	t.Setenv("SCHEDULER_ROUND_ROBIN_MIN_INTERVAL", strconv.Itoa(minInterval))

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	timeBefore := time.Now()
	feed := &Feed{}
	feed.ScheduleNextCheck(noRefreshDelay)

	if feed.NextCheckAt.IsZero() {
		t.Error(`The next_check_at must be set`)
	}

	expectedInterval := minInterval
	checkTargetInterval(t, feed, expectedInterval, timeBefore, "TestFeedScheduleNextCheckRoundRobinMinInterval")
}
