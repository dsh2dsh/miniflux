// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"miniflux.app/v2/internal/reader/opml"
	"miniflux.app/v2/internal/storage"
)

var exportUserFeedsCmd = cobra.Command{
	Use:   "export-user-feeds username",
	Short: "Export user feeds",

	Args: cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		return withStorage(
			func(ctx context.Context, store *storage.Storage) error {
				return exportUserFeeds(ctx, store, args[0])
			})
	},
}

func exportUserFeeds(ctx context.Context, store *storage.Storage,
	username string,
) error {
	user, err := store.UserByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("unable to find user: %w", err)
	} else if user == nil {
		return fmt.Errorf("user %q not found", username)
	}

	opmlExport, err := opml.NewHandler(store).Export(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("unable to export feeds: %w", err)
	}

	fmt.Println(opmlExport)
	return nil
}
