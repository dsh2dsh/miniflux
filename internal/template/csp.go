package template

import (
	"crypto/rand"
	"strings"

	"miniflux.app/v2/internal/model"
)

type contentSecurityPolicy struct {
	nonce string
}

func newContentSecurityPolicy() *contentSecurityPolicy {
	return &contentSecurityPolicy{nonce: rand.Text()}
}

func (self *contentSecurityPolicy) Nonce() string { return self.nonce }

func (self *contentSecurityPolicy) Content(user *model.User) string {
	policies := self.policies()
	if user != nil {
		self.userPolicies(user, policies)
	}

	var policy strings.Builder
	for key, value := range policies {
		if policy.Len() != 0 {
			policy.WriteByte(' ')
		}
		policy.WriteString(key + " " + value + ";")
	}
	return policy.String()
}

func (self *contentSecurityPolicy) policies() map[string]string {
	nonce := self.Nonce()
	return map[string]string{
		"default-src":  "'none'",
		"frame-src":    "*",
		"img-src":      "* data:",
		"manifest-src": "'self'",
		"media-src":    "*",
		"script-src":   "'nonce-" + nonce + "' 'strict-dynamic'",
		"style-src":    "'nonce-" + nonce + "'",
		"connect-src":  "'self'",

		// "require-trusted-types-for": "'script'",
		// "trusted-types":             "html url",
	}
}

func (self *contentSecurityPolicy) userPolicies(user *model.User,
	policies map[string]string,
) {
	if user.ExternalFontHosts == "" {
		return
	}

	policies["font-src"] = user.ExternalFontHosts
	if user.Stylesheet != "" {
		policies["style-src"] += " " + user.ExternalFontHosts
	}
}
