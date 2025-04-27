// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/validator"
)

var resetPassCmd = cobra.Command{
	Use:   "reset-password",
	Short: "Reset user password",
	Args:  cobra.ExactArgs(0),

	RunE: func(cmd *cobra.Command, args []string) error {
		return withStorage(resetPassword)
	},
}

func resetPassword(store *storage.Storage) error {
	ctx := context.Background()
	username, password := askCredentials()
	user, err := store.UserByUsername(ctx, username)
	if err != nil {
		return err
	} else if user == nil {
		return errors.New("user not found")
	}

	userModificationRequest := model.UserModificationRequest{
		Password: &password,
	}
	validationErr := validator.ValidateUserModification(ctx, store, user.ID,
		&userModificationRequest)
	if validationErr != nil {
		return validationErr.Error()
	}

	user.Password = password
	if err := store.UpdateUser(ctx, user); err != nil {
		return err
	}

	fmt.Println("Password changed!")
	return nil
}
