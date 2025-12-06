// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rewrite // import "miniflux.app/v2/internal/reader/rewrite"

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"text/scanner"

	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/urllib"
)

func ApplyContentRewriteRules(ctx context.Context, entry *model.Entry,
	userRules string,
) {
	contentRules := ContentRewrite{rules: parseRules(userRules)}
	contentRules.Apply(ctx, entry)
}

func parseRules(s string) []rule {
	scan := scanner.Scanner{Mode: scanner.ScanIdents | scanner.ScanStrings}
	scan.Init(strings.NewReader(s))

	var rules []rule
	for {
		switch scan.Scan() {
		case scanner.Ident:
			rules = append(rules, rule{name: scan.TokenText()})
		case scanner.String:
			if len(rules) == 0 {
				continue
			}
			s, _ := strconv.Unquote(scan.TokenText())
			last := len(rules) - 1
			rules[last].args = append(rules[last].args, s)
		case scanner.EOF:
			return rules
		}
	}
}

type ContentRewrite struct {
	rules []rule
}

func NewContentRewrite(userRules string) *ContentRewrite {
	return &ContentRewrite{rules: parseRules(userRules)}
}

func (self *ContentRewrite) Apply(ctx context.Context, entry *model.Entry) {
	rules := self.rules
	if len(rules) == 0 {
		rules = findDomainRule(urllib.Domain(entry.URL))
	}

	logging.FromContext(ctx).Debug("Applying rewrite rules",
		slog.Any("rules", rules), slog.String("entry_url", entry.URL))

	for _, r := range rules {
		self.applyRule(ctx, entry, r)
	}
	entry.Content = addPDFLink(entry.URL, entry.Content)
}

func (self *ContentRewrite) applyRule(ctx context.Context, entry *model.Entry,
	r rule,
) {
	switch r.name {
	case "add_image_title":
		entry.Content = addImageTitle(entry.Content)
	case "add_mailto_subject":
		entry.Content = addMailtoSubject(entry.Content)
	case "add_dynamic_image":
		entry.Content = addDynamicImage(entry.Content)
	case "add_dynamic_iframe":
		entry.Content = addDynamicIframe(entry.Content)
	case "add_youtube_video":
		entry.Content = addYoutubeVideoRewriteRule(entry.URL, entry.Content)
	case "add_invidious_video":
		entry.Content = addInvidiousVideo(entry.URL, entry.Content)
	case "add_youtube_video_using_invidious_player":
		entry.Content = addYoutubeVideoUsingInvidiousPlayer(entry.URL, entry.Content)
	case "add_youtube_video_from_id":
		entry.Content = addYoutubeVideoFromId(entry.Content)
	case "add_pdf_download_link":
		entry.Content = addPDFLink(entry.URL, entry.Content)
	case "nl2br":
		entry.Content = strings.ReplaceAll(entry.Content, "\n", "<br>")
	case "convert_text_link", "convert_text_links":
		entry.Content = replaceTextLinks(entry.Content)
	case "fix_medium_images":
		entry.Content = fixMediumImages(entry.Content)
	case "use_noscript_figure_images":
		entry.Content = useNoScriptImages(entry.Content)
	case "replace":
		r.applyReplaceContent(ctx, entry)
	case "replace_title":
		r.applyReplaceTitle(ctx, entry)
	case "remove":
		r.applyRemove(ctx, entry)
	case "add_castopod_episode":
		entry.Content = addCastopodEpisode(entry.URL, entry.Content)
	case "base64_decode":
		r.applyBase64Decode(entry)
	case "add_hn_links_using_hack":
		entry.Content = addHackerNewsLinksUsing(entry.Content, "hack")
	case "add_hn_links_using_opener":
		entry.Content = addHackerNewsLinksUsing(entry.Content, "opener")
	case "remove_tables":
		entry.Content = removeTables(entry.Content)
	case "remove_clickbait":
		entry.Title = titlelize(entry.Title)
	case "fix_ghost_cards":
		entry.Content = fixGhostCards(entry.Content)
	case "remove_img_blur_params":
		entry.Content = removeImgBlurParams(entry.Content)
	}
}
