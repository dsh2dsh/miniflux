package sanitizer

import (
	_ "embed"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"miniflux.app/v2/internal/config"
)

//go:embed testdata/miniflux_github.html
var githubHTML string

//go:embed testdata/miniflux_wikipedia.html
var wikipediaHTML string

func BenchmarkSanitizeContent(b *testing.B) {
	inputs := map[string]string{
		"https://github.com/miniflux/v2":         githubHTML,
		"https://fr.wikipedia.org/wiki/Miniflux": wikipediaHTML,
	}

	tests := make(map[*url.URL]string, len(inputs))
	for rawurl, s := range inputs {
		u, err := url.Parse(rawurl)
		require.NoError(b, err)
		tests[u] = s
	}

	b.ReportAllocs()
	for b.Loop() {
		for u, s := range tests {
			SanitizeContent(s, u)
		}
	}
}

func TestStripTags(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "plain text",
			title: "Foo bar baz",
			want:  "Foo bar baz",
		},
		{
			name:  "with html",
			title: "Foo <string>bar</strong> baz",
			want:  "Foo bar baz",
		},
		{
			name:  "broken html",
			title: "Foo <string>bar baz",
			want:  "Foo bar baz",
		},
		{
			name:  "with spaces",
			title: " Foo bar <b>baz</b>",
			want:  "Foo bar baz",
		},
		{
			name:  "with br",
			title: "Foo\n<br>\nbar\n<br>\nbaz",
			want:  "Foo\n\nbar\n\nbaz",
		},
		{
			name:  "with entities",
			title: "&amp;Foo &lt; bar &gt; baz",
			want:  "&amp;Foo &lt; bar &gt; baz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, StripTags(tt.title))
		})
	}
}

func TestSanitizeContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid input",
			input:    `<p>This is a <strong>text</strong> with an image: <img src="http://example.org/" alt="Test" loading="lazy">.</p>`,
			expected: `<p>This is a <strong>text</strong> with an image: <img src="http://example.org/" alt="Test" loading="lazy"/>.</p>`,
		},
		{
			name:     "with html and body",
			input:    `<html><head></head><body><p>This is a <strong>text</strong> with an image: <img src="http://example.org/" alt="Test" loading="lazy">.</p></body></html>`,
			expected: `<p>This is a <strong>text</strong> with an image: <img src="http://example.org/" alt="Test" loading="lazy"/>.</p>`,
		},
		{
			name:     "incorrect width and height",
			input:    `<img src="https://example.org/image.png" width="10px" height="20px">`,
			expected: `<img src="https://example.org/image.png" loading="lazy"/>`,
		},
		{
			name:     "incorrect width",
			input:    `<img src="https://example.org/image.png" width="10px" height="20">`,
			expected: `<img src="https://example.org/image.png" height="20" loading="lazy"/>`,
		},
		{
			name:     "empty width and height",
			input:    `<img src="https://example.org/image.png" width="" height="">`,
			expected: `<img src="https://example.org/image.png" loading="lazy"/>`,
		},
		{
			name:     "incorrect height",
			input:    `<img src="https://example.org/image.png" width="10" height="20px">`,
			expected: `<img src="https://example.org/image.png" width="10" loading="lazy"/>`,
		},
		{
			name:     "negative width",
			input:    `<img src="https://example.org/image.png" width="-10" height="20">`,
			expected: `<img src="https://example.org/image.png" height="20" loading="lazy"/>`,
		},
		{
			name:     "negative height",
			input:    `<img src="https://example.org/image.png" width="10" height="-20">`,
			expected: `<img src="https://example.org/image.png" width="10" loading="lazy"/>`,
		},
		{
			name:  "img with text data url",
			input: `<img src="data:text/plain;base64,SGVsbG8sIFdvcmxkIQ==" alt="Example"/>`,
		},
		{
			name:     "img with data url",
			input:    `<img src="data:image/gif;base64,test" alt="Example">`,
			expected: `<img src="data:image/gif;base64,test" alt="Example" loading="lazy"/>`,
		},
		{
			name:     "srcset",
			input:    `<img srcset="example-320w.jpg, example-480w.jpg 1.5x, example-640w.jpg 2x, example-640w.jpg 640w" src="example-640w.jpg" alt="Example">`,
			expected: `<img srcset="https://example.org/example-320w.jpg, https://example.org/example-480w.jpg 1.5x, https://example.org/example-640w.jpg 2x, https://example.org/example-640w.jpg 640w" src="https://example.org/example-640w.jpg" alt="Example" loading="lazy"/>`,
		},
		{
			name:     "invalid srcset",
			input:    `<img srcset="://example.com/example-320w.jpg" src="example-640w.jpg" alt="Example">`,
			expected: `<img src="https://example.org/example-640w.jpg" alt="Example" loading="lazy"/>`,
		},
		{
			name:     "srcset and no src",
			input:    `<img srcset="example-320w.jpg, example-480w.jpg 1.5x,   example-640w.jpg 2x, example-640w.jpg 640w" alt="Example">`,
			expected: `<img srcset="https://example.org/example-320w.jpg, https://example.org/example-480w.jpg 1.5x, https://example.org/example-640w.jpg 2x, https://example.org/example-640w.jpg 640w" alt="Example" loading="lazy"/>`,
		},
		{
			name:     "fetchpriority high",
			input:    `<img src="https://example.org/image.png" fetchpriority="high">`,
			expected: `<img src="https://example.org/image.png" fetchpriority="high" loading="lazy"/>`,
		},
		{
			name:     "fetchpriority low",
			input:    `<img src="https://example.org/image.png" fetchpriority="low">`,
			expected: `<img src="https://example.org/image.png" fetchpriority="low" loading="lazy"/>`,
		},
		{
			name:     "invalid fetchpriority",
			input:    `<img src="https://example.org/image.png" fetchpriority="invalid">`,
			expected: `<img src="https://example.org/image.png" loading="lazy"/>`,
		},
		{
			name:     "non img with fetchpriority",
			input:    `<p fetchpriority="high">Text</p>`,
			expected: `<p>Text</p>`,
		},
		{
			name:     "decoding sync",
			input:    `<img src="https://example.org/image.png" decoding="sync">`,
			expected: `<img src="https://example.org/image.png" decoding="sync" loading="lazy"/>`,
		},
		{
			name:     "decoding async",
			input:    `<img src="https://example.org/image.png" decoding="async">`,
			expected: `<img src="https://example.org/image.png" decoding="async" loading="lazy"/>`,
		},
		{
			name:     "invalid decoding",
			input:    `<img src="https://example.org/image.png" decoding="invalid">`,
			expected: `<img src="https://example.org/image.png" loading="lazy"/>`,
		},
		{
			name:     "non img with decoding",
			input:    `<p decoding="async">Text</p>`,
			expected: `<p>Text</p>`,
		},
		{
			name:     "source with srcset and media",
			input:    `<picture><source media="(min-width: 800px)" srcset="elva-800w.jpg"></picture>`,
			expected: `<picture><source media="(min-width: 800px)" srcset="https://example.org/elva-800w.jpg"/></picture>`,
		},
		{
			name:     "medium img with srcset",
			input:    `<img alt="Image for post" class="t u v ef aj" src="https://miro.medium.com/max/5460/1*aJ9JibWDqO81qMfNtqgqrw.jpeg" srcset="https://miro.medium.com/max/552/1*aJ9JibWDqO81qMfNtqgqrw.jpeg 276w, https://miro.medium.com/max/1000/1*aJ9JibWDqO81qMfNtqgqrw.jpeg 500w" sizes="500px" width="2730" height="3407">`,
			expected: `<img alt="Image for post" src="https://miro.medium.com/max/5460/1*aJ9JibWDqO81qMfNtqgqrw.jpeg" srcset="https://miro.medium.com/max/552/1*aJ9JibWDqO81qMfNtqgqrw.jpeg 276w, https://miro.medium.com/max/1000/1*aJ9JibWDqO81qMfNtqgqrw.jpeg 500w" sizes="500px" width="2730" height="3407" loading="lazy"/>`,
		},
		{
			name:     "self closing tags",
			input:    `<p>This <br> is a <strong>text</strong> <br/>with an image: <img src="http://example.org/" alt="Test" loading="lazy"/>.</p>`,
			expected: `<p>This <br/> is a <strong>text</strong> <br/>with an image: <img src="http://example.org/" alt="Test" loading="lazy"/>.</p>`,
		},
		{
			name:     "table",
			input:    `<table><tr><th>A</th><th colspan="2">B</th></tr><tr><td>C</td><td>D</td><td>E</td></tr></table>`,
			expected: `<table><tr><th>A</th><th colspan="2">B</th></tr><tr><td>C</td><td>D</td><td>E</td></tr></table>`,
		},
		{
			name:     "relative URL",
			input:    `This <a href="/test.html">link is relative</a> and this image: <img src="../folder/image.png"/>`,
			expected: `This <a href="https://example.org/test.html" target="_blank" rel="nofollow noreferrer noopener">link is relative</a> and this image: <img src="https://example.org/folder/image.png" loading="lazy"/>`,
		},
		{
			name:     "protocol relative url",
			input:    `This <a href="//static.example.org/index.html">link is relative</a>.`,
			expected: `This <a href="https://static.example.org/index.html" target="_blank" rel="nofollow noreferrer noopener">link is relative</a>.`,
		},
		{
			name:     "invalid tag",
			input:    `<p>My invalid <z>tag</z>.</p>`,
			expected: `<p>My invalid tag.</p>`,
		},
		{
			name:     "video tag",
			input:    `<p>My valid <video src="videofile.webm" autoplay poster="posterimage.jpg">fallback</video>.</p>`,
			expected: `<p>My valid <video src="https://example.org/videofile.webm" poster="https://example.org/posterimage.jpg" controls="controls">fallback</video>.</p>`,
		},
		{
			name:     "audio and source",
			input:    `<p>My music <audio controls="controls"><source src="foo.wav" type="audio/wav"></audio>.</p>`,
			expected: `<p>My music <audio controls="controls"><source src="https://example.org/foo.wav" type="audio/wav"/></audio>.</p>`,
		},
		{
			name:     "unknown tag",
			input:    `<p>My invalid <unknown>tag</unknown>.</p>`,
			expected: `<p>My invalid tag.</p>`,
		},
		{
			name:     "invalid nested tag",
			input:    `<p>My invalid <z>tag with some <em>valid</em> tag</z>.</p>`,
			expected: `<p>My invalid tag with some <em>valid</em> tag.</p>`,
		},
		{
			name:  "invalid iframe",
			input: `<iframe src="https://example.org/"></iframe>`,
		},
		{
			name:  "same domain iframe",
			input: `<iframe src="https://example.com/test"></iframe>`,
		},
		{
			name:     "invidious iframe",
			input:    `<iframe src="https://yewtu.be/watch?v=video_id"></iframe>`,
			expected: `<iframe src="https://yewtu.be/watch?v=video_id" loading="lazy" sandbox="allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox" credentialless=""></iframe>`,
		},
		{
			name:     "custom youtube embed url",
			input:    `<iframe src="https://www.invidious.custom/embed/1234"></iframe>`,
			expected: `<iframe src="https://www.invidious.custom/embed/1234" loading="lazy" sandbox="allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox" credentialless=""></iframe>`,
		},
		{
			name:     "iframe with child elements",
			input:    `<iframe src="https://www.youtube.com/"><p>test</p></iframe>`,
			expected: `<iframe src="https://www.youtube.com/" loading="lazy" sandbox="allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox" credentialless="" referrerpolicy="strict-origin-when-cross-origin"></iframe>`,
		},
		{
			name:     "iframe with referrer policy",
			input:    `<iframe src="https://www.youtube.com/embed/test123" referrerpolicy="strict-origin-when-cross-origin"></iframe>`,
			expected: `<iframe src="https://www.invidious.custom/embed/test123" loading="lazy" sandbox="allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox" credentialless=""></iframe>`,
		},
		{
			name:     "link with target",
			input:    `<p>This link is <a href="http://example.org/index.html">an anchor</a></p>`,
			expected: `<p>This link is <a href="http://example.org/index.html" target="_blank" rel="nofollow noreferrer noopener">an anchor</a></p>`,
		},
		{
			name:     "anchor link",
			input:    `<p>This link is <a href="#some-anchor">an anchor</a></p>`,
			expected: `<p>This link is <a href="https://example.org/foo.html#some-anchor" target="_blank" rel="nofollow noreferrer noopener">an anchor</a></p>`,
		},
		{
			name:     "invalid URL scheme",
			input:    `<p>This link is <a src="file:///etc/passwd">not valid</a></p>`,
			expected: `<p>This link is not valid</p>`,
		},
		{
			name:     "apt scheme",
			input:    `<p>This link is <a href="apt:some-package?channel=test">valid</a></p>`,
			expected: `<p>This link is <a href="apt:some-package?channel=test" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "bitcoin scheme",
			input:    `<p>This link is <a href="bitcoin:175tWpb8K1S7NmH4Zx6rewF9WQrcZv245W">valid</a></p>`,
			expected: `<p>This link is <a href="bitcoin:175tWpb8K1S7NmH4Zx6rewF9WQrcZv245W" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "callto scheme",
			input:    `<p>This link is <a href="callto:12345679">valid</a></p>`,
			expected: `<p>This link is <a href="callto:12345679" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "feed scheme",
			input:    `<p>This link is <a href="feed://example.com/rss.xml">valid</a></p>`,
			expected: `<p>This link is <a href="feed://example.com/rss.xml" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "feed scheme 2",
			input:    `<p>This link is <a href="feed:https://example.com/rss.xml">valid</a></p>`,
			expected: `<p>This link is <a href="feed:https://example.com/rss.xml" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "geo scheme",
			input:    `<p>This link is <a href="geo:13.4125,103.8667">valid</a></p>`,
			expected: `<p>This link is <a href="geo:13.4125,103.8667" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "itms scheme",
			input:    `<p>This link is <a href="itms://itunes.com/apps/my-app-name">valid</a></p>`,
			expected: `<p>This link is <a href="itms://itunes.com/apps/my-app-name" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "itms-apps scheme",
			input:    `<p>This link is <a href="itms-apps://itunes.com/apps/my-app-name">valid</a></p>`,
			expected: `<p>This link is <a href="itms-apps://itunes.com/apps/my-app-name" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "magnet scheme",
			input:    `<p>This link is <a href="magnet:?xt.1=urn:sha1:YNCKHTQCWBTRNJIV4WNAE52SJUQCZO5C&amp;xt.2=urn:sha1:TXGCZQTH26NL6OUQAJJPFALHG2LTGBC7">valid</a></p>`,
			expected: `<p>This link is <a href="magnet:?xt.1=urn:sha1:YNCKHTQCWBTRNJIV4WNAE52SJUQCZO5C&amp;xt.2=urn:sha1:TXGCZQTH26NL6OUQAJJPFALHG2LTGBC7" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "mailto scheme",
			input:    `<p>This link is <a href="mailto:jsmith@example.com?subject=A%20Test&amp;body=My%20idea%20is%3A%20%0A">valid</a></p>`,
			expected: `<p>This link is <a href="mailto:jsmith@example.com?subject=A%20Test&amp;body=My%20idea%20is%3A%20%0A" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "news scheme",
			input:    `<p>This link is <a href="news://news.server.example/*">valid</a></p>`,
			expected: `<p>This link is <a href="news://news.server.example/*" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "news scheme 2",
			input:    `<p>This link is <a href="news:example.group.this">valid</a></p>`,
			expected: `<p>This link is <a href="news:example.group.this" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "nntp scheme",
			input:    `<p>This link is <a href="nntp://news.server.example/example.group.this">valid</a></p>`,
			expected: `<p>This link is <a href="nntp://news.server.example/example.group.this" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "rtmp scheme",
			input:    `<p>This link is <a href="rtmp://mycompany.com/vod/mp4:mycoolvideo.mov">valid</a></p>`,
			expected: `<p>This link is <a href="rtmp://mycompany.com/vod/mp4:mycoolvideo.mov" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "sip scheme",
			input:    `<p>This link is <a href="sip:+1-212-555-1212:1234@gateway.com;user=phone">valid</a></p>`,
			expected: `<p>This link is <a href="sip:+1-212-555-1212:1234@gateway.com;user=phone" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "sips scheme",
			input:    `<p>This link is <a href="sips:alice@atlanta.com?subject=project%20x&amp;priority=urgent">valid</a></p>`,
			expected: `<p>This link is <a href="sips:alice@atlanta.com?subject=project%20x&amp;priority=urgent" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "skype scheme",
			input:    `<p>This link is <a href="skype:echo123?call">valid</a></p>`,
			expected: `<p>This link is <a href="skype:echo123?call" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "spotify scheme",
			input:    `<p>This link is <a href="spotify:track:2jCnn1QPQ3E8ExtLe6INsx">valid</a></p>`,
			expected: `<p>This link is <a href="spotify:track:2jCnn1QPQ3E8ExtLe6INsx" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "steam scheme",
			input:    `<p>This link is <a href="steam://settings/account">valid</a></p>`,
			expected: `<p>This link is <a href="steam://settings/account" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "svn scheme",
			input:    `<p>This link is <a href="svn://example.org">valid</a></p>`,
			expected: `<p>This link is <a href="svn://example.org" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "svn ssh scheme",
			input:    `<p>This link is <a href="svn://example.org">valid</a></p>`,
			expected: `<p>This link is <a href="svn://example.org" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "tel scheme",
			input:    `<p>This link is <a href="tel:+1-201-555-0123">valid</a></p>`,
			expected: `<p>This link is <a href="tel:+1-201-555-0123" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "webcal scheme",
			input:    `<p>This link is <a href="webcal://example.com/calendar.ics">valid</a></p>`,
			expected: `<p>This link is <a href="webcal://example.com/calendar.ics" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "xmpp scheme",
			input:    `<p>This link is <a href="xmpp:user@host?subscribe&amp;type=subscribed">valid</a></p>`,
			expected: `<p>This link is <a href="xmpp:user@host?subscribe&amp;type=subscribed" target="_blank" rel="nofollow noreferrer noopener">valid</a></p>`,
		},
		{
			name:     "blacklisted link",
			input:    `<p>This image is not valid <img src="https://stats.wordpress.com/some-tracker"></p>`,
			expected: `<p>This image is not valid </p>`,
		},
		{
			name:     "link with trackers",
			input:    `<p>This link has trackers <a href="https://example.com/page?utm_source=newsletter">Test</a></p>`,
			expected: `<p>This link has trackers <a href="https://example.com/page" target="_blank" rel="nofollow noreferrer noopener">Test</a></p>`,
		},
		{
			name:     "img src with trackers",
			input:    `<p>This image has trackers <img src="https://example.org/?id=123&utm_source=newsletter&utm_medium=email&fbclid=abc123"></p>`,
			expected: `<p>This image has trackers <img src="https://example.org/?id=123" loading="lazy"/></p>`,
		},
		{
			name:     "pixel tracker 0x0",
			input:    `<p><img src="https://tracker1.example.org/" height="0" width="0"> and <img src="https://tracker2.example.org/" height="0" width="0"/></p>`,
			expected: `<p> and </p>`,
		},
		{
			name:     "pixel tracker 1x1",
			input:    `<p><img src="https://tracker1.example.org/" height="1" width="1"> and <img src="https://tracker2.example.org/" height="1" width="1"/></p>`,
			expected: `<p> and </p>`,
		},
		{
			name:     "xml entities",
			input:    `<pre>echo "test" &gt; /etc/hosts</pre>`,
			expected: `<pre>echo &#34;test&#34; &gt; /etc/hosts</pre>`,
		},
		{
			name:     "espace attributes",
			input:    `<source sizes="<b>test</b>" src="https://example.org/image.jpg">test</source>`,
			expected: `<source sizes="&lt;b&gt;test&lt;/b&gt;" src="https://example.org/image.jpg"/>test`,
		},
		{
			name:     "replace youtube",
			input:    `<iframe src="http://www.youtube.com/embed/test123?version=3&#038;rel=1&#038;fs=1&#038;autohide=2&#038;showsearch=0&#038;showinfo=1&#038;iv_load_policy=1&#038;wmode=transparent"></iframe>`,
			expected: `<iframe src="https://www.invidious.custom/embed/test123?version=3&amp;rel=1&amp;fs=1&amp;autohide=2&amp;showsearch=0&amp;showinfo=1&amp;iv_load_policy=1&amp;wmode=transparent" loading="lazy" sandbox="allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox" credentialless=""></iframe>`,
		},
		{
			name:     "replace secure youtube",
			input:    `<iframe src="https://www.youtube.com/embed/test123"></iframe>`,
			expected: `<iframe src="https://www.invidious.custom/embed/test123" loading="lazy" sandbox="allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox" credentialless=""></iframe>`,
		},
		{
			name:     "replace secure youtube with params",
			input:    `<iframe src="https://www.youtube.com/embed/test123?rel=0&amp;controls=0"></iframe>`,
			expected: `<iframe src="https://www.invidious.custom/embed/test123?rel=0&amp;controls=0" loading="lazy" sandbox="allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox" credentialless=""></iframe>`,
		},
		{
			name:     "replace youtube already replaced",
			input:    `<iframe src="https://www.invidious.custom/embed/test123?rel=0&amp;controls=0" sandbox="allow-scripts allow-same-origin"></iframe>`,
			expected: `<iframe src="https://www.invidious.custom/embed/test123?rel=0&amp;controls=0" loading="lazy" sandbox="allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox" credentialless=""></iframe>`,
		},
		{
			name:     "replace protocol relative youtube",
			input:    `<iframe src="//www.youtube.com/embed/Bf2W84jrGqs" width="560" height="314" allowfullscreen="allowfullscreen"></iframe>`,
			expected: `<iframe src="https://www.invidious.custom/embed/Bf2W84jrGqs" width="560" height="314" allowfullscreen="allowfullscreen" loading="lazy" sandbox="allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox" credentialless=""></iframe>`,
		},
		{
			name:     "vimeo with query",
			input:    `<iframe src="https://player.vimeo.com/video/123456?title=0&amp;byline=0"></iframe>`,
			expected: `<iframe src="https://player.vimeo.com/video/123456?title=0&amp;byline=0&amp;dnt=1" loading="lazy" sandbox="allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox" credentialless=""></iframe>`,
		},
		{
			name:     "vimeo without query",
			input:    `<iframe src="https://player.vimeo.com/video/123456"></iframe>`,
			expected: `<iframe src="https://player.vimeo.com/video/123456?dnt=1" loading="lazy" sandbox="allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox" credentialless=""></iframe>`,
		},
		{
			name:     "replace noscript",
			input:    `<p>Before paragraph.</p><noscript>Inside <code>noscript</code> tag with an image: <img src="http://example.org/" alt="Test" loading="lazy"></noscript><p>After paragraph.</p>`,
			expected: `<p>Before paragraph.</p><p>After paragraph.</p>`,
		},
		{
			name:     "replace script",
			input:    `<p>Before paragraph.</p><script type="text/javascript">alert("1");</script><p>After paragraph.</p>`,
			expected: `<p>Before paragraph.</p><p>After paragraph.</p>`,
		},
		{
			name:     "replace style",
			input:    `<p>Before paragraph.</p><style>body { background-color: #ff0000; }</style><p>After paragraph.</p>`,
			expected: `<p>Before paragraph.</p><p>After paragraph.</p>`,
		},
		{
			name:     "hidden paragraph",
			input:    `<p>Before paragraph.</p><p hidden>This should <em>not</em> appear in the <strong>output</strong></p><p>After paragraph.</p>`,
			expected: `<p>Before paragraph.</p><p>After paragraph.</p>`,
		},
		{
			name:     "attributes stripped",
			input:    `<p style="color: red;">Some text.<hr style="color: blue"/>Test.</p>`,
			expected: `<p>Some text.<hr/>Test.</p>`,
		},
		{
			name:     "mathml",
			input:    `<math xmlns="http://www.w3.org/1998/Math/MathML"><msup><mi>x</mi><mn>2</mn></msup></math>`,
			expected: `<math xmlns="http://www.w3.org/1998/Math/MathML"><msup><mi>x</mi><mn>2</mn></msup></math>`,
		},
		{
			name:     "blocked resources 1",
			input:    `<p>Before paragraph.</p><img src="http://stats.wordpress.com/something.php" alt="Blocked Resource"><p>After paragraph.</p>`,
			expected: `<p>Before paragraph.</p><p>After paragraph.</p>`,
		},
		{
			name:     "blocked resources 2",
			input:    `<p>Before paragraph.</p><img src="http://twitter.com/share?text=This+is+google+a+search+engine&url=https%3A%2F%2Fwww.google.com" alt="Blocked Resource"><p>After paragraph.</p>`,
			expected: `<p>Before paragraph.</p><p>After paragraph.</p>`,
		},
		{
			name:     "blocked resources 3",
			input:    `<p>Before paragraph.</p><img src="http://www.facebook.com/sharer.php?u=https%3A%2F%2Fwww.google.com%[title]=This+Is%2C+Google+a+search+engine" alt="Blocked Resource"><p>After paragraph.</p>`,
			expected: `<p>Before paragraph.</p><p>After paragraph.</p>`,
		},
	}

	t.Setenv("YOUTUBE_EMBED_URL_OVERRIDE", "https://www.invidious.custom/embed/")
	require.NoError(t, config.Load(""))

	pageURL, err := url.Parse("https://example.org/foo.html")
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, SanitizeContent(tt.input, pageURL))
		})
	}
}
