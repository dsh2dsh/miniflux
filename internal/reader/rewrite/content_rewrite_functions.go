// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rewrite

import (
	"encoding/base64"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/model"

	nethtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/PuerkitoBio/goquery"
)

var (
	youtubeIdRegex = regexp.MustCompile(`youtube_id"?\s*[:=]\s*"([a-zA-Z0-9_-]{11})"`)
	textLinkRegex  = regexp.MustCompile(`(?mi)(\bhttps?:\/\/[-A-Z0-9+&@#\/%?=~_|!:,.;]*[-A-Z0-9+&@#\/%=~_|])`)
)

var (
	body = goquery.Single("body")

	figure = nethtml.Node{
		Type:     nethtml.ElementNode,
		Data:     "figure",
		DataAtom: atom.Figure,
	}

	imgNode = nethtml.Node{
		Type:     nethtml.ElementNode,
		Data:     "img",
		DataAtom: atom.Img,
	}

	linkNode = nethtml.Node{
		Type:     nethtml.ElementNode,
		Data:     "a",
		DataAtom: atom.A,
	}

	spaceNode = nethtml.Node{
		Type: nethtml.TextNode,
		Data: " ",
	}
)

// titlelize returns a copy of the string s with all Unicode letters that begin words
// mapped to their Unicode title case.
func titlelize(s string) string {
	// A closure is used here to remember the previous character
	// so that we can check if there is a space preceding the current
	// character.
	previous := ' '
	return strings.Map(
		func(current rune) rune {
			if unicode.IsSpace(previous) {
				previous = current
				return unicode.ToTitle(current)
			}
			previous = current
			return current
		}, strings.ToLower(s))
}

func addImageTitle(entryContent string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(entryContent))
	if err != nil {
		return entryContent
	}

	images := doc.Find("img[src][title]")
	if images.Length() == 0 {
		return entryContent
	}

	for _, img := range images.EachIter() {
		img.WrapAllNode(new(figure)).Parent().
			AppendHtml(`<figcaption><p></p></figcaption>`).
			Children().Last(). // figcaption
			Children().SetText(img.AttrOr("title", ""))
	}

	output, err := doc.FindMatcher(body).Html()
	if err != nil {
		return entryContent
	}
	return output
}

func addMailtoSubject(entryContent string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(entryContent))
	if err != nil {
		return entryContent
	}

	mailToLinks := doc.Find(`a[href^="mailto:"]`)
	if mailToLinks.Length() == 0 {
		return entryContent
	}

	for _, a := range mailToLinks.EachIter() {
		mailto, err := url.Parse(a.AttrOr("href", ""))
		if err != nil {
			continue
		}

		if subject := mailto.Query().Get("subject"); subject != "" {
			a.AppendNodes(new(spaceNode)).AppendHtml(`<span></span>`).
				Children().Last().SetText("[" + subject + "]")
		}
	}

	output, err := doc.FindMatcher(body).Html()
	if err != nil {
		return entryContent
	}
	return output
}

