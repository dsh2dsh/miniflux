// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package icon // import "miniflux.app/v2/internal/reader/icon"

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/image/draw"

	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/encoding"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/urllib"
)

type IconFinder struct {
	requestBuilder *fetcher.RequestBuilder
	websiteURL     string
	feedIconURL    string
	preferSiteIcon bool
}

func NewIconFinder(requestBuilder *fetcher.RequestBuilder, websiteURL,
	feedIconURL string, preferSiteIcon bool,
) *IconFinder {
	return &IconFinder{
		requestBuilder: requestBuilder,
		websiteURL:     websiteURL,
		feedIconURL:    feedIconURL,
		preferSiteIcon: preferSiteIcon,
	}
}

func (f *IconFinder) FindIcon() (*model.Icon, error) {
	slog.Debug("Begin icon discovery process",
		slog.String("website_url", f.websiteURL),
		slog.String("feed_icon_url", f.feedIconURL),
		slog.Bool("prefer_site_icon", f.preferSiteIcon))

	fetchFuncs := make([]func() (*model.Icon, error), 0, 2)
	if f.preferSiteIcon {
		fetchFuncs = append(fetchFuncs, f.tryFetchSiteIcon, f.tryFetchFeedIcon)
	} else {
		fetchFuncs = append(fetchFuncs, f.tryFetchFeedIcon, f.tryFetchSiteIcon)
	}

	for _, fetchFn := range fetchFuncs {
		if icon, err := fetchFn(); err == nil && icon != nil {
			return icon, nil
		}
	}
	return f.FetchDefaultIcon()
}

func (f *IconFinder) tryFetchFeedIcon() (*model.Icon, error) {
	if f.feedIconURL == "" {
		return nil, nil
	}

	icon, err := f.FetchFeedIcon()
	if err != nil {
		slog.Debug("Unable to download icon from feed",
			slog.String("website_url", f.websiteURL),
			slog.String("feed_icon_url", f.feedIconURL),
			slog.Any("error", err))
	}
	return icon, err
}

func (f *IconFinder) tryFetchSiteIcon() (*model.Icon, error) {
	icon, err := f.FetchIconsFromHTMLDocument()
	if err != nil {
		slog.Debug("Unable to fetch icons from HTML document",
			slog.String("website_url", f.websiteURL),
			slog.Any("error", err))
	}
	return icon, err
}

func (f *IconFinder) FetchDefaultIcon() (*model.Icon, error) {
	slog.Debug("Fetching default icon",
		slog.String("website_url", f.websiteURL),
	)

	iconURL, err := urllib.JoinBaseURLAndPath(urllib.RootURL(f.websiteURL), "favicon.ico")
	if err != nil {
		return nil, fmt.Errorf(`icon: unable to join root URL and path: %w`, err)
	}

	icon, err := f.DownloadIcon(iconURL)
	if err != nil {
		return nil, err
	}

	return icon, nil
}

func (f *IconFinder) FetchFeedIcon() (*model.Icon, error) {
	slog.Debug("Fetching feed icon",
		slog.String("website_url", f.websiteURL),
		slog.String("feed_icon_url", f.feedIconURL),
	)

	iconURL, err := urllib.AbsoluteURL(f.websiteURL, f.feedIconURL)
	if err != nil {
		return nil, fmt.Errorf(`icon: unable to convert icon URL to absolute URL: %w`, err)
	}

	return f.DownloadIcon(iconURL)
}

func (f *IconFinder) FetchIconsFromHTMLDocument() (*model.Icon, error) {
	slog.Debug("Searching icons from HTML document",
		slog.String("website_url", f.websiteURL))

	rootURL := urllib.RootURL(f.websiteURL)
	responseHandler, err := f.requestBuilder.Request(rootURL)
	if err != nil {
		return nil, fmt.Errorf("reader/icon: download website index page: %w", err)
	}
	defer responseHandler.Close()

	localizedError := responseHandler.LocalizedError()
	if localizedError != nil {
		return nil, fmt.Errorf("icon: unable to download website index page: %w",
			localizedError)
	}

	iconURLs, err := findIconURLsFromHTMLDocument(responseHandler.Body(),
		responseHandler.ContentType())
	if err != nil {
		return nil, err
	}

	rootURL = responseHandler.EffectiveURL()
	slog.Debug("Searched icon from HTML document",
		slog.String("website_url", f.websiteURL),
		slog.String("effective_url", rootURL),
		slog.String("icon_urls", strings.Join(iconURLs, ",")))

	for _, iconURL := range iconURLs {
		if strings.HasPrefix(iconURL, "data:") {
			slog.Debug("Found icon with data URL",
				slog.String("website_url", f.websiteURL),
			)
			return parseImageDataURL(iconURL)
		}

		iconURL, err = urllib.AbsoluteURL(rootURL, iconURL)
		if err != nil {
			return nil, fmt.Errorf(
				`icon: unable to convert icon URL to absolute URL: %w`, err)
		}

		if icon, err := f.DownloadIcon(iconURL); err != nil {
			slog.Debug("Unable to download icon from HTML document",
				slog.String("website_url", f.websiteURL),
				slog.String("icon_url", iconURL),
				slog.Any("error", err),
			)
		} else if icon != nil {
			slog.Debug("Found icon from HTML document",
				slog.String("website_url", f.websiteURL),
				slog.String("icon_url", iconURL),
			)
			return icon, nil
		}
	}
	return nil, nil
}

