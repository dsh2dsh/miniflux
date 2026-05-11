// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package processor // import "miniflux.app/v2/internal/reader/processor"

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseISO8601Duration parses a subset of ISO8601 durations, mainly for youtube video.
func parseISO8601Duration(duration string) (time.Duration, error) {
	after, ok := strings.CutPrefix(duration, "PT")
	if !ok {
		return 0, errors.New("the period doesn't start with PT")
	}

	var d time.Duration
	start := 0

	for i := 0; i < len(after); i++ {
		var unit time.Duration

		switch after[i] {
		case 'Y', 'W', 'D':
			return 0, fmt.Errorf("the '%c' specifier isn't supported", after[i])
		case 'H':
			unit = time.Hour
		case 'M':
			unit = time.Minute
		case 'S':
			unit = time.Second
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			continue
		default:
			return 0, errors.New("invalid character in the period")
		}

		val, err := strconv.Atoi(after[start:i])
		if err != nil {
			return 0, fmt.Errorf(
				"reader/processor: parsing %q as duration: %w", duration, err)
		}
		d += time.Duration(val) * unit
		start = i + 1
	}
	return d, nil
}
