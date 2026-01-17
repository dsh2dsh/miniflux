// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	"miniflux.app/v2/internal/model"
)

type feedIconResponse struct {
	ID       int64  `json:"id"`
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type entriesResponse struct {
	Total   int           `json:"total"`
	Entries model.Entries `json:"entries"`
}

type integrationsStatusResponse struct {
	HasIntegrations bool `json:"has_integrations,omitzero"`
}

type entryContentResponse struct {
	Content     string `json:"content,omitzero"`
	ReadingTime int    `json:"reading_time,omitzero"`
}

type feedCreationResponse struct {
	FeedID int64 `json:"feed_id,omitzero"`
}

type importFeedsResponse struct {
	Message string `json:"message,omitzero"`
}

type VersionResponse struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Compiler  string `json:"compiler"`
	Arch      string `json:"arch"`
	OS        string `json:"os"`
}