func addDynamicImage(entryContent string) string {
	doc, err := goquery.NewDocumentFromReaderWithOptions(
		strings.NewReader(entryContent),
		nethtml.ParseOptionEnableScripting(false))
	if err != nil {
		return entryContent
	}

	// Ordered most preferred to least preferred.
	candidateAttrs := [...]string{
		"data-src",
		"data-original",
		"data-orig",
		"data-url",
		"data-orig-file",
		"data-large-file",
		"data-medium-file",
		"data-original-mos",
		"data-2000src",
		"data-1000src",
		"data-800src",
		"data-655src",
		"data-500src",
		"data-380src",
	}
	candidateSrcsetAttrs := [...]string{"data-srcset"}

	var changed bool
	for _, img := range doc.Find("img,div").EachIter() {
		// Src-linked candidates
		for _, attr := range candidateAttrs {
			src, found := img.Attr(attr)
			if !found {
				continue
			}

			changed = true
			if img.Is("img") {
				img.SetAttr("src", src)
				break
			}

			s := (&goquery.Selection{Nodes: []*nethtml.Node{new(imgNode)}}).
				SetAttr("src", src)
			if alt, ok := img.Attr("alt"); ok {
				s.SetAttr("alt", alt)
			}

			img.ReplaceWithSelection(s)
			break
		}

		// Srcset-linked candidates
		for _, attr := range candidateSrcsetAttrs {
			src, found := img.Attr(attr)
			if !found {
				continue
			}

			changed = true
			if img.Is("img") {
				img.SetAttr("srcset", src)
				break
			}

			s := goquery.Selection{Nodes: []*nethtml.Node{new(imgNode)}}
			s.SetAttr("srcset", src)
			if alt, ok := img.Attr("alt"); ok {
				s.SetAttr("alt", alt)
			}

			img.ReplaceWithSelection(&s)
			break
		}
	}

	if !changed {
		for _, noscript := range doc.Find("noscript").EachIter() {
			if img := noscript.Find("img"); img.Length() == 1 {
				img.Unwrap()
				changed = true
			}
		}
	}

	if changed {
		output, err := doc.FindMatcher(body).Html()
		if err == nil {
			return output
		}
	}
	return entryContent
}

func addDynamicIframe(entryContent string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(entryContent))
	if err != nil {
		return entryContent
	}

	// Ordered most preferred to least preferred.
	candidateAttrs := []string{
		"data-src",
		"data-original",
		"data-orig",
		"data-url",
		"data-lazy-src",
	}

	var changed bool
	for _, iframe := range doc.Find("iframe").EachIter() {
		for _, attr := range candidateAttrs {
			if src, found := iframe.Attr(attr); found {
				changed = true
				iframe.SetAttr("src", src)
				break
			}
		}
	}

	if !changed {
		return entryContent
	}

	output, err := doc.FindMatcher(body).Html()
	if err != nil {
		return entryContent
	}
	return output
}

func fixMediumImages(entryContent string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(entryContent))
	if err != nil {
		return entryContent
	}

	var changed bool
	for _, fg := range doc.Find("figure.paragraph-image").EachIter() {
		noscript := fg.Find("noscript")
		if noscript.Length() != 0 {
			images, err := nethtml.ParseFragment(
				strings.NewReader(noscript.Text()), fg.Get(0).Parent)
			if err != nil {
				continue
			}
			fg.ReplaceWithNodes(images...)
			changed = true
		}
	}

	if !changed {
		return entryContent
	}

	output, err := doc.FindMatcher(body).Html()
	if err != nil {
		return entryContent
	}
	return output
}

func useNoScriptImages(entryContent string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(entryContent))
	if err != nil {
		return entryContent
	}

	for _, fg := range doc.Find("figure").EachIter() {
		img := fg.Find("img")
		if img.Length() == 0 {
			continue
		}

		noscript := fg.Find("noscript")
		if noscript.Length() == 0 {
			continue
		}

		images, err := nethtml.ParseFragment(
			strings.NewReader(noscript.Text()), fg.Get(0))
		if err != nil {
			continue
		}

		fg.PrependNodes(images...)
		img.Remove()
		noscript.Remove()
	}

	output, err := doc.FindMatcher(body).Html()
	if err != nil {
		return entryContent
	}
	return output
}

func getYoutubVideoIDFromURL(entryURL string) string {
	u, err := url.Parse(entryURL)
	if err != nil {
		return ""
	}

	hostname := u.Hostname()
	if hostname != "youtube.com" && !strings.HasSuffix(hostname, ".youtube.com") {
		return ""
	}

	if u.Path == "/watch" {
		if v := u.Query().Get("v"); v != "" {
			return v
		}
		return ""
	}

	if id, found := strings.CutPrefix(u.Path, "/shorts/"); found {
		if len(id) == 11 {
			// youtube shorts id are always 11 chars.
			return id
		}
	}

	return ""
}

func buildVideoPlayerIframe(absoluteVideoURL string) string {
	return `<iframe width="650" height="350" frameborder="0" src="` + absoluteVideoURL + `" allowfullscreen></iframe>`
}

