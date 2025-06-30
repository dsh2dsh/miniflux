// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config // import "miniflux.app/v2/internal/config"

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Parser handles configuration parsing.
type Parser struct {
	opts *Options
}

// NewParser returns a new Parser.
func NewParser() *Parser { return &Parser{opts: NewOptions()} }

// ParseEnvironmentVariables loads configuration values from environment
// variables.
func (p *Parser) ParseEnvironmentVariables() (*Options, error) {
	if err := env.Parse(p.env()); err != nil {
		return nil, fmt.Errorf("config: failed parse env vars: %w", err)
	} else if err := p.opts.init(); err != nil {
		return nil, fmt.Errorf("failed parse env vars: %w", err)
	}
	return p.opts, nil
}

func (p *Parser) env() *EnvOptions { return &p.opts.env }

// ParseEnvFile loads configuration values from a local file and from
// environment variables after that.
func (p *Parser) ParseEnvFile(filename string) (*Options, error) {
	envMap, err := godotenv.Read(filename)
	if err != nil {
		return nil, fmt.Errorf("config: failed parse %q: %w", filename, err)
	}

	err = env.ParseWithOptions(p.env(), env.Options{Environment: envMap})
	if err != nil {
		return nil, fmt.Errorf("config: failed parse %q: %w", filename, err)
	}
	return p.ParseEnvironmentVariables()
}

func parseBaseURL(value string) (string, string, string, error) {
	if value == "" {
		return defaultBaseURL, defaultBaseURL, "", nil
	}

	if value[len(value)-1:] == "/" {
		value = value[:len(value)-1]
	}

	parsedURL, err := url.Parse(value)
	if err != nil {
		return "", "", "", fmt.Errorf("config: invalid BASE_URL: %w", err)
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "https" && scheme != "http" {
		return "", "", "", errors.New("config: invalid BASE_URL: scheme must be http or https")
	}

	basePath := parsedURL.Path
	parsedURL.Path = ""
	return value, parsedURL.String(), basePath, nil
}
