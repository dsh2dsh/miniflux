package sanitizer

import (
	"net/url"
	"slices"
	"strings"

	"github.com/dsh2dsh/bluemonday/v2"
	"golang.org/x/net/html"

	"miniflux.app/v2/internal/config"
)

var (
	blockedResources = map[string][]string{
		"api.flattr.com":       nil,
		"feeds.feedburner.com": nil,
		"feedsportal.com":      nil,
		"pinterest.com":        {"/pin/create/button/"},
		"stats.wordpress.com":  nil,
		"twitter.com":          {"/intent/tweet", "/share"},
		"facebook.com":         {"/sharer.php"},
		"linkedin.com":         {"/shareArticle"},
	}

	// Interesting lists:
	// https://raw.githubusercontent.com/AdguardTeam/AdguardFilters/master/TrackParamFilter/sections/general_url.txt
	// https://firefox.settings.services.mozilla.com/v1/buckets/main/collections/query-stripping/records
	// https://github.com/Smile4ever/Neat-URL/blob/master/data/default-params-by-category.json
	// https://github.com/brave/brave-core/blob/master/components/query_filter/utils.cc
	// https://developers.google.com/analytics/devguides/collection/ga4/reference/config
	trackingParams = []string{
		// Facebook Click Identifiers
		"fbclid",
		"_openstat",
		"fb_action_ids",
		"fb_action_types",
		"fb_ref",
		"fb_source",
		"fb_comment_id",

		// Humble Bundles
		"hmb_campaign",
		"hmb_medium",
		"hmb_source",

		// Likely Google as well
		"itm_campaign",
		"itm_medium",
		"itm_source",

		// Google Click Identifiers
		"gclid",
		"dclid",
		"gbraid",
		"wbraid",
		"gclsrc",

		// Google Analytics
		"campaign_id",
		"campaign_medium",
		"campaign_name",
		"campaign_source",
		"campaign_term",
		"campaign_content",

		// Google
		"srsltid",

		// Yandex Click Identifiers
		"yclid",
		"ysclid",

		// Twitter Click Identifier
		"twclid",

		// Microsoft Click Identifier
		"msclkid",

		// Mailchimp Click Identifiers
		"mc_cid",
		"mc_eid",
		"mc_tc",

		// Wicked Reports click tracking
		"wickedid",

		// Hubspot Click Identifiers
		"hsa_cam",
		"_hsenc",
		"__hssc",
		"__hstc",
		"__hsfp",
		"_hsmi",
		"hsctatracking",

		// Olytics
		"rb_clickid",
		"oly_anon_id",
		"oly_enc_id",

		// Vero Click Identifier
		"vero_id",
		"vero_conv",

		// Marketo email tracking
		"mkt_tok",

		// Adobe email tracking
		"sc_cid",

		// Beehiiv
		"_bhlid",

		// Branch.io
		"_branch_match_id",
		"_branch_referrer",

		// Readwise
		"__readwiseLocation",
	}

	// Outbound tracking parameters are appending the website's url to outbound links.
	trackingOutbound = []string{
		// Ghost
		"ref",
	}

	trackingPrefixes = []string{
		"utm_", // https://en.wikipedia.org/wiki/UTM_parameters
		"mtm_", // https://matomo.org/faq/reports/common-campaign-tracking-use-cases-and-examples/
	}

	tracking    = make(map[string]struct{}, len(trackingParams))
	trackingRef = make(map[string]struct{}, len(trackingOutbound))
)

func init() {
	for _, s := range trackingParams {
		tracking[s] = struct{}{}
	}

	for _, s := range trackingOutbound {
		trackingRef[s] = struct{}{}
	}
}

func StripTracking(u *url.URL, refHostnames ...string) bool {
	if u.RawQuery == "" {
		return false
	}

	var hasTrackers bool
	query := u.Query()

	// Remove tracking parameters
	for param := range query {
		key := strings.ToLower(param)
		if trackingParam(key) {
			query.Del(param)
			hasTrackers = true
			continue
		}

		if _, ok := trackingRef[key]; ok {
			// handle duplicate parameters like ?a=b&a=c&a=d…
			for _, ref := range query[param] {
				if slices.Contains(refHostnames, ref) {
					query.Del(param)
					hasTrackers = true
					break
				}
			}
		}
	}

	if hasTrackers {
		u.RawQuery = query.Encode()
	}
	return hasTrackers
}

func trackingParam(key string) bool {
	if _, ok := tracking[key]; ok {
		return true
	}

	for _, prefix := range trackingPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func allowMathML(p *bluemonday.Policy) {
	p.AllowAttrs("xmlns").OnElements("math")
	p.AllowNoAttrs().OnElements(
		"annotation",
		"annotation-xml",
		"maction",
		"merror",
		"mfrac",
		"mi",
		"mmultiscripts",
		"mn",
		"mo",
		"mover",
		"mpadded",
		"mphantom",
		"mprescripts",
		"mroot",
		"mrow",
		"ms",
		"mspace",
		"msqrt",
		"mstyle",
		"msub",
		"msubsup",
		"msup",
		"mtable",
		"mtd",
		"mtext",
		"mtr",
		"munder",
		"munderover",
		"semantics",
	)
}

func blockedURL(u *url.URL) bool {
	hostname, p := u.Hostname(), u.EscapedPath()
	for {
		if paths, ok := blockedResources[hostname]; ok {
			if len(paths) == 0 {
				return true
			}
			return slices.ContainsFunc(paths, func(s string) bool {
				return strings.Contains(p, s)
			})
		}

		_, h, ok := strings.Cut(hostname, ".")
		if !ok {
			break
		}
		hostname = h
	}
	return false
}

func pixelTracker(attrs []html.Attribute) bool {
	var height, width bool
	for _, attr := range attrs {
		if attr.Val == "0" || attr.Val == "1" {
			switch attr.Key {
			case "height":
				height = true
			case "width":
				width = true
			}
		}
	}
	return height && width
}

func rewriteVimeo(u *url.URL) bool {
	// See https://help.vimeo.com/hc/en-us/articles/12426260232977-About-Player-parameters
	if !strings.HasPrefix(u.Path, "/video/") {
		return false
	}

	if u.RawQuery == "" {
		u.RawQuery = "dnt=1"
	} else {
		u.RawQuery += "&dnt=1"
	}
	return true
}

func rewriteYoutube(u *url.URL) bool {
	afterEmbed, ok := strings.CutPrefix(u.EscapedPath(), "/embed/")
	if !ok {
		return false
	}

	u2 := *config.YouTubeEmbedURL()
	u2.RawQuery = u.RawQuery
	*u = *u2.JoinPath(afterEmbed)
	return true
}
