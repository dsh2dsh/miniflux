// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package icon // import "miniflux.app/v2/internal/reader/icon"

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"iter"
	"log/slog"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/image/draw"

	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/encoding"
	"miniflux.app/v2/internal/reader/fetcher"
)

var (
	faviconURL *url.URL
	rootURL    *url.URL

	dataRe = regexp.MustCompile(`^data:` +
		`(?P<mediatype>image/[^;,]+)` +
		`(?:;(?P<encoding>base64|utf8))?` +
		`,(?P<data>.+)$`)
)

func init() {
	u, err := url.Parse("/")
	if err != nil {
		panic(fmt.Errorf("reader/icon: unable to parse site root: %w", err))
	}
	rootURL = u

	u, err = url.Parse("/favicon.ico")
	if err != nil {
		panic(fmt.Errorf("reader/icon: unable to parse favicon.ico: %w", err))
	}
	faviconURL = u
}

type IconFinder struct {
	requestBuilder *fetcher.RequestBuilder
	websiteURL     string
	feedIconURL    string
	preferSiteIcon bool

	site, feedIcon *url.URL
}

func NewIconFinder(requestBuilder *fetcher.RequestBuilder, websiteURL,
	feedIconURL string, preferSiteIcon bool,
) (*IconFinder, error) {
	self := &IconFinder{
		requestBuilder: requestBuilder,
		websiteURL:     websiteURL,
		feedIconURL:    feedIconURL,
		preferSiteIcon: preferSiteIcon,
	}
	return self.init()
}

func (self *IconFinder) init() (*IconFinder, error) {
	site, err := url.Parse(self.websiteURL)
	if err != nil {
		return nil, fmt.Errorf("reader/icon: unable parser website url: %w", err)
	}
	self.site = site

	if self.feedIconURL != "" {
		feedIcon, err := url.Parse(self.feedIconURL)
		if err != nil {
			return nil, fmt.Errorf("reader/icon: unable parser feed icon url: %w", err)
		}
		self.feedIcon = feedIcon
	}
	return self, nil
}

func (self *IconFinder) FindIcon(ctx context.Context) (*model.Icon, error) {
	logging.FromContext(ctx).Debug("Begin icon discovery process",
		slog.String("website_url", self.websiteURL),
		slog.String("feed_icon_url", self.feedIconURL),
		slog.Bool("prefer_site_icon", self.preferSiteIcon))

	fetchFuncs := make([]func(context.Context) (*model.Icon, error), 0, 2)
	if self.preferSiteIcon {
		fetchFuncs = append(fetchFuncs, self.tryFetchSiteIcon,
			self.tryFetchFeedIcon)
	} else {
		fetchFuncs = append(fetchFuncs, self.tryFetchFeedIcon,
			self.tryFetchSiteIcon)
	}

	for _, fetchFn := range fetchFuncs {
		if icon, err := fetchFn(ctx); err == nil && icon != nil {
			return icon, nil
		}
	}
	return self.fetchDefaultIcon(ctx)
}

func (self *IconFinder) tryFetchFeedIcon(ctx context.Context) (*model.Icon,
	error,
) {
	if self.feedIcon == nil {
		return nil, nil
	}

	log := logging.FromContext(ctx).With(
		slog.String("website_url", self.websiteURL),
		slog.String("feed_icon_url", self.feedIconURL))
	log.Debug("Fetching feed icon")

	icon, err := self.downloadIcon(ctx,
		self.site.ResolveReference(self.feedIcon).String())
	if err != nil {
		log.Debug("Unable to download icon from feed",
			slog.Any("error", err))
	}
	return icon, err
}

