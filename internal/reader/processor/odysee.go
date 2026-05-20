// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package processor // import "miniflux.app/v2/internal/reader/processor"

import (
	"context"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/urllib"
)

func shouldFetchOdyseeWatchTime(entry *model.Entry) bool {
	if !config.FetchOdyseeWatchTime() {
		return false
	}

	return urllib.DomainWithoutWWW(entry.URL) == "odysee.com"
}

func fetchOdyseeWatchTime(ctx context.Context, websiteURL string) (int, error) {
	return fetchWatchTime(ctx, websiteURL, `meta[property="og:video:duration"]`, false)
}
