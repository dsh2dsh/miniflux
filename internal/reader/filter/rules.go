package filter

import (
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"

	"miniflux.app/v2/internal/model"
)

func New(s string) (*Filter, error) {
	return NewCombinedFilter(s)
}

func NewCombinedFilter(humanRules ...string) (*Filter, error) {
	var size int
	for _, s := range humanRules {
		if strings.TrimSpace(s) == "" {
			continue
		}
		size += strings.Count(s, "\n") + 1
	}
	rules := make([]*Rule, 0, size)

	for j, s := range humanRules {
		if strings.TrimSpace(s) == "" {
			continue
		}
		var i int
		for line := range strings.SplitSeq(s, "\n") {
			i++
			line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
			if line == "" {
				continue
			}
			rule, err := NewRule(line)
			if err != nil {
				return nil, fmt.Errorf("parse rule set=%d line=%d: %w", j+1, i, err)
			}
			rules = append(rules, rule)
		}
	}
	return &Filter{rules: rules}, nil
}

type Filter struct {
	rules  []*Rule
	logger *slog.Logger
}

func (self *Filter) WithLogger(l *slog.Logger) *Filter {
	self.logger = l
	return self
}

func (self *Filter) Concat(filters ...*Filter) *Filter {
	size := len(self.rules)
	for _, f := range filters {
		size += len(f.rules)
	}

	rules := slices.Grow[[]*Rule](nil, size)
	rules = append(rules, self.rules...)
	for _, f := range filters {
		rules = append(rules, f.rules...)
	}
	return &Filter{rules: rules}
}

func (self *Filter) Match(entry *model.Entry) bool {
	return slices.ContainsFunc(self.rules, func(rule *Rule) bool {
		if rule.Match(entry) {
			self.logMatch(entry, rule)
			return true
		}
		return false
	})
}

func (self *Filter) logMatch(entry *model.Entry, rule *Rule) {
	if self.logger == nil {
		return
	}
	self.logger.Debug("Filtering entry based on rule",
		slog.String("entry_url", entry.URL),
		slog.String("filter_rule", rule.filter))
}

func (self *Filter) Allow(entry *model.Entry) bool {
	if len(self.rules) == 0 {
		return true
	}
	return self.Match(entry)
}

type Rule struct {
	field  string
	filter string

	re *regexp.Regexp
}

func NewRule(s string) (*Rule, error) {
	field, filter, found := strings.Cut(strings.TrimSpace(s), "=")
	if !found {
		return nil, fmt.Errorf("unexpected rule format %q", s)
	}

	field = strings.ToLower(strings.TrimSpace(field))
	if field == "" {
		return nil, fmt.Errorf("empty field in %q", s)
	}

	filter = strings.TrimSpace(filter)
	if filter == "" {
		return nil, fmt.Errorf("empty filter in %q", s)
	}

	self := &Rule{field: field, filter: filter}
	return self, self.init()
}

func (self *Rule) init() error {
	switch self.field {
	case
		"entryauthor", "author",
		"entrycommentsurl", "commentsurl",
		"entrycontent", "content",
		"entrytag", "tag",
		"entrytitle", "title",
		"entryurl", "url",
		"any":

		re, err := regexp.Compile(self.filter)
		if err != nil {
			return fmt.Errorf("compile rule regexp %q: %w", self.filter, err)
		}
		self.re = re
	case "entrydate", "date":
	default:
		return fmt.Errorf("unknown field %q", self.field)
	}
	return nil
}

func (self *Rule) Match(entry *model.Entry) bool {
	switch self.field {
	case "entryauthor", "author":
		return self.re.MatchString(entry.Author)
	case "entrycommentsurl", "commentsurl":
		return self.re.MatchString(entry.CommentsURL)
	case "entrycontent", "content":
		return self.re.MatchString(entry.Content)
	case "entrydate", "date":
		return matchDatePattern(self.filter, entry.Date)
	case "entrytag", "tag":
		return slices.ContainsFunc(entry.Tags, func(tag string) bool {
			return self.re.MatchString(tag)
		})
	case "entrytitle", "title":
		return self.re.MatchString(entry.Title)
	case "entryurl", "url":
		return self.re.MatchString(entry.URL)
	}

	return self.re.MatchString(entry.Author) ||
		self.re.MatchString(entry.CommentsURL) ||
		self.re.MatchString(entry.Content) ||
		self.re.MatchString(entry.Title) ||
		self.re.MatchString(entry.URL) ||
		slices.ContainsFunc(entry.Tags, func(tag string) bool {
			return self.re.MatchString(tag)
		})
}