func (self *IconFinder) downloadIcon(ctx context.Context, iconURL string,
) (*model.Icon, error) {
	log := logging.FromContext(ctx)
	log.Debug("Downloading icon",
		slog.String("website_url", self.websiteURL),
		slog.String("icon_url", iconURL))

	resp, err := self.requestBuilder.Request(iconURL)
	if err != nil {
		return nil, fmt.Errorf("reader/icon: download icon %q: %w", iconURL, err)
	}
	defer resp.Close()

	if lerr := resp.LocalizedError(); lerr != nil {
		return nil, fmt.Errorf("reader/icon: unable to download icon %q: %w",
			iconURL, lerr)
	}

	body, lerr := resp.ReadBody()
	if lerr != nil {
		return nil, fmt.Errorf("reader/icon: unable to read response body: %w",
			lerr)
	}

	icon := &model.Icon{
		Hash:     crypto.HashFromBytes(body),
		MimeType: resp.ContentType(),
		Content:  body,
	}

	const tooBig = 64 << 10 // 64k
	if len(icon.Content) < tooBig {
		log.Debug("icon don't need to be rescaled",
			slog.Int("size", len(icon.Content)), slog.Int("limit", tooBig))
		return icon, nil
	}
	return resizeIcon(ctx, icon), nil
}

func resizeIcon(ctx context.Context, icon *model.Icon) *model.Icon {
	knownTypes := [...]string{"image/jpeg", "image/png", "image/gif"}
	log := logging.FromContext(ctx)

	if !slices.Contains(knownTypes[:], icon.MimeType) {
		log.Debug("Icon resize skipped: unsupported MIME type",
			slog.String("mime_type", icon.MimeType))
		return icon
	}

	// Don't resize icons that we can't decode, or that already have the right
	// size.
	r := bytes.NewReader(icon.Content)
	config, _, err := image.DecodeConfig(r)
	if err != nil {
		log.Warn("Unable to decode icon metadata", slog.Any("error", err))
		return icon
	}

	if config.Height <= 32 && config.Width <= 32 {
		log.Debug("icon don't need to be rescaled",
			slog.Int("height", config.Height),
			slog.Int("width", config.Width))
		return icon
	}

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		log.Error("reader/icon: failed seek to start", slog.Any("error", err))
		return icon
	}

	var src image.Image
	switch icon.MimeType {
	case "image/jpeg":
		src, err = jpeg.Decode(r)
	case "image/png":
		src, err = png.Decode(r)
	case "image/gif":
		src, err = gif.Decode(r)
	}
	if err != nil || src == nil {
		log.Warn("Unable to decode icon image", slog.Any("error", err))
		return icon
	}

	dst := image.NewRGBA(image.Rect(0, 0, 32, 32))
	draw.BiLinear.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)

	var b bytes.Buffer
	if err = png.Encode(io.Writer(&b), dst); err != nil {
		log.Warn("Unable to encode resized icon", slog.Any("error", err))
		return icon
	}

	icon.Content = b.Bytes()
	icon.MimeType = "image/png"
	return icon
}

func (self *IconFinder) tryFetchSiteIcon(ctx context.Context) (*model.Icon,
	error,
) {
	// Try the website URL first, then fall back to the root URL if no icon is
	// found. The website URL may include a subdirectory (e.g.,
	// https://example.org/subfolder/), and icons can be referenced relative to
	// that path.
	log := logging.FromContext(ctx)
	for u := range self.siteURLs() {
		icon, err := self.fetchIconsFromHTMLDocument(ctx, u)
		if err != nil {
			log.Debug("Unable to fetch icons from HTML document",
				slog.String("document_url", u.String()),
				slog.Any("error", err))
		} else if icon != nil {
			return icon, nil
		}
	}
	return nil, nil
}

func (self *IconFinder) siteURLs() iter.Seq[*url.URL] {
	return func(yield func(*url.URL) bool) {
		u := *self.site
		if !yield(&u) || u.EscapedPath() == "" {
			return
		}
		yield(self.site.ResolveReference(rootURL))
	}
}

