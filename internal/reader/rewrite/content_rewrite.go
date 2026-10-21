// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rewrite

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strconv"
	"strings"
	"text/scanner"

	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/template"
)

type ContentRewrite struct {
	rules     []rule
	templates *template.Engine
	user      *model.User
	validate  bool
}

func NewContentRewrite(userRules string, u *model.User, t *template.Engine,
) (*ContentRewrite, error) {
	rules, err := parseRules(userRules)
	if err != nil {
		return nil, fmt.Errorf("rewrite: parse content rewrite rules: %w", err)
	}
	return &ContentRewrite{rules: rules, user: u, templates: t}, nil
}

func parseRules(s string) ([]rule, error) {
	if s = strings.TrimSpace(s); s == "" {
		return nil, nil
	}

	var err error
	scan := scanner.Scanner{
		Error: func(s *scanner.Scanner, msg string) {
			err = fmt.Errorf("scan token %s: %s", s.TokenText(), msg)
		},
		Mode: scanner.ScanIdents | scanner.ScanStrings,
	}
	scan.Init(strings.NewReader(s))

	var rules []rule
	for {
		tok := scan.Scan()
		if err != nil {
			return nil, err
		}

		switch tok {
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
			return rules, nil
		}
	}
}

func (self *ContentRewrite) WithValidate() *ContentRewrite {
	self.validate = true
	return self
}

func (self *ContentRewrite) Apply(ctx context.Context, entry *model.Entry,
) (err error) {
	rules := self.rules
	if len(rules) == 0 {
		rules = findDomainRule(entry.Hostname())
	}

	logging.FromContext(ctx).Debug("Applying content rewrite rules",
		slog.Any("rules", rules),
		slog.String("entry_url", entry.URL),
		slog.Bool("validate", self.validate))

	for _, r := range rules {
		err = self.applyRule(ctx, entry, r.WithValidate(self.validate))
		if err != nil {
			err = fmt.Errorf("rewrite: rewrite content rule failed: %s: %w",
				r.String(), err)
			break
		}
	}

	entry.Content = addPDFLink(entry.URL, entry.Content)
	return err
}

func (self *ContentRewrite) applyRule(ctx context.Context, entry *model.Entry,
	r *rule,
) (err error) {
	log := logging.FromContext(ctx).With(
		slog.Group("rule",
			slog.String("name", r.name),
			slog.Any("args", r.args),
			slog.Bool("validate", r.validate)),
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
		err = r.applyReplaceContent(entry)
	case "replace_title":
		err = r.applyReplaceTitle(entry)
	case "remove":
		err = r.applyRemove(entry)
	case "add_enclosure_links":
		entry.Content = addEnclosureLinks(entry)
	case "add_castopod_episode":
		entry.Content = addCastopodEpisode(entry.URL, entry.Content)
	case "base64_decode":
		err = r.applyBase64Decode(entry)
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
		entry.Content = html.UnescapeString(entry.Content)
	default:
		err = fmt.Errorf("unknown rule name: %s", r.name)
	}
	return err
}
