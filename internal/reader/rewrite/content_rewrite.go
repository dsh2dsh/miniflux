// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rewrite // import "miniflux.app/v2/internal/reader/rewrite"

import (
	"context"
	"html"
	"log/slog"
	"strconv"
	"strings"
	"text/scanner"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/sanitizer"
	"miniflux.app/v2/internal/template"
	"miniflux.app/v2/internal/urllib"
)

type ContentRewrite struct {
	rules     []rule
	templates *template.Engine
	user      *model.User

	sanitized bool
}

func NewContentRewrite(userRules string, u *model.User, t *template.Engine,
) *ContentRewrite {
	return &ContentRewrite{rules: parseRules(userRules), user: u, templates: t}
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

func (self *ContentRewrite) Sanitized() bool { return self.sanitized }

func (self *ContentRewrite) Apply(ctx context.Context, entry *model.Entry) {
	rules := self.rules
	if len(rules) == 0 {
		rules = findDomainRule(urllib.Domain(entry.URL))
	}

	logging.FromContext(ctx).Debug("Applying content rewrite rules",
		slog.Any("rules", rules), slog.String("entry_url", entry.URL))

	self.sanitized = false
	for _, r := range rules {
		self.applyRule(ctx, entry, r)
	}
	entry.Content = addPDFLink(entry.URL, entry.Content)
}

func (self *ContentRewrite) applyRule(ctx context.Context, entry *model.Entry,
	r rule,
) {
	log := logging.FromContext(ctx).With(
		slog.Group("rule", slog.String("name", r.name), slog.Any("args", r.args)),
		slog.String("entry_url", entry.URL))
	log.Debug("Applying content rewrite rule")

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
		self.youtubeIframe(log, entry)
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
		r.applyReplaceContent(log, entry)
	case "replace_title":
		r.applyReplaceTitle(log, entry)
	case "remove":
		r.applyRemove(log, entry)
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
	case "html_unescape":
		self.unsafeContent(entry, html.UnescapeString)
	}
}

func (self *ContentRewrite) unsafeContent(entry *model.Entry,
	fn func(string) string,
) {
	entry.Content = fn(entry.Content)
	self.sanitized = false
}

func (self *ContentRewrite) youtubeIframe(log *slog.Logger, entry *model.Entry,
) {
	yt := youtubeVideo(entry)
	if yt.VideoId == "" {
		log.Warn("Cannot find Youtube video id for add_youtube_video rule")
		return
	}

	iframe := config.YouTubeEmbedUrlOverride() + yt.VideoId
	log.Debug("render youtube.html for add_youtube_video rule",
		slog.Bool("description", yt.Description != ""),
		slog.String("iframe", iframe))

	entry.Content = self.render("youtube.html", map[string]any{
		"Entry":       entry,
		"IframeSrc":   iframe,
		"Width":       yt.Width,
		"Height":      yt.Height,
		"Description": template.HTML(yt.Description),
		"Thumbnail":   &yt.Thumbnail,
	})
	self.sanitized = true
}

type youtubeContent struct {
	Width, Height int
	VideoId       string
	Description   string
	Thumbnail     struct {
		Width, Height int
		URL           string
	}
}

func youtubeVideo(entry *model.Entry) (c youtubeContent) {
	atom := entry.Atom()
	if atom == nil || atom.Youtube == nil {
		return c
	}
	c.VideoId = atom.Youtube.VideoId

	media := atom.Media
	if media == nil || len(media.Groups) == 0 {
		return c
	}

	mg := &media.Groups[0]
	if len(mg.Contents) != 0 {
		c.Width = mg.Contents[0].Width
		c.Height = mg.Contents[0].Height
	}

	if len(mg.ThumbnailsEx) != 0 {
		c.Thumbnail.URL = mg.ThumbnailsEx[0].URL
		c.Thumbnail.Width = mg.ThumbnailsEx[0].Width
		c.Thumbnail.Height = mg.ThumbnailsEx[0].Height
	}

	if len(mg.Descriptions) != 0 {
		c.Description = sanitizer.StripTags(mg.Descriptions[0].Text)
	}
	return c
}

func (self *ContentRewrite) render(name string, data map[string]any) string {
	data["language"] = self.user.Language
	return string(self.templates.Render(name, data))
}
