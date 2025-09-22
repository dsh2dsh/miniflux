// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config // import "miniflux.app/v2/internal/config"

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v4"
)

// Opts holds parsed configuration options.
var Opts *Options

// Load loads configuration values from a local file (if filename isn't empty)
// and from environment variables after that.
func Load(filename string) error {
	return parseEnvFile(NewParser(), filename)
}

func parseEnvFile(cfg *Parser, filename string) (err error) {
	if filename != "" {
		Opts, err = cfg.ParseEnvFile(filename)
		return err
	}
	Opts, err = cfg.ParseEnvironmentVariables()
	return err
}

func LoadYAML(filename, envName string) error {
	cfg := NewParser()
	if filename == "" {
		return parseEnvFile(cfg, envName)
	}

	b, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("config: reading %q: %w", filename, err)
	}

	if err := yaml.Unmarshal(b, cfg.opts); err != nil {
		return fmt.Errorf("config: parse yaml %q: %w", filename, err)
	}
	return parseEnvFile(cfg, envName)
}
