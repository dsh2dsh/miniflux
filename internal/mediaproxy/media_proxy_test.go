// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mediaproxy // import "miniflux.app/v2/internal/mediaproxy"

import (
	_ "embed"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/mux"
	"miniflux.app/v2/internal/model"
)

//go:embed testdata/miniflux_wikipedia.html
var wikipediaHTML string

func BenchmarkProxy(b *testing.B) {
	b.Setenv("MEDIA_PROXY_MODE", "all")
	b.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image")
	b.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")
	require.NoError(b, config.Load(""))

	m := mux.New()
	m.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}",
		func(http.ResponseWriter, *http.Request) {}, "proxy")

	b.ReportAllocs()
	for b.Loop() {
		RewriteDocumentWithRelativeProxyURL(m, wikipediaHTML)
	}
}

func TestProxyFilterWithHttpDefault(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "http-only")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}", func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<p><img src="http://website/folder/image.png" alt="Test"/></p>`
	output := RewriteDocumentWithRelativeProxyURL(r, input)
	expected := `<p><img src="/proxy/okK5PsdNY8F082UMQEAbLPeUFfbe2WnNfInNmR9T4WA=/aHR0cDovL3dlYnNpdGUvZm9sZGVyL2ltYWdlLnBuZw==" alt="Test"/></p>`

	if expected != output {
		t.Errorf(`Not expected output: got %q instead of %q`, output, expected)
	}
}

func TestProxyFilterWithHttpsDefault(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "http-only")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image")

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}", func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<p><img src="https://website/folder/image.png" alt="Test"/></p>`
	output := RewriteDocumentWithRelativeProxyURL(r, input)
	expected := `<p><img src="https://website/folder/image.png" alt="Test"/></p>`

	if expected != output {
		t.Errorf(`Not expected output: got %q instead of %q`, output, expected)
	}
}

func TestProxyFilterWithHttpNever(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "none")

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}", func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<p><img src="http://website/folder/image.png" alt="Test"/></p>`
	output := RewriteDocumentWithRelativeProxyURL(r, input)
	expected := input

	if expected != output {
		t.Errorf(`Not expected output: got %q instead of %q`, output, expected)
	}
}

func TestProxyFilterWithHttpsNever(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "none")

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}", func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<p><img src="https://website/folder/image.png" alt="Test"/></p>`
	output := RewriteDocumentWithRelativeProxyURL(r, input)
	expected := input

	if expected != output {
		t.Errorf(`Not expected output: got %q instead of %q`, output, expected)
	}
}

func TestProxyFilterWithHttpAlways(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "all")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}", func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<p><img src="http://website/folder/image.png" alt="Test"/></p>`
	output := RewriteDocumentWithRelativeProxyURL(r, input)
	expected := `<p><img src="/proxy/okK5PsdNY8F082UMQEAbLPeUFfbe2WnNfInNmR9T4WA=/aHR0cDovL3dlYnNpdGUvZm9sZGVyL2ltYWdlLnBuZw==" alt="Test"/></p>`

	if expected != output {
		t.Errorf(`Not expected output: got %q instead of %q`, output, expected)
	}
}

func TestProxyFilterWithHttpsAlways(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "all")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}", func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<p><img src="https://website/folder/image.png" alt="Test"/></p>`
	output := RewriteDocumentWithRelativeProxyURL(r, input)
	expected := `<p><img src="/proxy/LdPNR1GBDigeeNp2ArUQRyZsVqT_PWLfHGjYFrrWWIY=/aHR0cHM6Ly93ZWJzaXRlL2ZvbGRlci9pbWFnZS5wbmc=" alt="Test"/></p>`

	if expected != output {
		t.Errorf(`Not expected output: got %q instead of %q`, output, expected)
	}
}

