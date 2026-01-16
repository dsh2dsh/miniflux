// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package timezone // import "miniflux.app/v2/internal/timezone"

import (
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Convert returns the provided time expressed in the given timezone.
func Convert(tz string, times ...*time.Time) {
	for _, t := range times {
		*t = convert(tz, t)
	}
}

func convert(tz string, t *time.Time) time.Time {
	name := t.Location().String()
	if name == tz {
		return *t
	}

	userTimezone := getLocation(tz)
	if name != "" {
		return t.In(userTimezone)
	}

	if t.Before(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)) {
		return time.Date(0, time.January, 1, 0, 0, 0, 0, userTimezone)
	}

	// In this case, the provided date is already converted to the user timezone
	// by Postgres, but the timezone information is not set in the time struct. We
	// cannot use time.In() because the date will be converted a second time.
	return time.Date(
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second(),
		t.Nanosecond(),
		userTimezone,
	)
}

// Now returns the current time in the given timezone.
func Now(tz string) time.Time { return time.Now().In(getLocation(tz)) }

func getLocation(tz string) *time.Location { return locations.Location(tz) }

var locations = newLocationCache()

func newLocationCache() *locationCache {
	return &locationCache{locations: make(map[string]*time.Location)}
}

type locationCache struct {
	mu        sync.RWMutex
	locations map[string]*time.Location
	sg        singleflight.Group
}

func (self *locationCache) Location(tz string) *time.Location {
	self.mu.RLock()
	loc, ok := self.locations[tz]
	self.mu.RUnlock()
	if ok {
		return loc
	}

	v, _, _ := self.sg.Do(tz, func() (any, error) {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			loc = time.Local
		}
		self.mu.Lock()
		self.locations[tz] = loc
		self.mu.Unlock()
		return loc, nil
	})
	return v.(*time.Location)
}
