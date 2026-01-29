package sites

import (
	"github.com/dsh2dsh/gofeed/v2/atom"
	"github.com/dsh2dsh/gofeed/v2/ext"

	"miniflux.app/v2/internal/model"
)

func (self *SitesTestSuite) TestYoutube() {
	testEntry := model.Entry{
		URL:     "https://www.youtube.com/watch?v=1234",
		Content: "foo bar baz",
	}

	gotEntry := self.rewrite(withYoutubeAtom(&testEntry, "1234",
		"Video & Description"))
	self.Equal(`<pre>Video &amp; Description</pre>`, gotEntry.Content)

	var yt Youtube
	self.Require().NoError(gotEntry.DecodeSiteData(&yt))
	self.Equal(&Youtube{VideoId: "1234"}, &yt)

	self.withConfig(nil)
	s := self.render(gotEntry)
	self.T().Log("\n", s)

	self.Contains(s, `<div class="youtube">`)
	self.Contains(s, `<pre>Video &amp; Description</pre>`)
	self.Contains(s, `<details class="video" name="youtube-iframe">`)
	self.Contains(s, `<iframe`)
	self.Contains(s, `src="https://www.youtube-nocookie.com/embed/1234"`)
	self.Contains(s, `loading="lazy"`)
	self.Contains(s, `referrerpolicy="strict-origin-when-cross-origin"`)
	self.Contains(s, `credentialless`)

	self.withConfig(map[string]string{
		"YOUTUBE_EMBED_URL_OVERRIDE": "https://invidious.custom/embed/",
	})
	s = self.render(gotEntry)
	self.T().Log("\n", s)

	self.Contains(s, `src="https://invidious.custom/embed/1234"`)
}

func withYoutubeAtom(entry *model.Entry, videoId, descr string,
) *model.Entry {
	return entry.WithAtom(&atom.Entry{
		Youtube: &ext.Youtube{VideoId: videoId},
		Media: &ext.Media{
			Groups: []ext.MediaGroup{
				{
					Contents:     []ext.MediaContent{{Width: 640, Height: 390}},
					Descriptions: []ext.MediaDescription{{Text: descr}},
				},
			},
		},
	})
}
