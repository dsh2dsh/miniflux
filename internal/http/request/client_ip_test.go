// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package request // import "miniflux.app/v2/internal/http/request"

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func withTrustedProxies(t *testing.T, proxies ...string) func(ip string) bool {
	t.Helper()

	m := make(map[string]struct{}, len(proxies))
	for _, ip := range proxies {
		m[ip] = struct{}{}
	}

	return func(ip string) bool {
		_, ok := m[ip]
		return ok
	}
}

func TestFindClientIPWithoutHeaders(t *testing.T) {
	withoutProxy := withTrustedProxies(t)
	r := http.Request{RemoteAddr: "192.168.0.1:4242"}
	assert.Equal(t, "192.168.0.1", FindClientIP(&r, withoutProxy))

	r = http.Request{RemoteAddr: "192.168.0.1"}
	assert.Equal(t, "192.168.0.1", FindClientIP(&r, withoutProxy))

	r = http.Request{RemoteAddr: "fe80::14c2:f039:edc7:edc7"}
	assert.Equal(t, "fe80::14c2:f039:edc7:edc7", FindClientIP(&r, withoutProxy))

	r = http.Request{RemoteAddr: "fe80::14c2:f039:edc7:edc7%eth0"}
	assert.Equal(t, "fe80::14c2:f039:edc7:edc7", FindClientIP(&r, withoutProxy))

	r = http.Request{RemoteAddr: "[fe80::14c2:f039:edc7:edc7%eth0]:4242"}
	assert.Equal(t, "fe80::14c2:f039:edc7:edc7", FindClientIP(&r, withoutProxy))
}

func TestFindClientIPWithXFFHeader(t *testing.T) {
	// Test with multiple IPv4 addresses.
	headers := http.Header{}
	headers.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
	r := http.Request{RemoteAddr: "192.168.0.1:4242", Header: headers}
	assert.Equal(t, "203.0.113.195",
		FindClientIP(&r, withTrustedProxies(t, "70.41.3.18", "150.172.238.178")))

	assert.Equal(t, "70.41.3.18",
		FindClientIP(&r, withTrustedProxies(t, "150.172.238.178")))
	assert.Equal(t, "150.172.238.178", FindClientIP(&r, withTrustedProxies(t)))
	assert.Equal(t, "192.168.0.1", FindClientIP(&r,
		withTrustedProxies(t, "203.0.113.195", "70.41.3.18", "150.172.238.178")))

	// Test with single IPv6 address.
	headers = http.Header{}
	headers.Set("X-Forwarded-For", "2001:db8:85a3:8d3:1319:8a2e:370:7348")
	r = http.Request{RemoteAddr: "192.168.0.1:4242", Header: headers}
	assert.Equal(t, "2001:db8:85a3:8d3:1319:8a2e:370:7348",
		FindClientIP(&r, withTrustedProxies(t)))
	assert.Equal(t, "192.168.0.1", FindClientIP(&r,
		withTrustedProxies(t, "2001:db8:85a3:8d3:1319:8a2e:370:7348")))

	// Test with single IPv6 address with zone
	headers = http.Header{}
	headers.Set("X-Forwarded-For", "fe80::14c2:f039:edc7:edc7%eth0")
	r = http.Request{RemoteAddr: "192.168.0.1:4242", Header: headers}
	assert.Equal(t, "fe80::14c2:f039:edc7:edc7",
		FindClientIP(&r, withTrustedProxies(t)))

	// Test with single IPv4 address.
	headers = http.Header{}
	headers.Set("X-Forwarded-For", "70.41.3.18")
	r = http.Request{RemoteAddr: "192.168.0.1:4242", Header: headers}
	assert.Equal(t, "70.41.3.18", FindClientIP(&r, withTrustedProxies(t)))

	// Test with invalid IP address.
	headers = http.Header{}
	headers.Set("X-Forwarded-For", "fake IP")
	r = http.Request{RemoteAddr: "192.168.0.1:4242", Header: headers}
	assert.Equal(t, "192.168.0.1", FindClientIP(&r, withTrustedProxies(t)))
}

func TestClientIPWithXRealIPHeader(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Real-Ip", "192.168.122.1")
	r := http.Request{RemoteAddr: "192.168.0.1:4242", Header: headers}
	assert.Equal(t, "192.168.122.1", FindClientIP(&r, withTrustedProxies(t)))
}

func TestClientIPWithBothHeaders(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
	headers.Set("X-Real-Ip", "192.168.122.1")

	r := http.Request{RemoteAddr: "192.168.0.1:4242", Header: headers}
	assert.Equal(t, "203.0.113.195",
		FindClientIP(&r, withTrustedProxies(t, "70.41.3.18", "150.172.238.178")))
	assert.Equal(t, "70.41.3.18",
		FindClientIP(&r, withTrustedProxies(t, "150.172.238.178")))
	assert.Equal(t, "150.172.238.178", FindClientIP(&r, withTrustedProxies(t)))
}

func TestClientIPWithUnixSocketRemoteAddress(t *testing.T) {
	r := http.Request{RemoteAddr: "@"}
	assert.Equal(t, "@", FindClientIP(&r, withTrustedProxies(t)))
}

func TestClientIPWithUnixSocketRemoteAddrAndBothHeaders(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
	headers.Set("X-Real-Ip", "192.168.122.1")

	r := http.Request{RemoteAddr: "@", Header: headers}
	assert.Equal(t, "203.0.113.195",
		FindClientIP(&r, withTrustedProxies(t, "70.41.3.18", "150.172.238.178")))
	assert.Equal(t, "150.172.238.178", FindClientIP(&r, withTrustedProxies(t)))
}
