package rewrite

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRules(t *testing.T) {
	rulesText := `add_dynamic_image,replace("article/(.*).svg"|"article/$1.png"),remove(".spam, .ads:not(.keep)")`
	expected := []rule{
		{name: "add_dynamic_image"},
		{name: "replace", args: []string{"article/(.*).svg", "article/$1.png"}},
		{name: "remove", args: []string{".spam, .ads:not(.keep)"}},
	}
	assert.Equal(t, expected, parseRules(rulesText))

	gotStrings := make([]string, len(expected))
	for i := range expected {
		gotStrings[i] = expected[i].String()
	}
	assert.Equal(t, rulesText, strings.Join(gotStrings, ","))

	assert.Equal(t, expected, parseRules(`
add_dynamic_image
replace("article/(.*).svg"|"article/$1.png")
remove(".spam, .ads:not(.keep)")`))
}
