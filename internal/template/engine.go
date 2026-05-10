// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package template // import "miniflux.app/v2/internal/template"

import (
	"embed"
	"fmt"
	"html/template"
	"log/slog"

	"miniflux.app/v2/internal/http/mux"
)

var (
	//go:embed templates/common/*.html templates/common/*.svg
	commonTemplateFiles embed.FS

	//go:embed templates/views/*.html
	viewTemplateFiles embed.FS

	//go:embed templates/standalone/*.html
	standaloneTemplateFiles embed.FS
)

type HTML = template.HTML

// Engine handles the templating system.
type Engine struct {
	router    *mux.ServeMux
	templates map[string]*template.Template
	funcMap   *funcMap
}

// NewEngine returns a new template engine.
func NewEngine(router *mux.ServeMux) *Engine {
	return &Engine{
		router:    router,
		templates: make(map[string]*template.Template),
		funcMap:   newFuncMap(router),
	}
}

func (self *Engine) Router() *mux.ServeMux { return self.router }

// ParseTemplates parses template files embed into the application.
func (self *Engine) ParseTemplates() error {
	funcMap := self.funcMap.Map()
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
		self.templates[dirEntry.Name()] = template.Must(commonTemplatesClone.ParseFS(viewTemplateFiles, fullName))
	}

	dirEntries, err = standaloneTemplateFiles.ReadDir("templates/standalone")
	if err != nil {
		return fmt.Errorf("template: failed read templates/standalone/: %w", err)
	}
	for _, dirEntry := range dirEntries {
		fullName := "templates/standalone/" + dirEntry.Name()
		slog.Debug("Parsing template", slog.String("template_name", fullName))
		self.templates[dirEntry.Name()] = template.Must(template.New(dirEntry.Name()).Funcs(funcMap).ParseFS(standaloneTemplateFiles, fullName))
	}
	return nil
}

// Render process a template.
func (self *Engine) Render(name string, data map[string]any, opts ...Option,
) []byte {
	parsedTemplate, ok := self.templates[name]
	if !ok {
		panic("This template does not exists: " + name)
	}

	t := Template{Template: template.Must(parsedTemplate.Clone())}
	for _, fn := range opts {
		fn(&t)
	}

	b, err := t.LookupExecute(data, "layout.html", name)
	if err != nil {
		panic(err)
	}
	return b
}
