// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package template // import "miniflux.app/v2/internal/template"

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"strings"
	"time"

	"miniflux.app/v2/internal/http/mux"
	"miniflux.app/v2/internal/locale"
)

//go:embed templates/common/*.html
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
	var commonTemplateContents strings.Builder

	dirEntries, err := commonTemplateFiles.ReadDir("templates/common")
	if err != nil {
		return fmt.Errorf("template: filed read templates/common/: %w", err)
	}

	for _, dirEntry := range dirEntries {
		fullName := "templates/common/" + dirEntry.Name()
		fileData, err := commonTemplateFiles.ReadFile(fullName)
		if err != nil {
			return fmt.Errorf("template: failed read %q: %w", fullName, err)
		}
		commonTemplateContents.Write(fileData)
	}

	dirEntries, err = viewTemplateFiles.ReadDir("templates/views")
	if err != nil {
		return fmt.Errorf("template: failed read templates/views/: %w", err)
	}

	for _, dirEntry := range dirEntries {
		templateName := dirEntry.Name()
		fullName := "templates/views/" + templateName
		fileData, err := viewTemplateFiles.ReadFile(fullName)
		if err != nil {
			return fmt.Errorf("template: failed read %q: %w", fullName, err)
		}

		var templateContents strings.Builder
		templateContents.WriteString(commonTemplateContents.String())
		templateContents.Write(fileData)

		slog.Debug("Parsing template",
			slog.String("template_name", templateName),
		)

		e.templates[templateName] = template.Must(template.New("main").Funcs(e.funcMap.Map()).Parse(templateContents.String()))
	}

	dirEntries, err = standaloneTemplateFiles.ReadDir("templates/standalone")
	if err != nil {
		return fmt.Errorf("template: failed read templates/standalone/: %w", err)
	}

	for _, dirEntry := range dirEntries {
		templateName := dirEntry.Name()
		fileData, err := standaloneTemplateFiles.ReadFile("templates/standalone/" + dirEntry.Name())
		if err != nil {
			return err
		}

		slog.Debug("Parsing template",
			slog.String("template_name", templateName),
		)
		e.templates[templateName] = template.Must(template.New("base").Funcs(e.funcMap.Map()).Parse(string(fileData)))
	}
	return nil
}

// Render process a template.
func (e *Engine) Render(name string, data map[string]any) []byte {
	tpl, ok := e.templates[name]
	if !ok {
		panic("This template does not exists: " + name)
	}

	printer := locale.NewPrinter(data["language"].(string))

	// Functions that need to be declared at runtime.
	tpl.Funcs(template.FuncMap{
		"elapsed": func(timezone string, t time.Time) string {
			return elapsedTime(printer, timezone, t)
		},
		"t":      printer.Printf,
		"plural": printer.Plural,
	})

	var b bytes.Buffer
	err := tpl.ExecuteTemplate(&b, "base", data)
	if err != nil {
		panic(err)
	}

	return b.Bytes()
}
