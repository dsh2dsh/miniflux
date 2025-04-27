// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Timezones returns all timezones supported by the database.
func (s *Storage) Timezones(ctx context.Context) (map[string]string, error) {
	rows, _ := s.db.Query(ctx,
		`SELECT name FROM pg_timezone_names ORDER BY name ASC`)

	timezones := make(map[string]string)
	var timezone string
	_, err := pgx.ForEachRow(rows, []any{&timezone}, func() error {
		switch {
		case timezone == "localtime":
		case strings.HasPrefix(timezone, "posix"):
		case strings.HasPrefix(timezone, "SystemV"):
		default:
			timezones[timezone] = timezone
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch timezones: %w`, err)
	}
	return timezones, nil
}
