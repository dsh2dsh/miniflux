package sanitizer

import (
	"net/url"
	"strings"

	"github.com/dsh2dsh/bluemonday/v2"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"miniflux.app/v2/internal/config"
)

const iframeSandbox = "allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox"

var (
	allowIframes = []string{
		"bandcamp.com",
		"cdn.embedly.com",
		"dailymotion.com",
		"open.spotify.com",
		"player.bilibili.com",
		"player.twitch.tv",
		"player.vimeo.com",
		"soundcloud.com",
		"vk.com",
		"w.soundcloud.com",
		"youtube-nocookie.com",
		"youtube.com",
	}

	allowSchemes = []string{
		"apt",
		"bitcoin",
		"callto",
		"davs",
		"ed2k",
		"facetime",
		"feed",
		"ftp",
		"geo",
		"git",
		"gopher",
		"irc",
		"irc6",
		"ircs",
		"itms-apps",
		"itms",
		"magnet",
		"news",
		"nntp",
		"rtmp",
		"sftp",
		"sip",
		"sips",
		"skype",
		"spotify",
		"ssh",
		"steam",
		"svn",
		"svn+ssh",
		"tel",
		"webcal",
		"xmpp",

		// iOS Apps
		"opener", // https://www.opener.link
		"hack",   // https://apps.apple.com/it/app/hack-for-hacker-news-reader/id1464477788?l=en-GB
	}

	contentPolicy = bluemonday.UGCPolicy()
	titlePolicy   = bluemonday.StrictPolicy()

	allowedIframe = make(map[string]struct{})
)

func init() {
	p := contentPolicy
	p.AddTargetBlankToFullyQualifiedLinks(true)
	p.AllowDataURIImages()
	p.AllowURLSchemes(allowSchemes...)
	p.RequireNoReferrerOnLinks(true)

	p.AllowAttrs("id").DeleteFromGlobally()

	p.SetAttr("controls", "controls").OnElements("audio", "video").
		SetAttr("loading", "lazy").OnElements("iframe", "img").
		SetAttr("sandbox", iframeSandbox).OnElements("iframe")

	p.SetAttr("referrerpolicy", "strict-origin-when-cross-origin").
		OnElements("iframe")

	p.AllowAttrs("hidden").Globally()

	allowMathML(p)

	p.AllowAttrs("decoding").WithValues("sync", "async").OnElements("img").
		AllowAttrs("fetchpriority").WithValues("high", "low").OnElements("img")

	p.AllowAttrs("poster").OnElements("video").
		AllowAttrs("sizes").OnElements("img", "source").
		AllowAttrs("src").OnElements("audio", "iframe", "source", "video")

	p.AllowElements("picture").
		AllowAttrs("type", "media", "srcset").OnElements("source")

	p.AllowAttrs("height", "width").Matching(bluemonday.Number).
		OnElements("iframe", "video")

	p.AllowAttrs("allowfullscreen", "frameborder").OnElements("iframe")

	for _, hostname := range allowIframes {
		allowedIframe[hostname] = struct{}{}
	}
}

func StripTags(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	return titlePolicy.Sanitize(s)
}

func SanitizeContent(s string, pageURL *url.URL) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	p := rewritePolicy{p: *contentPolicy, pageURL: pageURL}
	p.init()
	return p.Sanitize(s)
}

type rewritePolicy struct {
	p       bluemonday.Policy
	pageURL *url.URL
}

func (self *rewritePolicy) Sanitize(s string) string {
	return self.p.Sanitize(s)
}

func (self *rewritePolicy) init() *rewritePolicy {
	self.p.RewriteTokenURL(self.rewriteURL)
	return self
}

func (self *rewritePolicy) rewriteURL(t *html.Token, u *url.URL) *url.URL {
	switch t.DataAtom {
	case atom.Iframe:
		return self.allowIframe(u)
	case atom.Img:
		if pixelTracker(t.Attr) {
			return nil
		}
	}

	if blockedURL(u) {
		return nil
	}
	StripTracking(u, self.pageURL.Hostname())

	if !u.IsAbs() {
		u = self.pageURL.ResolveReference(u)
	}
	return u
}

func (self *rewritePolicy) allowIframe(u *url.URL) *url.URL {
	if !u.IsAbs() {
		u = self.pageURL.ResolveReference(u)
	}
	rewriteIframeSrc(u)

	domain := strings.TrimPrefix(u.Hostname(), "www.")
	if _, ok := allowedIframe[domain]; ok {
		return u
	}

	if s := config.InvidiousInstance(); s != "" && strings.TrimPrefix(s, "www.") == domain {
		return u
	}

	if s := config.YouTubeEmbedDomain(); s != "" &&
		strings.TrimPrefix(s, "www.") == domain {
		return u
	}
	return nil
}

func rewriteIframeSrc(u *url.URL) bool {
	switch strings.TrimPrefix(u.Hostname(), "www.") {
	case "youtube.com":
		return rewriteYoutube(u)
	case "player.vimeo.com":
		return rewriteVimeo(u)
	}
	return false
}