func TestAbsoluteProxyFilterWithHttpsAlways(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "all")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}", func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<p><img src="https://website/folder/image.png" alt="Test"/></p>`
	output := RewriteDocumentWithAbsoluteProxyURL(r, input)
	expected := `<p><img src="http://localhost/proxy/LdPNR1GBDigeeNp2ArUQRyZsVqT_PWLfHGjYFrrWWIY=/aHR0cHM6Ly93ZWJzaXRlL2ZvbGRlci9pbWFnZS5wbmc=" alt="Test"/></p>`

	if expected != output {
		t.Errorf(`Not expected output: got %q instead of %q`, output, expected)
	}
}

func TestAbsoluteProxyFilterWithCustomPortAndSubfolderInBaseURL(t *testing.T) {
	os.Clearenv()
	t.Setenv("BASE_URL", "http://example.org:88/folder/")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	if config.BaseURL() != "http://example.org:88/folder" {
		t.Fatalf(`Unexpected base URL, got "%s"`, config.BaseURL())
	}

	if config.RootURL() != "http://example.org:88" {
		t.Fatalf(`Unexpected root URL, got "%s"`, config.RootURL())
	}

	router := mux.New()
	if config.BasePath() != "" {
		router = router.PrefixGroup(config.BasePath())
	}

	router.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}", func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<p><img src="http://website/folder/image.png" alt="Test"/></p>`
	output := RewriteDocumentWithAbsoluteProxyURL(router, input)
	expected := `<p><img src="http://example.org:88/folder/proxy/okK5PsdNY8F082UMQEAbLPeUFfbe2WnNfInNmR9T4WA=/aHR0cDovL3dlYnNpdGUvZm9sZGVyL2ltYWdlLnBuZw==" alt="Test"/></p>`

	if expected != output {
		t.Errorf(`Not expected output: got %q instead of %q`, output, expected)
	}
}

func TestAbsoluteProxyFilterWithHttpsAlwaysAndAudioTag(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "all")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "audio")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}", func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<audio src="https://website/folder/audio.mp3"></audio>`
	output := RewriteDocumentWithAbsoluteProxyURL(r, input)
	expected := `<audio src="http://localhost/proxy/EmBTvmU5B17wGuONkeknkptYopW_Tl6Y6_W8oYbN_Xs=/aHR0cHM6Ly93ZWJzaXRlL2ZvbGRlci9hdWRpby5tcDM="></audio>`

	if expected != output {
		t.Errorf(`Not expected output: got %q instead of %q`, output, expected)
	}
}

func TestProxyFilterWithHttpsAlwaysAndCustomProxyServer(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "all")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image")
	t.Setenv("MEDIA_PROXY_CUSTOM_URL", "https://proxy-example/proxy")

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}", func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<p><img src="https://website/folder/image.png" alt="Test"/></p>`
	output := RewriteDocumentWithRelativeProxyURL(r, input)
	expected := `<p><img src="https://proxy-example/proxy/aHR0cHM6Ly93ZWJzaXRlL2ZvbGRlci9pbWFnZS5wbmc=" alt="Test"/></p>`

	if expected != output {
		t.Errorf(`Not expected output: got %q instead of %q`, output, expected)
	}
}

func TestProxyFilterWithHttpsAlwaysAndIncorrectCustomProxyServer(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "all")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image")
	t.Setenv("MEDIA_PROXY_CUSTOM_URL", "http://:8080example.com")

	err := config.Load("")
	require.ErrorContains(t, err, "env: parse error on field")
}

func TestAbsoluteProxyFilterWithHttpsAlwaysAndCustomProxyServer(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "all")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image")
	t.Setenv("MEDIA_PROXY_CUSTOM_URL", "https://proxy-example/proxy")

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}", func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<p><img src="https://website/folder/image.png" alt="Test"/></p>`
	output := RewriteDocumentWithAbsoluteProxyURL(r, input)
	expected := `<p><img src="https://proxy-example/proxy/aHR0cHM6Ly93ZWJzaXRlL2ZvbGRlci9pbWFnZS5wbmc=" alt="Test"/></p>`

	if expected != output {
		t.Errorf(`Not expected output: got %q instead of %q`, output, expected)
	}
}

