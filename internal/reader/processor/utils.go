// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package processor // import "miniflux.app/v2/internal/reader/processor"

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// TODO: use something less horrible than a regex to parse ISO 8601 durations.

var iso8601Regex = regexp.MustCompile(`^P((?P<year>\d+)Y)?((?P<month>\d+)M)?((?P<week>\d+)W)?((?P<day>\d+)D)?(T((?P<hour>\d+)H)?((?P<minute>\d+)M)?((?P<second>\d+)S)?)?$`)

func parseISO8601(from string) (time.Duration, error) {
	var match []string
	var d time.Duration

	if iso8601Regex.MatchString(from) {
		match = iso8601Regex.FindStringSubmatch(from)
	} else {
		return 0, errors.New("processor: could not parse duration string")
	}

	for i, name := range iso8601Regex.SubexpNames() {
		part := match[i]
		if i == 0 || name == "" || part == "" {
			continue
		}

		val, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("reader/processor: %w", err)
		}

		switch name {
		case "hour":
			d += time.Duration(val) * time.Hour
		case "minute":
			d += time.Duration(val) * time.Minute
		case "second":
			d += time.Duration(val) * time.Second
		default:
			return 0, fmt.Errorf("processor: unknown field %s", name)
		}
	}

	return d, nil
}
