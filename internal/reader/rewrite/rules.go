package rewrite

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/andybalholm/cascadia"

	"miniflux.app/v2/internal/model"
)

type rule struct {
	name     string
	args     []string
	validate bool
}

var _ fmt.Stringer = (*rule)(nil)

func (self *rule) WithValidate(v bool) *rule {
	self.validate = v
	return self
}

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

func (self *rule) applyReplaceContent(entry *model.Entry) error {
	// Format: replace("search-term"|"replace-term")
	if len(self.args) < 2 {
		return errors.New("unable find search and replace terms")
	}

	s, err := replaceCustom(entry.Content, self.args[0], self.args[1])
	if err != nil {
		return err
	}
	entry.Content = s
	return nil
}

func (self *rule) applyReplaceTitle(entry *model.Entry) error {
	// Format: replace_title("search-term"|"replace-term")
	if len(self.args) < 2 {
		return errors.New("unable find search and replace terms")
	}

	s, err := replaceCustom(entry.Title, self.args[0], self.args[1])
	if err != nil {
		return err
	}
	entry.Title = s
	return nil
}

func (self *rule) applyRemove(entry *model.Entry) error {
	// Format: remove("#selector > .element, .another")
	if len(self.args) == 0 {
		return errors.New("selector required")
	}

	selector := self.args[0]
	if self.validate {
		if _, err := cascadia.Compile(selector); err != nil {
			return fmt.Errorf("compile cascadia selector: %w", err)
		}
	}

	s, err := removeCustom(entry.Content, selector)
	if err != nil {
		return err
	}
	entry.Content = s
	return nil
}

func (self *rule) applyBase64Decode(entry *model.Entry) error {
	selector := "body"
	if len(self.args) > 0 {
		selector = self.args[0]
	}

	if self.validate {
		if _, err := cascadia.Compile(selector); err != nil {
			return fmt.Errorf("compile cascadia selector: %w", err)
		}
	}

	entry.Content = applyFuncOnTextContent(entry.Content, selector,
		decodeBase64Content)
	return nil
}