func TestProxyFilterWithHttpInvalid(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "http-only")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}", func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<p><img src="http://website/folder/image.png" alt="Test"/></p>`
	output := RewriteDocumentWithRelativeProxyURL(r, input)
	expected := `<p><img src="/proxy/okK5PsdNY8F082UMQEAbLPeUFfbe2WnNfInNmR9T4WA=/aHR0cDovL3dlYnNpdGUvZm9sZGVyL2ltYWdlLnBuZw==" alt="Test"/></p>`

	if expected != output {
		t.Errorf(`Not expected output: got %q instead of %q`, output, expected)
	}
}

func TestProxyFilterWithHttpsInvalid(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "http-only")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}", func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<p><img src="https://website/folder/image.png" alt="Test"/></p>`
	output := RewriteDocumentWithRelativeProxyURL(r, input)
	expected := `<p><img src="https://website/folder/image.png" alt="Test"/></p>`

	if expected != output {
		t.Errorf(`Not expected output: got %q instead of %q`, output, expected)
	}
}

func TestProxyFilterWithSrcset(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "all")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")

	require.NoError(t, config.Load(""))

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}",
		func(http.ResponseWriter, *http.Request) {}, "proxy")

	input := `<p><img src="http://website/folder/image.png" srcset="http://website/folder/image2.png 656w, http://website/folder/image3.png 360w" alt="test"></p>`
	expected := `<p><img src="/proxy/okK5PsdNY8F082UMQEAbLPeUFfbe2WnNfInNmR9T4WA=/aHR0cDovL3dlYnNpdGUvZm9sZGVyL2ltYWdlLnBuZw==" srcset="/proxy/aY5Hb4urDnUCly2vTJ7ExQeeaVS-52O7kjUr2v9VrAs=/aHR0cDovL3dlYnNpdGUvZm9sZGVyL2ltYWdlMi5wbmc= 656w, /proxy/QgAmrJWiAud_nNAsz3F8OTxaIofwAiO36EDzH_YfMzo=/aHR0cDovL3dlYnNpdGUvZm9sZGVyL2ltYWdlMy5wbmc= 360w" alt="test"/></p>`

	assert.Equal(t, expected, RewriteDocumentWithRelativeProxyURL(r, input))
}

func TestProxyFilterWithEmptySrcset(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "all")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")
	require.NoError(t, config.Load(""))

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}",
		func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<p><img src="http://website/folder/image.png" srcset="" alt="test"></p>`
	expected := `<p><img src="/proxy/okK5PsdNY8F082UMQEAbLPeUFfbe2WnNfInNmR9T4WA=/aHR0cDovL3dlYnNpdGUvZm9sZGVyL2ltYWdlLnBuZw==" alt="test"/></p>`
	assert.Equal(t, expected, RewriteDocumentWithRelativeProxyURL(r, input))
}

func TestProxyFilterWithPictureSource(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "all")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")
	require.NoError(t, config.Load(""))

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}",
		func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<picture><source srcset="http://website/folder/image2.png 656w,   http://website/folder/image3.png 360w, https://website/some,image.png 2x"></picture>`
	expected := `<picture><source srcset="/proxy/aY5Hb4urDnUCly2vTJ7ExQeeaVS-52O7kjUr2v9VrAs=/aHR0cDovL3dlYnNpdGUvZm9sZGVyL2ltYWdlMi5wbmc= 656w, /proxy/QgAmrJWiAud_nNAsz3F8OTxaIofwAiO36EDzH_YfMzo=/aHR0cDovL3dlYnNpdGUvZm9sZGVyL2ltYWdlMy5wbmc= 360w, /proxy/ZIw0hv8WhSTls5aSqhnFaCXlUrKIqTnBRaY0-NaLnds=/aHR0cHM6Ly93ZWJzaXRlL3NvbWUsaW1hZ2UucG5n 2x"/></picture>`
	assert.Equal(t, expected, RewriteDocumentWithRelativeProxyURL(r, input))
}

