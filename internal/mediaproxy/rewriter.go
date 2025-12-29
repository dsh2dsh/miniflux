// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mediaproxy // import "miniflux.app/v2/internal/mediaproxy"

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"slices"
	"strings"

	"github.com/dsh2dsh/bluemonday/v2"
	"golang.org/x/net/html/atom"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/mux"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/sanitizer"
)

var relRoot = url.URL{Path: "/"}

func RewriteDocumentWithRelativeProxyURL(m *mux.ServeMux, content string,
) string {
	p := proxyRewriter{m: m}
	return p.Proxify(content)
}

func RewriteDocumentWithAbsoluteProxyURL(m *mux.ServeMux, content string,
) string {
	p := proxyRewriter{m: m}
	return p.WithAbsoluteProxy().Proxify(content)
}

func ProxifyEnclosures(m *mux.ServeMux, enclosures []model.Enclosure) {
	for i := range enclosures {
		e := &enclosures[i]
		e.URL = proxifyAbsoluteURL(m, e.MimeType, e.URL)
	}
}

func proxifyAbsoluteURL(m *mux.ServeMux, mimeType, mediaURL string) string {
	if mediaURL == "" {
		return mediaURL
	}

	u, err := url.Parse(mediaURL)
	if err != nil {
		return mediaURL
	}

	p := proxyRewriter{m: m}
	if !p.ShouldProxifyScheme(u.Scheme) {
		return mediaURL
	}

	i := slices.IndexFunc(config.MediaProxyResourceTypes(),
		func(mediaType string) bool {
			return strings.HasPrefix(mimeType, mediaType+"/")
		})
	if i == -1 {
		return mediaURL
	}

	u = p.WithAbsoluteProxy().proxify(u)
	if u == nil {
		return mediaURL
	}
	return u.String()
}

type proxyRewriter struct {
	m *mux.ServeMux

	absProxy bool

	mode  string
	audio bool
	image bool
	video bool
}

func New(m *mux.ServeMux) *proxyRewriter {
	return &proxyRewriter{m: m}
}

func (self *proxyRewriter) WithAbsoluteProxy() *proxyRewriter {
	self.absProxy = true
	return self
}

func (self *proxyRewriter) Proxify(content string) string {
	if !self.mediaTypes() {
		return content
	}
	return sanitizer.Proxify(content, self.RewriteURL)
}

func (self *proxyRewriter) mediaTypes() (ok bool) {
	self.mode = config.MediaProxyMode()
	if self.mode == "none" {
		return false
	}

	for _, s := range config.MediaProxyResourceTypes() {
		switch s {
		case "audio":
			self.audio = true
			ok = true
		case "image":
			self.image = true
			ok = true
		case "video":
			self.video = true
			ok = true
		}
	}
	return ok
}

func (self *proxyRewriter) RewriteURL(t *bluemonday.Token, attr string,
	u *url.URL,
) *url.URL {
	if !self.shouldProxify(t, attr, u) {
		return u
	}
	return self.proxify(u)
}

func (self *proxyRewriter) shouldProxify(t *bluemonday.Token, attr string,
	u *url.URL,
) bool {
	var ok bool
	switch t.DataAtom {
	case atom.Audio:
		ok = self.audio
	case atom.Img:
		ok = self.image

	case atom.Video:
		switch attr {
		case "poster":
			ok = self.image
		default:
			ok = self.video
		}

	case atom.Source:
		switch t.ParentAtom() {
		case atom.Audio:
			ok = self.audio
		case atom.Img, atom.Picture:
			ok = self.image
		case atom.Video:
			ok = self.video
		}
	}
	return ok && self.schemeOk(u.Scheme)
}

func (self *proxyRewriter) schemeOk(scheme string) bool {
	switch {
	case strings.EqualFold(scheme, "data"):
		return false
	case self.mode == "all":
		return true
	}
	return self.mode != "none" && !strings.EqualFold(scheme, "https")
}

func (self *proxyRewriter) proxify(u *url.URL) *url.URL {
	urlBytes := []byte(u.String())
	if customProxy := config.MediaCustomProxyURL(); customProxy != nil {
		return customProxy.JoinPath(
			base64.URLEncoding.EncodeToString(urlBytes))
	}

	mac := hmac.New(sha256.New, config.MediaProxyPrivateKey())
	mac.Write(urlBytes)
	digest := mac.Sum(nil)

	proxyPath := route.Path(self.m, "proxy",
		"encodedDigest", base64.URLEncoding.EncodeToString(digest),
		"encodedURL", base64.URLEncoding.EncodeToString(urlBytes))

	if self.absProxy {
		return config.Root().JoinPath(proxyPath)
	}
	return relRoot.JoinPath(proxyPath)
}

func (self *proxyRewriter) ShouldProxifyScheme(scheme string) bool {
	return self.mediaTypes() && self.schemeOk(scheme)
}
