package ui

import (
	"strings"
	"testing"

	"miniflux.app/v2/internal/model"
)

func TestCSPExternalFont(t *testing.T) {
	want := []string{
		`default-src 'none';`,
		`img-src * data:;`,
		`media-src *;`,
		`frame-src *;`,
		`style-src 'nonce-1234';`,
		`script-src 'nonce-1234'`,
		`'strict-dynamic';`,
		`font-src test.com;`,
		`manifest-src 'self';`,
	}
	csp := contentSecurityPolicy{nonce: "1234"}
	got := csp.Content(&model.User{ExternalFontHosts: "test.com"})

	for _, value := range want {
		if !strings.Contains(got, value) {
			t.Errorf(`Unexpected result, didn't find %q in %q`, value, got)
		}
	}
}

func TestCSPNoUser(t *testing.T) {
	want := []string{
		`default-src 'none';`,
		`img-src * data:;`,
		`media-src *;`,
		`frame-src *;`,
		`style-src 'nonce-1234';`,
		`script-src 'nonce-1234'`,
		`'strict-dynamic';`,
		`manifest-src 'self';`,
	}
	csp := contentSecurityPolicy{nonce: "1234"}
	got := csp.Content(nil)

	for _, value := range want {
		if !strings.Contains(got, value) {
			t.Errorf(`Unexpected result, didn't find %q in %q`, value, got)
		}
	}
}

func TestCSPCustomJSExternalFont(t *testing.T) {
	want := []string{
		`default-src 'none';`,
		`img-src * data:;`,
		`media-src *;`,
		`frame-src *;`,
		`style-src 'nonce-1234';`,
		`script-src 'nonce-1234'`,
		`'strict-dynamic';`,
		`manifest-src 'self';`,
	}
	csp := contentSecurityPolicy{nonce: "1234"}
	got := csp.Content(&model.User{ExternalFontHosts: "test.com", CustomJS: "alert(1)"})

	for _, value := range want {
		if !strings.Contains(got, value) {
			t.Errorf(`Unexpected result, didn't find %q in %q`, value, got)
		}
	}
}

func TestCSPExternalFontStylesheet(t *testing.T) {
	want := []string{
		`default-src 'none';`,
		`img-src * data:;`,
		`media-src *;`,
		`frame-src *;`,
		`style-src 'nonce-1234' test.com;`,
		`script-src 'nonce-1234'`,
		`'strict-dynamic';`,
		`manifest-src 'self';`,
	}
	csp := contentSecurityPolicy{nonce: "1234"}
	got := csp.Content(&model.User{ExternalFontHosts: "test.com", Stylesheet: "a {color: red;}"})

	for _, value := range want {
		if !strings.Contains(got, value) {
			t.Errorf(`Unexpected result, didn't find %q in %q`, value, got)
		}
	}
}
