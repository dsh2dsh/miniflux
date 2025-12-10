package rewrite

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"miniflux.app/v2/internal/model"
)

type rule struct {
	name string
	args []string
}

var _ fmt.Stringer = (*rule)(nil)

func (self *rule) String() string {
	if len(self.args) == 0 {
		return self.name
	}

	var sb strings.Builder
	sb.WriteString(self.name)
	sb.WriteByte('(')
	for i, s := range self.args {
		if i > 0 {
			sb.WriteByte('|')
		}
		sb.WriteString(strconv.Quote(s))
	}
	sb.WriteByte(')')
	return sb.String()
}

func (self *rule) applyReplaceContent(log *slog.Logger, entry *model.Entry) {
	// Format: replace("search-term"|"replace-term")
	if len(self.args) < 2 {
		log.Warn("Cannot find search and replace terms for replace rule")
	}
	entry.Content = replaceCustom(entry.Content, self.args[0], self.args[1])
}

func (self *rule) applyReplaceTitle(log *slog.Logger, entry *model.Entry) {
	// Format: replace_title("search-term"|"replace-term")
	if len(self.args) < 2 {
		log.Warn("Cannot find search and replace terms for replace_title rule")
	}
	entry.Title = replaceCustom(entry.Title, self.args[0], self.args[1])
}

func (self *rule) applyRemove(log *slog.Logger, entry *model.Entry) {
	// Format: remove("#selector > .element, .another")
	if len(self.args) == 0 {
		log.Warn("Cannot find selector for remove rule")
		return
	}
	entry.Content = removeCustom(entry.Content, self.args[0])
}

func (self *rule) applyBase64Decode(entry *model.Entry) {
	selector := "body"
	if len(self.args) > 0 {
		selector = self.args[0]
	}
	entry.Content = applyFuncOnTextContent(entry.Content, selector,
		decodeBase64Content)
}
