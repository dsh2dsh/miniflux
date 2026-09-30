// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package proxyrotator

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

func TestProxyRotator(t *testing.T) {
	proxyURLs := []*url.URL{
		mustParse(t, "http://proxy1.example.com"),
		mustParse(t, "http://proxy2.example.com"),
		mustParse(t, "http://proxy3.example.com"),
	}

	rotator := NewProxyRotator(proxyURLs)
	if !rotator.HasProxies() {
		t.Fatalf("Expected rotator to have proxies")
	}

	seenProxies := make(map[string]bool)
	for range len(proxyURLs) * 2 {
		proxy := rotator.GetNextProxy()
		if proxy == nil {
			t.Fatalf("Expected a proxy, got nil")
		}

		seenProxies[proxy.String()] = true
	}

	if len(seenProxies) != len(proxyURLs) {
		t.Fatalf("Expected to see all proxies, but saw: %v", seenProxies)
	}
}

func TestProxyRotatorEmpty(t *testing.T) {
	rotator := NewProxyRotator([]*url.URL{})
	if rotator.HasProxies() {
		t.Fatalf("Expected rotator to have no proxies")
	}

	proxy := rotator.GetNextProxy()
	if proxy != nil {
		t.Fatalf("Expected no proxy, got: %v", proxy)
	}
}
