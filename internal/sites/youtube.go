package sites

import (
	"context"
	"errors"
	"fmt"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/sanitizer"
	"miniflux.app/v2/internal/template"
)

type Youtube struct {
	VideoId string `json:"videoId,omitempty"`
}

const (
	youtubeDomain        = "youtube.com"
	youtubeVideoMimeType = "application/x-shockwave-flash"
	youtubeTemplate      = "youtube.html"
)

func init() {
	addRewriterFunc(youtubeRewrite, youtubeDomain)
	addRenderFunc(youtubeRender, youtubeDomain)
}

func youtubeRewrite(ctx context.Context, entry *model.Entry) {
	atom := entry.Atom()
	if atom == nil || atom.Youtube == nil {
		return
	}
	entry.WithSiteData(&Youtube{VideoId: atom.Youtube.VideoId})

	media := atom.Media
	if media == nil || len(media.Groups) == 0 {
		return
	}
	mg := &media.Groups[0]

	if len(mg.Descriptions) != 0 {
		s := sanitizer.StripTags(mg.Descriptions[0].Text)
		entry.Content = `<pre>` + s + `</pre>`
	}
}

func youtubeRender(ctx context.Context, user *model.User, entry *model.Entry,
	t *template.Engine,
) ([]byte, error) {
	var yt Youtube
	if err := entry.DecodeSiteData(&yt); err != nil {
		return nil, fmt.Errorf("render Youtube entry: %w", err)
	} else if yt.VideoId == "" {
		return nil, errors.New("Youtube video id not found")
	}

	var w, h int
	for enc := range entry.Enclosures().WithMimeType(youtubeVideoMimeType) {
		w, h = enc.Width, enc.Height
		break
	}

	b := renderTemplate(t, youtubeTemplate, user, entry, map[string]any{
		"Description": template.HTML(entry.Content),
		"IframeSrc":   config.YouTubeEmbedUrlOverride() + yt.VideoId,
		"Width":       w,
		"Height":      h,
	})
	return b, nil
}
