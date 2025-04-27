// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/validator"
)

var createAdminCmd = cobra.Command{
	Use:   "create-admin",
	Short: "Create an admin user from an interactive terminal",
	Args:  cobra.ExactArgs(0),

	RunE: func(cmd *cobra.Command, args []string) error {
		return withStorage(createAdminUserFromInteractiveTerminal)
	},
}

func createAdminUserFromEnvironmentVariables(store *storage.Storage) error {
	return createAdminUser(context.Background(), store,
		config.Opts.AdminUsername(),
		config.Opts.AdminPassword())
}

func createAdminUserFromInteractiveTerminal(store *storage.Storage) error {
	username, password := askCredentials()
	return createAdminUser(context.Background(), store, username, password)
}

func createAdminUser(ctx context.Context, store *storage.Storage, username,
	password string,
) error {
	userCreationRequest := model.UserCreationRequest{
		Username: username,
		Password: password,
		IsAdmin:  true,
	}

	if store.UserExists(ctx, userCreationRequest.Username) {
		slog.Info("Skipping admin user creation because it already exists",
			slog.String("username", userCreationRequest.Username),
		)
		return nil
	}

	validateErr := validator.ValidateUserCreationWithPassword(ctx, store,
		&userCreationRequest)
	if validateErr != nil {
		return validateErr.Error()
	}

	user, err := store.CreateUser(ctx, &userCreationRequest)
	if err != nil {
		return err
	}

	slog.Info("Created new admin user",
		slog.String("username", user.Username),
		slog.Int64("user_id", user.ID))
	return nil
}