func TestProxyFilterOnlyNonHTTPWithPictureSource(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "http-only")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")
	require.NoError(t, config.Load(""))

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}",
		func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<picture><source srcset="http://website/folder/image2.png 656w, https://website/some,image.png 2x"></picture>`
	expected := `<picture><source srcset="/proxy/aY5Hb4urDnUCly2vTJ7ExQeeaVS-52O7kjUr2v9VrAs=/aHR0cDovL3dlYnNpdGUvZm9sZGVyL2ltYWdlMi5wbmc= 656w, https://website/some,image.png 2x"/></picture>`
	assert.Equal(t, expected, RewriteDocumentWithRelativeProxyURL(r, input))
}

func TestProxyWithImageDataURL(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "all")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image")
	require.NoError(t, config.Load(""))

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}",
		func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<img src="data:image/gif;base64,test"/>`
	output := RewriteDocumentWithRelativeProxyURL(r, input)
	assert.Equal(t, input, output)
}

func TestProxyWithImageSourceDataURL(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "all")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")
	require.NoError(t, config.Load(""))

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}",
		func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<picture><source srcset="data:image/gif;base64,test"/></picture>`
	expected := `<picture><source srcset="data:image/gif;base64,test"/></picture>`
	assert.Equal(t, expected, RewriteDocumentWithRelativeProxyURL(r, input))
}

func TestProxyFilterWithVideo(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "all")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "video")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")
	require.NoError(t, config.Load(""))

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}",
		func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<video poster="https://example.com/img.png" src="https://example.com/video.mp4"></video>`
	expected := `<video poster="https://example.com/img.png" src="/proxy/0y3LR8zlx8S8qJkj1qWFOO6x3a-5yf2gLWjGIJV5yyc=/aHR0cHM6Ly9leGFtcGxlLmNvbS92aWRlby5tcDQ="></video>`
	assert.Equal(t, expected, RewriteDocumentWithRelativeProxyURL(r, input))
}

