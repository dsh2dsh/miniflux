// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package template // import "miniflux.app/v2/internal/template"

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"time"

	"miniflux.app/v2/internal/http/mux"
	"miniflux.app/v2/internal/locale"
)

//go:embed templates/common/*.html templates/common/*.svg
var commonTemplateFiles embed.FS

//go:embed templates/views/*.html
var viewTemplateFiles embed.FS

//go:embed templates/standalone/*.html
var standaloneTemplateFiles embed.FS

// Engine handles the templating system.
type Engine struct {
	templates map[string]*template.Template
	funcMap   *funcMap
}

// NewEngine returns a new template engine.
func NewEngine(router *mux.ServeMux) *Engine {
	return &Engine{
		templates: make(map[string]*template.Template),
		funcMap:   &funcMap{router},
	}
}

// ParseTemplates parses template files embed into the application.
func (e *Engine) ParseTemplates() error {
	funcMap := e.funcMap.Map()
	commonTemplates := template.Must(template.New("").
		Funcs(funcMap).ParseFS(commonTemplateFiles, "templates/common/*"))

	dirEntries, err := viewTemplateFiles.ReadDir("templates/views")
	if err != nil {
		return fmt.Errorf("template: filed read templates/common/: %w", err)
	}
	for _, dirEntry := range dirEntries {
		fullName := "templates/views/" + dirEntry.Name()
		slog.Debug("Parsing template", slog.String("template_name", fullName))
		commonTemplatesClone, err := commonTemplates.Clone()
		if err != nil {
			panic("Unable to clone the common template")
		}
		e.templates[dirEntry.Name()] = template.Must(commonTemplatesClone.ParseFS(viewTemplateFiles, fullName))
	}

	dirEntries, err = standaloneTemplateFiles.ReadDir("templates/standalone")
	if err != nil {
		return fmt.Errorf("template: failed read templates/standalone/: %w", err)
	}
	for _, dirEntry := range dirEntries {
		fullName := "templates/standalone/" + dirEntry.Name()
		slog.Debug("Parsing template", slog.String("template_name", fullName))
		e.templates[dirEntry.Name()] = template.Must(template.New(dirEntry.Name()).Funcs(funcMap).ParseFS(standaloneTemplateFiles, fullName))
	}
	return nil
}

// Render process a template.
func (e *Engine) Render(name string, data map[string]any) []byte {
	tpl, ok := e.templates[name]
	if !ok {
		panic("This template does not exists: " + name)
	}
	tpl = template.Must(tpl.Clone())

	// Functions that need to be declared at runtime.
	printer := locale.NewPrinter(data["language"].(string))
	tpl.Funcs(template.FuncMap{
		"elapsed": func(timezone string, t time.Time) string {
			return elapsedTime(printer, timezone, t)
		},
		"t":      printer.Printf,
		"plural": printer.Plural,
	})

	b, err := lookupExecute(tpl, data, "layout.html", name)
	if err != nil {
		panic(err)
	}
	return b
}

func lookupExecute(tt *template.Template, data map[string]any,
	names ...string,
) ([]byte, error) {
	for _, name := range names {
		if t := tt.Lookup(name); t != nil {
			var b bytes.Buffer
			if err := t.Execute(&b, data); err != nil {
				return nil, fmt.Errorf("executing %q: %w", name, err)
			}
			return b.Bytes(), nil
		}
	}
	return nil, fmt.Errorf("none of [%v] defined", names)
}