func addVideoPlayerIframe(absoluteVideoURL, entryContent string) string {
	return buildVideoPlayerIframe(absoluteVideoURL) + `<br>` + entryContent
}

func addYoutubeVideoUsingInvidiousPlayer(entryURL, entryContent string) string {
	if videoURL := getYoutubVideoIDFromURL(entryURL); videoURL != "" {
		return addVideoPlayerIframe(`https://`+config.InvidiousInstance()+
			`/embed/`+videoURL, entryContent)
	}
	return entryContent
}

// For reference: https://github.com/miniflux/v2/pull/1314
func addYoutubeVideoFromId(entryContent string) string {
	matches := youtubeIdRegex.FindAllStringSubmatch(entryContent, -1)
	if matches == nil {
		return entryContent
	}

	var videoPlayerHTML strings.Builder
	for _, match := range matches {
		if len(match) == 2 {
			videoPlayerHTML.WriteString(buildVideoPlayerIframe(
				config.YouTubeEmbedUrlOverride() + match[1]))
			videoPlayerHTML.WriteString("<br>")
		}
	}

	videoPlayerHTML.WriteString(entryContent)
	return videoPlayerHTML.String()
}

func addInvidiousVideo(entryURL, entryContent string) string {
	u, err := url.Parse(entryURL)
	if err != nil {
		return entryContent
	}

	if u.Path != "/watch" {
		return entryContent
	}

	qs := u.Query()
	videoID := qs.Get("v")
	if videoID == "" {
		return entryContent
	}
	qs.Del("v")

	embedVideoURL := "https://" + u.Hostname() + `/embed/` + videoID
	if len(qs) > 0 {
		embedVideoURL += "?" + qs.Encode()
	}

	return addVideoPlayerIframe(embedVideoURL, entryContent)
}

func addPDFLink(entryURL, entryContent string) string {
	if strings.HasSuffix(entryURL, ".pdf") {
		return fmt.Sprintf(`<a href=%q>PDF</a><br>%s`, entryURL, entryContent)
	}
	return entryContent
}

func addEnclosureLinks(entry *model.Entry) string {
	var links strings.Builder
	for _, enclosure := range entry.Enclosures() {
		if enclosure.URL == "" {
			continue
		}
		enclosureURL := html.EscapeString(enclosure.URL)
		links.WriteString(`<li><a href="`)
		links.WriteString(enclosureURL)
		links.WriteString(`">`)
		links.WriteString(enclosureURL)
		links.WriteString(`</a></li>`)
	}

	if links.Len() > 0 {
		return entry.Content + "<hr/><ul>" + strings.TrimSpace(links.String()) + "</ul>"
	}
	return entry.Content
}

func replaceTextLinks(input string) string {
	return textLinkRegex.ReplaceAllString(input, `<a href="${1}">${1}</a>`)
}

func replaceCustom(entryContent, searchTerm, replaceTerm string,
) (string, error) {
	re, err := regexp.Compile(searchTerm)
	if err != nil {
		return "", fmt.Errorf("compile search term: %w", err)
	}
	return re.ReplaceAllString(entryContent, replaceTerm), nil
}

func removeCustom(entryContent, selector string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(entryContent))
	if err != nil {
		return "", fmt.Errorf("parse content into goquery doc: %w", err)
	}

	doc.Find(selector).Remove()
	output, _ := doc.FindMatcher(goquery.Single("body")).Html()
	return output, nil
}

func addCastopodEpisode(entryURL, entryContent string) string {
	player := `<iframe width="650" frameborder="0" src="` + entryURL + `/embed/light"></iframe>`

	return player + `<br>` + entryContent
}

