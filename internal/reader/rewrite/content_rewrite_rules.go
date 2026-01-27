// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rewrite // import "miniflux.app/v2/internal/reader/rewrite"

import "strings"

var (
	addImageTitleRules  = []rule{{name: "add_image_title"}}
	addDynamicImageRule = rule{name: "add_dynamic_image"}
)

// List of predefined rewrite rules (alphabetically sorted).
//
// See https://miniflux.app/docs/rules.html#rewrite-rules
var domainRules = map[string][]rule{
	"abstrusegoose.com":      addImageTitleRules,
	"amazingsuperpowers.com": addImageTitleRules,

	"bleepingcomputer.com": {
		addDynamicImageRule,
		{
			name: "remove",
			args: []string{".ia_ad, .cz-related-article-wrapp, div[align]"},
		},
	},

	"blog.cloudflare.com": {
		addImageTitleRules[0],
		{
			name: "remove",
			args: []string{"figure.kg-image-card figure.kg-image + img"},
		},
	},

	"cowbirdsinlove.com":    addImageTitleRules,
	"drawingboardcomic.com": addImageTitleRules,
	"exocomics.com":         addImageTitleRules,
	"explainxkcd.com":       addImageTitleRules,
	"framatube.org":         {{name: "nl2br"}, {name: "convert_text_link"}},
	"happletea.com":         addImageTitleRules,

	"ilpost.it": {
		{
			name: "remove",
			args: []string{".art_tag, #audioPlayerArticle, .author-container, .caption, .ilpostShare, .lastRecents, #mc_embed_signup, .outbrain_inread, p:has(.leggi-anche), .youtube-overlay"},
		},
	},

	"imogenquest.net":  addImageTitleRules,
	"lukesurl.com":     addImageTitleRules,
	"medium.com":       {{name: "fix_medium_images"}},
	"mercworks.net":    addImageTitleRules,
	"monkeyuser.com":   addImageTitleRules,
	"mrlovenstein.com": addImageTitleRules,
	"nedroid.com":      addImageTitleRules,

	"oglaf.com": {
		{
			name: "replace",
			args: []string{
				"media.oglaf.com/story/tt(.+).gif",
				"media.oglaf.com/comic/$1.jpg",
			},
		},
		addImageTitleRules[0],
	},

	"optipess.com":   addImageTitleRules,
	"peebleslab.com": addImageTitleRules,

	"phoronix.com": {
		{
			name: "remove",
			args: []string{"img[src^='/assets/categories/']"},
		},
	},

	"quantamagazine.org": {
		{name: "add_youtube_video_from_id"},
		{
			name: "remove",
			args: []string{"h6:not(.byline,.post__title__kicker), #comments, .next-post__content, .footer__section, figure .outer--content, script"},
		},
	},

	"qwantz.com":             {addImageTitleRules[0], {name: "add_mailto_subject"}},
	"sentfromthemoon.com":    addImageTitleRules,
	"thedoghousediaries.com": addImageTitleRules,

	"theverge.com": {
		addDynamicImageRule,
		{
			name: "remove",
			args: []string{"div.duet--recirculation--related-list, .hidden"},
		},
	},

	"treelobsters.com": addImageTitleRules,
	"xkcd.com":         addImageTitleRules,
	"youtube.com":      {{name: "add_youtube_video"}},
}

func findDomainRule(hostname string) []rule {
	for {
		if rules, ok := domainRules[hostname]; ok {
			return rules
		}
		_, domain, ok := strings.Cut(hostname, ".")
		if !ok {
			return nil
		}
		hostname = domain
	}
}
