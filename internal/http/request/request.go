package request

import (
	"net/http"
	"net/url"
	"path"

	"miniflux.app/v2/internal/config"
)

func URI(r *http.Request) string { return baseRequestURI(r.URL) }

func baseRequestURI(u *url.URL) string {
	if bp := config.BasePath(); bp != "" {
		return path.Join(bp, u.RequestURI())
	}
	return u.RequestURI()
}

func URIWithQuery(r *http.Request, keyValues ...string) string {
	if len(keyValues) < 2 {
		return URI(r)
	}

	values := r.URL.Query()
	for i := 0; i < len(keyValues)-1; i += 2 {
		values.Set(keyValues[i], keyValues[i+1])
	}

	u := *r.URL
	u.RawQuery = values.Encode()
	return baseRequestURI(&u)
}