// applyFuncOnTextContent replaces the text nodes under the selector with the
// plain text returned by decoder.
func applyFuncOnTextContent(entryContent, selector string,
	decoder func(string) (string, bool),
) string {
	var changed bool
	var fn func(i int, s *goquery.Selection)
	fn = func(i int, s *goquery.Selection) {
		if n := s.Nodes[0]; n.Type == nethtml.TextNode {
			if s, ok := decoder(n.Data); ok && s != n.Data {
				n.Data = html.EscapeString(s)
				changed = true
			}
			return
		}
		s.Contents().Each(fn)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(entryContent))
	if err != nil {
		return entryContent
	}

	doc.Find(selector).Each(fn)
	if !changed {
		return entryContent
	}

	output, err := doc.FindMatcher(body).Html()
	if err != nil {
		return entryContent
	}
	return output
}

func decodeBase64Content(entryContent string) (string, bool) {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(entryContent))
	if err != nil {
		return entryContent, false
	}
	return string(b), true
}

func removeTables(entryContent string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(entryContent))
	if err != nil {
		return entryContent
	}

	// Column definitions have no content to keep.
	doc.Find("col").Remove()

	// Replace each table element with its own content in place, so the
	// surrounding content keeps its original document order. Appending to the
	// parent would move the unwrapped content to the end of the parent instead.
	const selector = "table, caption, colgroup, thead, tbody, tfoot, tr, th, td"
	for _, s := range doc.Find(selector).EachIter() {
		s.ReplaceWithSelection(s.Contents())
	}

	output, err := doc.FindMatcher(body).Html()
	if err != nil {
		return entryContent
	}
	return output
}

func fixGhostCards(entryContent string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(entryContent))
	if err != nil {
		return entryContent
	}

	const cardSelector = "figure.kg-card"
	cards := doc.Find(cardSelector)
	if cards.Length() == 0 {
		return entryContent
	}

	var currentList *goquery.Selection
	for _, card := range cards.EachIter() {
		title := card.Find(".kg-bookmark-title").First().Text()
		author := card.Find(".kg-bookmark-author").First().Text()
		href := card.Find("a.kg-bookmark-container").First().AttrOr("href", "")

		// if there is no link or title, skip processing
		if href == "" || title == "" {
			continue
		}

		s := goquery.Selection{Nodes: []*nethtml.Node{new(linkNode)}}
		s.SetAttr("href", href)
		if author == "" || strings.HasSuffix(title, author) {
			s.SetText(title)
		} else {
			s.SetText(title + " - " + author)
		}

		next := card.Next()

		// if the next element is also a card, start a list
		if next.Is(cardSelector) && currentList == nil {
			currentList = card.BeforeHtml("<ul></ul>").Prev()
		}

		if currentList != nil {
			// add this card to the list, then delete it
			currentList.AppendHtml("<li></li>").
				Children().Last(). // <li>
				AppendSelection(&s)
			card.Remove()
		} else {
			// replace single card
			card.ReplaceWithSelection(&s)
		}

		// if the next element is not a card, start a new list
		if !next.Is(cardSelector) && currentList != nil {
			currentList = nil
		}
	}

	output, err := doc.FindMatcher(body).Html()
	if err != nil {
		return entryContent
	}
	return strings.TrimSpace(output)
}

func removeImgBlurParams(entryContent string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(entryContent))
	if err != nil {
		return entryContent
	}

	changed := false

	doc.Find("img[src]").Each(func(i int, img *goquery.Selection) {
		srcAttr, exists := img.Attr("src")
		if !exists {
			return
		}

		parsedURL, err := url.Parse(srcAttr)
		if err != nil {
			return
		}

		// Only strip query parameters if this is a blurry placeholder image
		if parsedURL.RawQuery != "" {
			// Check if there's a blur parameter with a non-zero value
			if blurValue := parsedURL.Query().Get("blur"); blurValue != "" {
				if blurInt, err := strconv.Atoi(blurValue); err == nil && blurInt > 0 {
					parsedURL.RawQuery = ""
					img.SetAttr("src", parsedURL.String())
					changed = true
				}
			}
		}
	})

	if changed {
		output, _ := doc.FindMatcher(goquery.Single("body")).Html()
		return output
	}

	return entryContent
}
