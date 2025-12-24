// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mediaproxy // import "miniflux.app/v2/internal/mediaproxy"

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/dsh2dsh/bluemonday/v2"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/mux"
	"miniflux.app/v2/internal/urllib"
)

type urlProxyRewriter func(router *mux.ServeMux, url string) string

func RewriteDocumentWithRelativeProxyURL(m *mux.ServeMux, htmlDocument string,
) string {
	return genericProxyRewriter(m, ProxifyRelativeURL, htmlDocument)
}

func RewriteDocumentWithAbsoluteProxyURL(m *mux.ServeMux, htmlDocument string,
) string {
	return genericProxyRewriter(m, ProxifyAbsoluteURL, htmlDocument)
}

func genericProxyRewriter(m *mux.ServeMux, proxifyURL urlProxyRewriter,
	htmlDocument string,
) string {
	proxyMode := config.MediaProxyMode()
	if proxyMode == "none" {
		return htmlDocument
	}

	var mediaTypes struct {
		audio, image, video bool
	}
	for _, mediaType := range config.MediaProxyResourceTypes() {
		switch mediaType {
		case "audio":
			mediaTypes.audio = true
		case "image":
			mediaTypes.image = true
		case "video":
			mediaTypes.video = true
		}
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlDocument))
	if err != nil {
		return htmlDocument
	}

	var modified bool

	if mediaTypes.audio {
		doc.Find("audio, audio source").Each(func(i int, audio *goquery.Selection) {
			if srcAttrValue, ok := audio.Attr("src"); ok {
				if shouldProxifyURL(srcAttrValue, proxyMode) {
					audio.SetAttr("src", proxifyURL(m, srcAttrValue))
					modified = true
				}
			}
		})
	}

	if mediaTypes.image {
		doc.Find("img, picture source").Each(func(i int, img *goquery.Selection) {
			if srcAttrValue, ok := img.Attr("src"); ok {
				if shouldProxifyURL(srcAttrValue, proxyMode) {
					img.SetAttr("src", proxifyURL(m, srcAttrValue))
					modified = true
				}
			}

			if srcsetAttrValue, ok := img.Attr("srcset"); ok {
				proxifySourceSet(img, m, proxifyURL, proxyMode, srcsetAttrValue)
				modified = true
			}
		})

		if !mediaTypes.video {
			doc.Find("video").Each(func(i int, video *goquery.Selection) {
				if posterAttrValue, ok := video.Attr("poster"); ok {
					if shouldProxifyURL(posterAttrValue, proxyMode) {
						video.SetAttr("poster", proxifyURL(m, posterAttrValue))
						modified = true
					}
				}
			})
		}
	}

	if mediaTypes.video {
		doc.Find("video, video source").Each(func(i int, video *goquery.Selection) {
			if srcAttrValue, ok := video.Attr("src"); ok {
				if shouldProxifyURL(srcAttrValue, proxyMode) {
					video.SetAttr("src", proxifyURL(m, srcAttrValue))
					modified = true
				}
			}

			if posterAttrValue, ok := video.Attr("poster"); ok {
				if shouldProxifyURL(posterAttrValue, proxyMode) {
					video.SetAttr("poster", proxifyURL(m, posterAttrValue))
					modified = true
				}
			}
		})
	}

	if !modified {
		return htmlDocument
	}

	body := doc.FindMatcher(goquery.Single("body"))
	if body.Length() == 0 {
		body = doc.Selection
	}

	output, err := body.Html()
	if err != nil {
		return htmlDocument
	}
	return output
}

func proxifySourceSet(element *goquery.Selection, router *mux.ServeMux,
	proxifyFunction urlProxyRewriter, proxyOption, srcset string,
) {
	images := bluemonday.ParseSrcSetAttribute(srcset)
	for i := range images {
		image := &images[i]
		if shouldProxifyURL(image.ImageURL, proxyOption) {
			image.ImageURL = proxifyFunction(router, image.ImageURL)
		}
	}
	element.SetAttr("srcset", images.String())
}

// shouldProxifyURL checks if the media URL should be proxified based on the media proxy option and URL scheme.
func shouldProxifyURL(mediaURL, mediaProxyOption string) bool {
	switch {
	case mediaURL == "":
		return false
	case strings.HasPrefix(mediaURL, "data:"):
		return false
	case mediaProxyOption == "all":
		return true
	case mediaProxyOption != "none" && !urllib.IsHTTPS(mediaURL):
		return true
	default:
		return false
	}
}

// ShouldProxifyURLWithMimeType checks if the media URL should be proxified based on the media proxy option, URL scheme, and MIME type.
func ShouldProxifyURLWithMimeType(mediaURL, mediaMimeType, mediaProxyOption string, mediaProxyResourceTypes []string) bool {
	if !shouldProxifyURL(mediaURL, mediaProxyOption) {
		return false
	}

	for _, mediaType := range mediaProxyResourceTypes {
		if strings.HasPrefix(mediaMimeType, mediaType+"/") {
			return true
		}
	}

	return false
}
