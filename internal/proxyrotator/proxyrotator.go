// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package proxyrotator

import (
	"net/url"
	"sync"
)

// ProxyRotator manages a list of proxies and rotates through them.
type ProxyRotator struct {
	proxies      []*url.URL
	currentIndex int
	mutex        sync.Mutex
}

// NewProxyRotator creates a new ProxyRotator with the given proxy URLs.
func NewProxyRotator(proxies []*url.URL) *ProxyRotator {
	return &ProxyRotator{
		proxies:      proxies,
		currentIndex: 0,
		mutex:        sync.Mutex{},
	}
}

// GetNextProxy returns the next proxy in the rotation.
func (pr *ProxyRotator) GetNextProxy() *url.URL {
	if len(pr.proxies) == 0 {
		return nil
	}

	pr.mutex.Lock()
	proxy := pr.proxies[pr.currentIndex]
	pr.currentIndex = (pr.currentIndex + 1) % len(pr.proxies)
	pr.mutex.Unlock()

	return proxy
}

// HasProxies checks if there are any proxies available in the rotator.
func (pr *ProxyRotator) HasProxies() bool {
	return len(pr.proxies) > 0
}
