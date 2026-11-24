package form

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewIntegrationFormLinktacoVisibility(t *testing.T) {
	scenarios := []struct {
		name     string
		values   url.Values
		expected string
	}{
		{name: "missing field", values: url.Values{}, expected: "PUBLIC"},
		{name: "empty value", values: url.Values{"linktaco_visibility": {""}}, expected: "PUBLIC"},
		{name: "public", values: url.Values{"linktaco_visibility": {"PUBLIC"}}, expected: "PUBLIC"},
		{name: "private", values: url.Values{"linktaco_visibility": {"PRIVATE"}}, expected: "PRIVATE"},
		{name: "invalid value", values: url.Values{"linktaco_visibility": {"UNLISTED"}}, expected: "PUBLIC"},
		{name: "lowercase value", values: url.Values{"linktaco_visibility": {"private"}}, expected: "PUBLIC"},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/integration", strings.NewReader(scenario.values.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			integrationForm := NewIntegrationForm(request)
			assert.Equal(t, scenario.expected, integrationForm.LinktacoVisibility,
				"unexpected visibility")
		})
	}
}
