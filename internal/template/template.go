package template

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/locale"
)

type Template struct {
	*template.Template

	printer *locale.Printer
	r       *http.Request
}

func (self *Template) LookupExecute(data map[string]any, names ...string,
) ([]byte, error) {
	tt := self.Funcs(self.funcMap())

	for _, name := range names {
		if t := tt.Lookup(name); t != nil {
			var b bytes.Buffer
			if err := t.Execute(&b, data); err != nil {
				return nil, fmt.Errorf("template: executing %q: %w", name, err)
			}
			return b.Bytes(), nil
		}
	}
	return nil, fmt.Errorf("none of [%v] defined", names)
}

func (self *Template) funcMap() template.FuncMap {
	return template.FuncMap{
		"requestURI": self.requestURI,

		"elapsed": func(timezone string, t time.Time) string {
			return elapsedTime(self.printer, timezone, t)
		},
		"t":      self.printer.Printf,
		"plural": self.printer.Plural,
	}
}

func (self *Template) requestURI(keyValues ...string) string {
	return request.URIWithQuery(self.r, keyValues...)
}
