// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config // import "miniflux.app/v2/internal/config"

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Parser handles configuration parsing.
type Parser struct {
	opts *options
}

// NewParser returns a new Parser.
func NewParser() *Parser { return &Parser{opts: NewOptions()} }

// ParseEnvironmentVariables loads configuration values from environment
// variables.
func (p *Parser) ParseEnvironmentVariables() (*options, error) {
	if err := env.Parse(p.env()); err != nil {
		return nil, fmt.Errorf("config: failed parse env vars: %w", err)
	} else if err := p.opts.init(); err != nil {
		return nil, fmt.Errorf("failed parse env vars: %w", err)
	}
	return p.opts, nil
}

func (p *Parser) env() *envOptions { return &p.opts.env }

// ParseEnvFile loads configuration values from a local file and from
// environment variables after that.
func (p *Parser) ParseEnvFile(filename string) (*options, error) {
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
