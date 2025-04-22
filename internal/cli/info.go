// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"miniflux.app/v2/internal/version"
)

var infoCmd = cobra.Command{
	Use:   "info",
	Short: "Show build information",
	Args:  cobra.ExactArgs(0),
	Run:   func(cmd *cobra.Command, args []string) { info() },
}

func info() {
	fmt.Println("Version:", version.Version)
	fmt.Println("Commit:", version.Commit)
	fmt.Println("Build Date:", version.BuildDate)
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("Compiler:", runtime.Compiler)
	fmt.Println("Arch:", runtime.GOARCH)
	fmt.Println("OS:", runtime.GOOS)
}
