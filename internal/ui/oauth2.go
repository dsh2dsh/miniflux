// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/oauth2"
)

func getOAuth2Manager(ctx context.Context) *oauth2.Manager {
	return oauth2.NewManager(
		ctx,
		config.OAuth2ClientID(),
		config.OAuth2ClientSecret(),
		config.OAuth2RedirectURL(),
		config.OIDCDiscoveryEndpoint(),
	)
}
