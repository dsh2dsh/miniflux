// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"database/sql"
	"fmt"

	"github.com/spf13/cobra"

	"miniflux.app/v2/internal/storage"
)

var flushSessionsCmd = cobra.Command{
	Use:   "flush-sessions",
	Short: "Flush all sessions (disconnect users)",
	Args:  cobra.ExactArgs(0),

	RunE: func(cmd *cobra.Command, args []string) error {
		return withStorage(func(_ *sql.DB, store *storage.Storage) error {
			return flushSessions(store)
		})
	},
}

func flushSessions(store *storage.Storage) error {
	fmt.Println("Flushing all sessions (disconnect users)")
	if err := store.FlushAllSessions(); err != nil {
		return err
	}
	return nil
}
