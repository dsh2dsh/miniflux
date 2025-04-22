// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"miniflux.app/v2/internal/config"
)

var healthCmd = cobra.Command{
	Use:   "healthcheck auto|endpoint",
	Short: `Perform a health check on the given endpoint`,

	Long: `Perform a health check on the given endpoint.

The value "auto" try to guess the health check endpoint.
`,

	Example: `
$ miniflux healthcheck http://127.0.0.1:8080
`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return doHealthCheck(args[0])
	},
}

func doHealthCheck(healthCheckEndpoint string) error {
	if healthCheckEndpoint == "auto" {
		healthCheckEndpoint = "http://" + config.Opts.ListenAddr() +
			config.Opts.BasePath() + "/healthcheck"
	}

	slog.Debug("Executing health check request",
		slog.String("endpoint", healthCheckEndpoint))

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(healthCheckEndpoint)
	if err != nil {
		return fmt.Errorf(`health check failure: %w`, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(`health check failed with status code %d`, resp.StatusCode)
	}
	slog.Debug(`Health check is passing`)
	return nil
}
