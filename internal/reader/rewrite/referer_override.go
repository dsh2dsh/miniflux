// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rewrite // import "miniflux.app/v2/internal/reader/rewrite"

import (
	"strings"

	"miniflux.app/v2/internal/urllib"
)

var domainReferrers = map[string]string{
	"appinn.com":           "https://appinn.com",
	"bjp.org.cn":           "https://bjp.org.cn",
	"cdnfile.sspai.com":    "https://sspai.com",
	"cdninstagram.com":     "https://www.instagram.com",
	"f.video.weibocdn.com": "https://weibo.com",
	"i.pximg.net":          "https://www.pixiv.net",
	"img.hellogithub.com":  "https://hellogithub.com",
	"moyu.im":              "https://i.jandan.net",
	"sinaimg.cn":           "https://weibo.com",
	"www.parkablogs.com":   "https://www.parkablogs.com",
}

// GetRefererForURL returns the referer for the given URL if it exists,
// otherwise an empty string.
func GetRefererForURL(u string) string {
	hostname := urllib.Domain(u)
	for {
		if s, ok := domainReferrers[hostname]; ok {
			return s
		}
		_, domain, ok := strings.Cut(hostname, ".")
		if !ok {
			return ""
		}
		hostname = domain
	}
}
