// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package request

import (
	"net"
	"net/http"
	"slices"
	"strings"
)

// FindClientIP returns the real client IP address using trusted reverse-proxy
// headers when allowed.
func FindClientIP(r *http.Request, trustedProxy func(ip string) bool) string {
	clientIP := FindRemoteIP(r)
	if !trustedProxy(clientIP) {
		return clientIP
	}

	if ip := XForwardedFor(r, trustedProxy); ip != "" {
		return ip
	}

	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		ip = dropIPv6zone(ip)
		if net.ParseIP(ip) != nil {
			return ip
		}
	}
	// Fallback to TCP/IP source IP address.
	return clientIP
}

func XForwardedFor(r *http.Request, trustedProxy func(ip string) bool) string {
	values := r.Header.Values("X-Forwarded-For")
	for _, value := range slices.Backward(values) {
		items := strings.Split(value, ",")
		for _, ip := range slices.Backward(items) {
			if ip = strings.TrimSpace(ip); trustedProxy(ip) {
				continue
			}
			if ip = dropIPv6zone(ip); net.ParseIP(ip) == nil {
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
	switch r.RemoteAddr {
	case "", "@":
		return "127.0.0.1"
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	return dropIPv6zone(ip)
}
