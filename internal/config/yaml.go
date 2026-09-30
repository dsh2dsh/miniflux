package config

import (
	"fmt"
	"net/url"
	"strings"

	"go.yaml.in/yaml/v4"
)

type yamlOptions struct {
	HostLimits   map[string]HostLimits `yaml:"host_limits" validate:"dive,keys,required,endkeys,required"`
	PrivateHosts map[string][]string   `yaml:"privateHosts" validate:"dive,keys,required,ip|hostname_port,endkeys,dive,required,url"`
	Proxies      []*Proxy              `yaml:"proxies" validate:"dive,required"`
}

type HostLimits struct {
	Connections int64   `yaml:"connections" validate:"omitempty,min=0"`
	Rate        float64 `yaml:"rate" validate:"omitempty,min=0"`
}

type Proxy struct {
	Id        string   `yaml:"id" validate:"required"`
	Name      string   `yaml:"name" validate:"required"`
	ParsedURL *yamlURL `yaml:"url" validate:"required"`
}

func NewProxy(u *url.URL) *Proxy {
	return &Proxy{ParsedURL: (*yamlURL)(u)}
}

type yamlURL url.URL

func (self *yamlURL) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return fmt.Errorf("unmarshal URL string: %w", err)
	} else if strings.TrimSpace(s) == "" {
		return nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return fmt.Errorf("parse URL string: %w", err)
	}

	*self = yamlURL(*u)
	return nil
}

func (self *Proxy) URL() *url.URL { return (*url.URL)(self.ParsedURL) }

func (self *Proxy) Redacted() string {
	if u := self.URL(); u != nil {
		return u.Redacted()
	}
	return ""
}
