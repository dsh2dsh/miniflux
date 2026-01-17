// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package request // import "miniflux.app/v2/internal/http/request"

import (
	"net"
	"net/http"
	"slices"
	"strings"
)

// FindClientIP returns the real client IP address using trusted reverse-proxy
// headers when allowed.
func FindClientIP(r *http.Request, trustedProxy func(ip string) bool) string {
	if clientIP := XForwardedFor(r, trustedProxy); clientIP != "" {
		return clientIP
	}

	clientIP := r.Header.Get("X-Real-IP")
	if clientIP != "" {
		clientIP = dropIPv6zone(strings.TrimSpace(clientIP))
		if net.ParseIP(clientIP) != nil {
			return clientIP
		}
	}

	// Fallback to TCP/IP source IP address.
	return FindRemoteIP(r)
}

func XForwardedFor(r *http.Request, trustedProxy func(ip string) bool) string {
	values := r.Header.Values("X-Forwarded-For")
	for _, value := range slices.Backward(values) {
		items := strings.Split(value, ",")
		for _, ip := range slices.Backward(items) {
			ip = strings.TrimSpace(ip)
			if trustedProxy(ip) {
				continue
			}
			ip = dropIPv6zone(ip)
			if net.ParseIP(ip) == nil {
				return ""
			}
			return ip
		}
	}
	return ""
}

func dropIPv6zone(address string) string {
	before, _, _ := strings.Cut(address, "%")
	return before
}

// FindRemoteIP returns the remote client IP address without considering HTTP
// headers.
func FindRemoteIP(r *http.Request) string {
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteIP = r.RemoteAddr
	}
	return dropIPv6zone(remoteIP)
}
