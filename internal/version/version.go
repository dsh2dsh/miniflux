// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package version // import "miniflux.app/v2/internal/version"

import "strings"

const (
	devVersion = "Development Version"
	repoURL    = "https://github.com/dsh2dsh/miniflux"
)

// Variables populated at build time when using LD_FLAGS.
var (
	Commit    = "Unknown (built outside VCS)"
	BuildDate = "Unknown (built outside VCS)"
	Version   = devVersion
)

type Info struct{}

func New() Info { return Info{} }

func (Info) Commit() string { return Commit }

func (self Info) CommitURL() string {
	if strings.HasPrefix(self.Commit(), "Unknown ") {
		return ""
	}
	return repoURL + "/commit/" + self.Commit()
}

func (Info) BuildDate() string { return BuildDate }

func (Info) Version() string { return Version }

func (self Info) VersionURL() string {
	if self.Version() == devVersion {
		return ""
	}

	tag, commits, found := strings.Cut(self.Version(), "-")
	if !found {
		return repoURL + "/releases/tag/v" + tag
	}

	_, hash, found := strings.Cut(commits, "-g")
	if !found {
		return ""
	}
	return repoURL + "/compare/v" + tag + "..." + hash
}
