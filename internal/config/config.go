// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config // import "miniflux.app/v2/internal/config"

// Opts holds parsed configuration options.
var Opts *Options

// Load loads configuration values from a local file (if filename isn't empty)
// and from environment variables after that.
func Load(filename string) (err error) {
	cfg := NewParser()
	if filename != "" {
		Opts, err = cfg.ParseFile(filename)
		return
	}
	Opts, err = cfg.ParseEnvironmentVariables()
	return
}
