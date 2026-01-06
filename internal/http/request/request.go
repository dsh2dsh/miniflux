package request

import (
	"net/http"
	"path"

	"miniflux.app/v2/internal/config"
)

func RequestURI(r *http.Request) string {
	if bp := config.BasePath(); bp != "" {
		return path.Join(bp, r.URL.RequestURI())
	}
	return r.URL.RequestURI()
}