func TestProxyFilterVideoPoster(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "all")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")
	require.NoError(t, config.Load(""))

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}",
		func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<video poster="https://example.com/img.png" src="https://example.com/video.mp4"></video>`
	expected := `<video poster="/proxy/aDFfroYL57q5XsojIzATT6OYUCkuVSPXYJQAVrotnLw=/aHR0cHM6Ly9leGFtcGxlLmNvbS9pbWcucG5n" src="https://example.com/video.mp4"></video>`
	assert.Equal(t, expected, RewriteDocumentWithRelativeProxyURL(r, input))
}

func TestProxyFilterVideoPosterOnce(t *testing.T) {
	os.Clearenv()
	t.Setenv("MEDIA_PROXY_MODE", "all")
	t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", "image,video")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")

	err := config.Load("")
	if err != nil {
		t.Fatalf(`Parsing failure: %v`, err)
	}

	r := mux.New()
	r.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}", func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	input := `<video poster="https://example.com/img.png" src="https://example.com/video.mp4"></video>`
	expected := `<video poster="/proxy/aDFfroYL57q5XsojIzATT6OYUCkuVSPXYJQAVrotnLw=/aHR0cHM6Ly9leGFtcGxlLmNvbS9pbWcucG5n" src="/proxy/0y3LR8zlx8S8qJkj1qWFOO6x3a-5yf2gLWjGIJV5yyc=/aHR0cHM6Ly9leGFtcGxlLmNvbS92aWRlby5tcDQ="></video>`
	output := RewriteDocumentWithRelativeProxyURL(r, input)

	if expected != output {
		t.Errorf(`Not expected output: got %s`, output)
	}
}

func TestProxifyAbsoluteURL(t *testing.T) {
	tests := []struct {
		name                    string
		mediaURL                string
		mediaMimeType           string
		mediaProxyOption        string
		mediaProxyResourceTypes string
		expected                string
	}{
		{
			name:                    "Empty URL should not be proxified",
			mediaURL:                "",
			mediaMimeType:           "image/jpeg",
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "image",
		},
		{
			name:                    "Data URL should not be proxified",
			mediaURL:                "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==",
			mediaMimeType:           "image/png",
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "image",
			expected:                "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==",
		},
		{
			name:                    "HTTP URL with all mode and matching MIME type should be proxified",
			mediaURL:                "http://example.com/image.jpg",
			mediaMimeType:           "image/jpeg",
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "image",
			expected:                "http://localhost/proxy/_rzaC4Dl22I2O1tEhyYF6Aaj9Bkd3fC4plmB4JP9I5c=/aHR0cDovL2V4YW1wbGUuY29tL2ltYWdlLmpwZw==",
		},
		{
			name:                    "HTTPS URL with all mode and matching MIME type should be proxified",
			mediaURL:                "https://example.com/image.jpg",
			mediaMimeType:           "image/jpeg",
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "image",
			expected:                "http://localhost/proxy/KwOr2nVplyXP97zKHxFaouUuMkOjIf7DiLy8lcBmNao=/aHR0cHM6Ly9leGFtcGxlLmNvbS9pbWFnZS5qcGc=",
		},
		{
			name:                    "HTTP URL with http-only mode and matching MIME type should be proxified",
			mediaURL:                "http://example.com/image.jpg",
			mediaMimeType:           "image/jpeg",
			mediaProxyOption:        "http-only",
			mediaProxyResourceTypes: "image",
			expected:                "http://localhost/proxy/_rzaC4Dl22I2O1tEhyYF6Aaj9Bkd3fC4plmB4JP9I5c=/aHR0cDovL2V4YW1wbGUuY29tL2ltYWdlLmpwZw==",
		},
		{
			name:                    "HTTPS URL with http-only mode should not be proxified",
			mediaURL:                "https://example.com/image.jpg",
			mediaMimeType:           "image/jpeg",
			mediaProxyOption:        "http-only",
			mediaProxyResourceTypes: "image",
			expected:                "https://example.com/image.jpg",
		},
		{
			name:                    "URL with none mode should not be proxified",
			mediaURL:                "http://example.com/image.jpg",
			mediaMimeType:           "image/jpeg",
			mediaProxyOption:        "none",
			mediaProxyResourceTypes: "image",
			expected:                "http://example.com/image.jpg",
		},
		{
			name:                    "URL with matching MIME type should be proxified",
			mediaURL:                "http://example.com/video.mp4",
			mediaMimeType:           "video/mp4",
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "video",
			expected:                "http://localhost/proxy/GdyLH_DE6rY0UA03nvvKFeqNTfR5zi9de73dHIeZvok=/aHR0cDovL2V4YW1wbGUuY29tL3ZpZGVvLm1wNA==",
		},
		{
			name:                    "URL with non-matching MIME type should not be proxified",
			mediaURL:                "http://example.com/video.mp4",
			mediaMimeType:           "video/mp4",
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "image",
			expected:                "http://example.com/video.mp4",
		},
		{
			name:                    "URL with multiple resource types and matching MIME type should be proxified",
			mediaURL:                "http://example.com/audio.mp3",
			mediaMimeType:           "audio/mp3",
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "image,audio,video",
			expected:                "http://localhost/proxy/gh3uo3dUkPvaPew0199KILUsFSIv8Ju06k45Q06L2Tw=/aHR0cDovL2V4YW1wbGUuY29tL2F1ZGlvLm1wMw==",
		},
		{
			name:                    "URL with multiple resource types but non-matching MIME type should not be proxified",
			mediaURL:                "http://example.com/document.pdf",
			mediaMimeType:           "application/pdf",
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "image,audio,video",
			expected:                "http://example.com/document.pdf",
		},
		{
			name:                    "URL with partial MIME type match should be proxified",
			mediaURL:                "http://example.com/image.jpg",
			mediaMimeType:           "image/jpeg",
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "image",
			expected:                "http://localhost/proxy/_rzaC4Dl22I2O1tEhyYF6Aaj9Bkd3fC4plmB4JP9I5c=/aHR0cDovL2V4YW1wbGUuY29tL2ltYWdlLmpwZw==",
		},
		{
			name:                    "URL with audio MIME type and audio resource type should be proxified",
			mediaURL:                "http://example.com/song.ogg",
			mediaMimeType:           "audio/ogg",
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "audio",
			expected:                "http://localhost/proxy/Y-pNt-d5MBktNwED8_Oe9xNv06toNjE_duGoR9L63VA=/aHR0cDovL2V4YW1wbGUuY29tL3Nvbmcub2dn",
		},
		{
			name:                    "URL with video MIME type and video resource type should be proxified",
			mediaURL:                "http://example.com/movie.webm",
			mediaMimeType:           "video/webm",
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "video",
			expected:                "http://localhost/proxy/KKBj8WuwPUGtqUzfHGr1u3AFluy419BTmH-53ui6atU=/aHR0cDovL2V4YW1wbGUuY29tL21vdmllLndlYm0=",
		},
	}

	os.Clearenv()
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test")

	m := mux.New()
	m.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}",
		func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MEDIA_PROXY_MODE", tt.mediaProxyOption)
			t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", tt.mediaProxyResourceTypes)
			require.NoError(t, config.Load(""))

			result := proxifyAbsoluteURL(m, tt.mediaMimeType, tt.mediaURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProxifyEnclosures_urls(t *testing.T) {
	tests := []struct {
		name                    string
		url                     string
		mimeType                string
		mediaProxyOption        string
		mediaProxyResourceTypes string
		expectedURLChanged      bool
	}{
		{
			name:                    "HTTP URL with audio type - proxy mode all",
			url:                     "http://example.com/audio.mp3",
			mimeType:                "audio/mpeg",
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "audio,video",
			expectedURLChanged:      true,
		},
		{
			name:                    "HTTPS URL with video type - proxy mode all",
			url:                     "https://example.com/video.mp4",
			mimeType:                "video/mp4",
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "audio,video",
			expectedURLChanged:      true,
		},
		{
			name:                    "HTTP URL with video type - proxy mode http-only",
			url:                     "http://example.com/video.mp4",
			mimeType:                "video/mp4",
			mediaProxyOption:        "http-only",
			mediaProxyResourceTypes: "audio,video",
			expectedURLChanged:      true,
		},
		{
			name:                    "HTTPS URL with video type - proxy mode http-only",
			url:                     "https://example.com/video.mp4",
			mimeType:                "video/mp4",
			mediaProxyOption:        "http-only",
			mediaProxyResourceTypes: "audio,video",
		},
		{
			name:                    "HTTP URL with image type - not in resource types",
			url:                     "http://example.com/image.jpg",
			mimeType:                "image/jpeg",
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "audio,video",
		},
		{
			name:                    "HTTP URL with image type - in resource types",
			url:                     "http://example.com/image.jpg",
			mimeType:                "image/jpeg",
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "audio,video,image",
			expectedURLChanged:      true,
		},
		{
			name:                    "HTTP URL - proxy mode none",
			url:                     "http://example.com/audio.mp3",
			mimeType:                "audio/mpeg",
			mediaProxyOption:        "none",
			mediaProxyResourceTypes: "audio,video",
		},
		{
			name:                    "Empty URL",
			url:                     "",
			mimeType:                "audio/mpeg",
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "audio,video",
		},
		{
			name:                    "Non-media MIME type",
			url:                     "http://example.com/doc.pdf",
			mimeType:                "application/pdf",
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "audio,video",
		},
	}

	router := mux.New()
	router.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}",
		func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	os.Clearenv()
	t.Setenv("BASE_URL", "http://localhost")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test-private-key")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MEDIA_PROXY_MODE", tt.mediaProxyOption)
			t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", tt.mediaProxyResourceTypes)
			require.NoError(t, config.Load(""))

			enclosures := []model.Enclosure{{
				URL:      tt.url,
				MimeType: tt.mimeType,
			}}

			originalURL := enclosures[0].URL

			// Call the method
			ProxifyEnclosures(router, enclosures)

			// Check if URL changed as expected
			if !tt.expectedURLChanged {
				assert.Equal(t, originalURL, enclosures[0].URL)
				return
			}

			// If URL should have changed, verify it's not empty
			assert.NotEmpty(t, enclosures[0].URL)
			assert.NotEqual(t, originalURL, enclosures[0].URL)
		})
	}
}

func TestProxifyEnclosures_list(t *testing.T) {
	// Initialize config for testing
	os.Clearenv()
	t.Setenv("BASE_URL", "http://localhost")
	t.Setenv("MEDIA_PROXY_PRIVATE_KEY", "test-private-key")

	router := mux.New()
	router.NameHandleFunc("/proxy/{encodedDigest}/{encodedURL}",
		func(w http.ResponseWriter, r *http.Request) {}, "proxy")

	tests := []struct {
		name                    string
		enclosures              []model.Enclosure
		mediaProxyOption        string
		mediaProxyResourceTypes string
		expectedChangedCount    int
	}{
		{
			name: "Mixed enclosures with all proxy mode",
			enclosures: []model.Enclosure{
				{URL: "http://example.com/audio.mp3", MimeType: "audio/mpeg"},
				{URL: "https://example.com/video.mp4", MimeType: "video/mp4"},
				{URL: "http://example.com/image.jpg", MimeType: "image/jpeg"},
				{URL: "http://example.com/doc.pdf", MimeType: "application/pdf"},
			},
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "audio,video",
			expectedChangedCount:    2, // audio and video should be proxified
		},
		{
			name: "Mixed enclosures with http-only proxy mode",
			enclosures: []model.Enclosure{
				{URL: "http://example.com/audio.mp3", MimeType: "audio/mpeg"},
				{URL: "https://example.com/video.mp4", MimeType: "video/mp4"},
				{URL: "http://example.com/video2.mp4", MimeType: "video/mp4"},
			},
			mediaProxyOption:        "http-only",
			mediaProxyResourceTypes: "audio,video",
			expectedChangedCount:    2, // only HTTP URLs should be proxified
		},
		{
			name: "No media types in resource list",
			enclosures: []model.Enclosure{
				{URL: "http://example.com/audio.mp3", MimeType: "audio/mpeg"},
				{URL: "http://example.com/video.mp4", MimeType: "video/mp4"},
			},
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "image",
			expectedChangedCount:    0, // no matching resource types
		},
		{
			name: "Proxy mode none",
			enclosures: []model.Enclosure{
				{URL: "http://example.com/audio.mp3", MimeType: "audio/mpeg"},
				{URL: "http://example.com/video.mp4", MimeType: "video/mp4"},
			},
			mediaProxyOption:        "none",
			mediaProxyResourceTypes: "audio,video",
			expectedChangedCount:    0,
		},
		{
			name:                    "Empty enclosure list",
			enclosures:              []model.Enclosure{},
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "audio,video",
			expectedChangedCount:    0,
		},
		{
			name: "Enclosures with empty URLs",
			enclosures: []model.Enclosure{
				{URL: "", MimeType: "audio/mpeg"},
				{URL: "http://example.com/video.mp4", MimeType: "video/mp4"},
			},
			mediaProxyOption:        "all",
			mediaProxyResourceTypes: "audio,video",
			expectedChangedCount:    1, // only the non-empty URL should be processed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MEDIA_PROXY_MODE", tt.mediaProxyOption)
			t.Setenv("MEDIA_PROXY_RESOURCE_TYPES", tt.mediaProxyResourceTypes)
			require.NoError(t, config.Load(""))

			// Store original URLs
			originalURLs := make([]string, len(tt.enclosures))
			for i, enclosure := range tt.enclosures {
				originalURLs[i] = enclosure.URL
			}

			// Call the method
			ProxifyEnclosures(router, tt.enclosures)

			// Count how many URLs actually changed
			changedCount := 0
			for i, enclosure := range tt.enclosures {
				if enclosure.URL != originalURLs[i] {
					changedCount++
					// Verify that changed URLs are not empty (unless they were empty originally)
					if originalURLs[i] != "" && enclosure.URL == "" {
						t.Errorf("Enclosure %d: ProxifyEnclosureURL resulted in empty URL", i)
					}
				}
			}

			if changedCount != tt.expectedChangedCount {
				t.Errorf("ProxifyEnclosureURL() changed %d URLs, want %d", changedCount, tt.expectedChangedCount)
			}
		})
	}
}
