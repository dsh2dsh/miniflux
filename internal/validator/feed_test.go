// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/model"
)

func TestValidateFeedModificationProxyURL(t *testing.T) {
	require.NoError(t, config.Load("", config.WithYAMLString(`
proxies:
  - name: "Proxy 1"
    id:   "proxy1"
    url:  "http://127.0.0.1:3128"`)))

	tests := []struct {
		name     string
		proxyURL string
		wantErr  bool
	}{
		{
			name:     "empty proxy URL",
			proxyURL: "",
			wantErr:  false,
		},
		{
			name:     "valid proxy URL",
			proxyURL: "http://127.0.0.1:3128",
			wantErr:  false,
		},
		{
			name:     "invalid proxy URL",
			proxyURL: "example.org",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &model.FeedModificationRequest{ProxyURL: &tt.proxyURL}
			err := ValidateFeedModification(t.Context(), nil, 0, 0, r)
			if tt.wantErr {
				require.NotNil(t, err, "expected error")
				return
			}
			require.Nil(t, err, "got %v", err)
		})
	}
}

func TestValidateScraperRules(t *testing.T) {
	tests := []struct {
		rules   string
		wantErr bool
	}{
		{"", false},
		{"article", false},
		{"div.content, .post-body > p", false},
		{"div:has(> p)", true},
		{"div[", true},
		{"p:first", true},
	}
	for _, tt := range tests {
		err := ValidateScraperRules(t.Context(), tt.rules)
		if tt.wantErr {
			t.Log(err)
			require.NotNil(t, err, "expected error for %q", tt.rules)
			continue
		}
		require.Nil(t, err, "unexpected error for %q", tt.rules)
	}
}

func TestValidateScraperRules_invalidSelector(t *testing.T) {
	err := ValidateScraperRules(t.Context(), `remove("div:has(> p)")`)
	require.NotNil(t, err, "expected an error for an invalid selector")
}
