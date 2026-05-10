package template

import (
	"bytes"
	"fmt"
	"html/template"
	"time"

	"miniflux.app/v2/internal/locale"
)

type Template struct {
	*template.Template

	printer *locale.Printer
}

func (self *Template) LookupExecute(data map[string]any, names ...string,
) ([]byte, error) {
	self.Funcs(self.funcMap())

	for _, name := range names {
		if t := self.Lookup(name); t != nil {
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
		"elapsed": func(timezone string, t time.Time) string {
			return elapsedTime(self.printer, timezone, t)
		},
		"t":      self.printer.Printf,
		"plural": self.printer.Plural,
	}
}