func (f *IconFinder) DownloadIcon(iconURL string) (*model.Icon, error) {
	slog.Debug("Downloading icon",
		slog.String("website_url", f.websiteURL),
		slog.String("icon_url", iconURL),
	)

	responseHandler, err := f.requestBuilder.Request(iconURL)
	if err != nil {
		return nil, fmt.Errorf("reader/icon: download website icon: %w", err)
	}
	defer responseHandler.Close()

	if localizedError := responseHandler.LocalizedError(); localizedError != nil {
		return nil, fmt.Errorf("icon: unable to download website icon: %w", localizedError)
	}

	responseBody, localizedError := responseHandler.ReadBody()
	if localizedError != nil {
		return nil, fmt.Errorf("icon: unable to read response body: %w", localizedError)
	}

	icon := &model.Icon{
		Hash:     crypto.HashFromBytes(responseBody),
		MimeType: responseHandler.ContentType(),
		Content:  responseBody,
	}

	const tooBig = 64 << 10
	if len(icon.Content) < tooBig {
		slog.Debug("icon don't need to be rescaled",
			slog.Int("size", len(icon.Content)), slog.Int("limit", tooBig))
		return icon, nil
	}
	return resizeIcon(icon), nil
}

func resizeIcon(icon *model.Icon) *model.Icon {
	if !slices.Contains([]string{"image/jpeg", "image/png", "image/gif"}, icon.MimeType) {
		slog.Info("icon isn't a png/gif/jpeg/ico, can't resize", slog.String("mimetype", icon.MimeType))
		return icon
	}

	// Don't resize icons that we can't decode, or that already have the right size.
	r := bytes.NewReader(icon.Content)
	config, _, err := image.DecodeConfig(r)
	if err != nil {
		slog.Warn("unable to decode the metadata of the icon", slog.Any("error", err))
		return icon
	}
	if config.Height <= 32 && config.Width <= 32 {
		slog.Debug("icon don't need to be rescaled", slog.Int("height", config.Height), slog.Int("width", config.Width))
		return icon
	}

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		slog.Error("reader/icon: failed seek to start", slog.Any("error", err))
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
		slog.Warn("unable to decode the icon", slog.Any("error", err))
		return icon
	}

	dst := image.NewRGBA(image.Rect(0, 0, 32, 32))
	draw.BiLinear.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)

	var b bytes.Buffer
	if err = png.Encode(io.Writer(&b), dst); err != nil {
		slog.Warn("unable to encode the new icon", slog.Any("error", err))
	}

	icon.Content = b.Bytes()
	icon.MimeType = "image/png"
	return icon
}

func findIconURLsFromHTMLDocument(body io.Reader, contentType string,
) ([]string, error) {
	htmlDocumentReader, err := encoding.NewCharsetReader(body, contentType)
	if err != nil {
		return nil, fmt.Errorf("icon: unable to create charset reader: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(htmlDocumentReader)
	if err != nil {
		return nil, fmt.Errorf("icon: unable to read document: %w", err)
	}

	queries := []string{
		"link[rel~='icon' i][href]",
		"link[rel='apple-touch-icon' i][href]",
		"link[rel='apple-touch-icon-precomposed.png'][href]",
	}

	iconURLs := []string{}
	for _, query := range queries {
		slog.Debug("Searching icon URL in HTML document",
			slog.String("query", query))
		for _, s := range doc.Find("head").First().Find(query).EachIter() {
			href, exists := s.Attr("href")
			if !exists {
				continue
			}
			if iconURL := strings.TrimSpace(href); iconURL != "" {
				iconURLs = append(iconURLs, iconURL)
				slog.Debug("Found icon URL in HTML document",
					slog.String("query", query),
					slog.String("icon_url", iconURL))
			}
		}
	}
	return iconURLs, nil
}

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/Data_URIs#syntax
// data:[<mediatype>][;encoding],<data>
// we consider <mediatype> to be mandatory, and it has to start with `image/`.
// we consider `base64`, `utf8` and the empty string to be the only valid encodings
func parseImageDataURL(value string) (*model.Icon, error) {
	re := regexp.MustCompile(`^data:` +
		`(?P<mediatype>image/[^;,]+)` +
		`(?:;(?P<encoding>base64|utf8))?` +
		`,(?P<data>.+)$`)

	matches := re.FindStringSubmatch(value)
	if matches == nil {
		return nil, fmt.Errorf(`icon: invalid data URL %q`, value)
	}

	mediaType := matches[re.SubexpIndex("mediatype")]
	encoding := matches[re.SubexpIndex("encoding")]
	data := matches[re.SubexpIndex("data")]

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
