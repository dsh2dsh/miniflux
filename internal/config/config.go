// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v4"
)

// opts holds parsed configuration options.
var opts *options

// Load loads configuration values from a local file (if filename isn't empty)
// and from environment variables after that.
func Load(filename string, opts ...LoadOption) error {
	cfg := NewParser()
	for _, fn := range opts {
		if err := fn(cfg); err != nil {
			return err
		}
	}
	return parseEnvFile(cfg, filename)
}

func parseEnvFile(cfg *Parser, filename string) (err error) {
	if filename != "" {
		opts, err = cfg.ParseEnvFile(filename)
		return err
	}
	opts, err = cfg.ParseEnvironmentVariables()
	return err
}

func LoadYAML(filename, envName string) error {
	return Load(envName, WithYAMLFile(filename))
}

type LoadOption func(*Parser) error

func WithYAMLFile(filename string) LoadOption {
	return func(cfg *Parser) error {
		if filename == "" {
			return nil
		}

		b, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("config: reading %q: %w", filename, err)
		}

		if err := yaml.Unmarshal(b, &cfg.opts.yaml); err != nil {
			return fmt.Errorf("config: parse yaml %q: %w", filename, err)
		}
		return nil
	}
}

func WithYAMLString(in string) LoadOption {
	return WithYAMLBytes([]byte(in))
}

func WithYAMLBytes(b []byte) LoadOption {
	return func(cfg *Parser) error {
		if len(b) == 0 {
			return nil
		}

		if err := yaml.Unmarshal(b, &cfg.opts.yaml); err != nil {
			return fmt.Errorf("config: parse yaml from bytes: %w", err)
		}
		return nil
	}
}
