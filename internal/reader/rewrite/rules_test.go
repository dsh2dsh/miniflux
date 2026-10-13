package rewrite

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRules(t *testing.T) {
	rulesText := `add_dynamic_image,replace("article/(.*).svg"|"article/$1.png"),remove(".spam, .ads:not(.keep)")`
	expected := []rule{
		{name: "add_dynamic_image"},
		{name: "replace", args: []string{"article/(.*).svg", "article/$1.png"}},
		{name: "remove", args: []string{".spam, .ads:not(.keep)"}},
	}
	rules, err := parseRules(rulesText)
	require.NoError(t, err)
	assert.Equal(t, expected, rules)

	gotStrings := make([]string, len(expected))
	for i := range expected {
		gotStrings[i] = expected[i].String()
	}
	assert.Equal(t, rulesText, strings.Join(gotStrings, ","))

	rules, err = parseRules(`
add_dynamic_image
replace("article/(.*).svg"|"article/$1.png")
remove(".spam, .ads:not(.keep)")`)
	require.NoError(t, err)
	assert.Equal(t, expected, rules)
}