func (self *IconFinder) fetchIconsFromHTMLDocument(ctx context.Context,
	u *url.URL,
) (*model.Icon, error) {
	documentURL := u.String()
	log := logging.FromContext(ctx).With(
		slog.String("document_url", documentURL))
	log.Debug("Searching icons from HTML document")

	resp, err := self.requestBuilder.Request(documentURL)
	if err != nil {
		return nil, fmt.Errorf("reader/icon: download website page %q: %w",
			documentURL, err)
	}
	defer resp.Close()

	if lerr := resp.LocalizedError(); lerr != nil {
		return nil, fmt.Errorf("icon: unable to download website page %q: %w",
			documentURL, lerr)
	}

	iconURLs, err := findIconURLsFromHTMLDocument(ctx, u, resp.Body(),
		resp.ContentType())
	if err != nil {
		return nil, err
	}

	log.Debug("Searched icon from HTML document",
		slog.String("icon_urls", strings.Join(iconURLs, ",")))

	for _, iconURL := range iconURLs {
		if strings.HasPrefix(iconURL, "data:") {
			log.Debug("Found icon with data URL")
			return parseImageDataURL(iconURL)
		}

		if icon, err := self.downloadIcon(ctx, iconURL); err != nil {
			log.Debug("Unable to download icon from HTML document",
				slog.String("icon_url", iconURL),
				slog.Any("error", err))
		} else if icon != nil {
			log.Debug("Downloaded icon from HTML document",
				slog.String("icon_url", iconURL))
			return icon, nil
		}
	}
	return nil, nil
}

func findIconURLsFromHTMLDocument(ctx context.Context, u *url.URL,
	body io.Reader, contentType string,
) ([]string, error) {
	r, err := encoding.NewCharsetReader(body, contentType)
	if err != nil {
		return nil, fmt.Errorf(
			"reader/icon: unable to create charset reader: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("reader/icon: unable to read document: %w", err)
	}

	queries := [...]string{
		"link[rel~='icon' i][href]",
		"link[rel='apple-touch-icon' i][href]",
	}

	documentURL := u.String()
	log := logging.FromContext(ctx).With(
		slog.String("document_url", documentURL))

	foundURLs := []string{}
	for _, query := range queries {
		log.Debug("Searching icon URL in HTML document",
			slog.String("query", query))
		for _, s := range doc.Find("head").First().Find(query).EachIter() {
			href, exists := s.Attr("href")
			if !exists {
				continue
			} else if href = strings.TrimSpace(href); href == "" {
				continue
			}

			parsedHref, err := url.Parse(href)
			if err != nil {
				log.Warn("Unable to convert icon URL to absolute URL",
					slog.String("href", href),
					slog.Any("error", err))
				continue
			}
			iconURL := u.ResolveReference(parsedHref).String()

			foundURLs = append(foundURLs, iconURL)
			log.Debug("Found icon URL in HTML document",
				slog.String("query", query),
				slog.String("href", href),
				slog.String("icon_url", iconURL))
		}
	}
	return foundURLs, nil
}

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/Data_URIs#syntax
// data:[<mediatype>][;encoding],<data>
// we consider <mediatype> to be mandatory, and it has to start with `image/`.
// we consider `base64`, `utf8` and the empty string to be the only valid encodings
func parseImageDataURL(value string) (*model.Icon, error) {
	matches := dataRe.FindStringSubmatch(value)
	if matches == nil {
		return nil, fmt.Errorf(`icon: invalid data URL %q`, value)
	}

	mediaType := matches[dataRe.SubexpIndex("mediatype")]
	encoding := matches[dataRe.SubexpIndex("encoding")]
	data := matches[dataRe.SubexpIndex("data")]

	var blob []byte
	switch encoding {
	case "base64":
		var err error
		blob, err = base64.StdEncoding.DecodeString(data)
		if err != nil {
			return nil, fmt.Errorf(`icon: invalid data %q (%w)`, value, err)
		}
	case "":
		decodedData, err := url.QueryUnescape(data)
		if err != nil {
			return nil, fmt.Errorf(`icon: unable to decode data URL %q`, value)
		}
		blob = []byte(decodedData)
	case "utf8":
		blob = []byte(data)
	}

	return &model.Icon{
		Hash:     crypto.HashFromBytes(blob),
		Content:  blob,
		MimeType: mediaType,
	}, nil
}

func (self *IconFinder) fetchDefaultIcon(ctx context.Context) (*model.Icon,
	error,
) {
	logging.FromContext(ctx).Debug("Fetching default icon",
		slog.String("website_url", self.websiteURL))
	icon, err := self.downloadIcon(ctx,
		self.site.ResolveReference(faviconURL).String())
	if err != nil {
		return nil, err
	}
	return icon, nil
}
