// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package template // import "miniflux.app/v2/internal/template"

import (
	"bytes"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"

	"miniflux.app/v2/internal/http/mux"
)

// TestRenderConcurrency renders the same template concurrently in different
// languages. Because Render binds per-request, language-specific functions
// ("t", "plural", "elapsed") onto the template, doing so on a shared template
// while other requests execute it corrupts the output: a request can be served
// another request's language. Each concurrent render must match the output of
// the equivalent sequential render for its language.
func TestRenderConcurrency(t *testing.T) {
	m := mux.New()
	m.NameHandleFunc("/unread",
		func(http.ResponseWriter, *http.Request) {}, "unread")
	engine := NewEngine(m)
	engine.ParseTemplates()

	languages := [...]string{
		"en_US",
		"fr_FR",
		"de_DE",
		"es_ES",
		"pt_BR",
		"ru_RU",
		"zh_CN",
		"it_IT",
	}

	data := map[string]any{
		"theme": "system_serif",
	}

	// Establish the expected output for each language sequentially.
	expected := make(map[string][]byte, len(languages))
	for _, language := range languages {
		expected[language] = engine.Render("offline.html", data,
			WithLanguage(language))
	}

	const iterations = 300
	var wg sync.WaitGroup
	var mismatches atomic.Int64

	for i := range iterations {
		wg.Go(func() {
			language := languages[i%len(languages)]
			got := engine.Render("offline.html", data, WithLanguage(language))
			if !bytes.Equal(got, expected[language]) {
				mismatches.Add(1)
			}
		})
	}
	wg.Wait()

	n := mismatches.Load()
	assert.Zero(t, n,
		"concurrent Render produced wrong output for %d/%d requests (wrong-language translations)",
		n, iterations)
}
