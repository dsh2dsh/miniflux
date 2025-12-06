package rewrite

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"miniflux.app/v2/internal/logging"
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

func (self *rule) applyReplaceContent(ctx context.Context, entry *model.Entry) {
	// Format: replace("search-term"|"replace-term")
	if len(self.args) < 2 {
		logging.FromContext(ctx).Warn(
			"Cannot find search and replace terms for replace rule",
			slog.Any("rule", self),
			slog.String("entry_url", entry.URL),
		)
	}
	entry.Content = replaceCustom(entry.Content, self.args[0], self.args[1])
}

func (self *rule) applyReplaceTitle(ctx context.Context, entry *model.Entry) {
	// Format: replace_title("search-term"|"replace-term")
	if len(self.args) < 2 {
		logging.FromContext(ctx).Warn(
			"Cannot find search and replace terms for replace_title rule",
			slog.Any("rule", self), slog.String("entry_url", entry.URL))
	}
	entry.Title = replaceCustom(entry.Title, self.args[0], self.args[1])
}

func (self *rule) applyRemove(ctx context.Context, entry *model.Entry) {
	// Format: remove("#selector > .element, .another")
	if len(self.args) == 0 {
		logging.FromContext(ctx).Warn("Cannot find selector for remove rule",
			slog.Any("rule", self), slog.String("entry_url", entry.URL))
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
