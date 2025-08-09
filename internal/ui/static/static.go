// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package static // import "miniflux.app/v2/internal/ui/static"

import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"path"
	"strconv"
	"strings"

	"github.com/cespare/xxhash/v2"
)

const binDir = "bin"

var (
	javascripts *bundles
	stylesheets *bundles

	hashedNames   = make(map[string]string)
	unhashedNames = make(map[string]string)

	//go:embed bin/*
	binaryFiles embed.FS

	//go:embed css/*.css
	stylesheetFiles embed.FS

	//go:embed css.json
	stylesheetManifest []byte

	//go:embed js/*.js
	javascriptFiles embed.FS

	//go:embed js.json
	javascriptManifest []byte

	binaryFileChecksums map[string]string
)

// CalculateBinaryFileChecksums generates hash of embed binary files.
func CalculateBinaryFileChecksums(ctx context.Context) error {
	slog.Info("calculate binary file hashes")
	dirEntries, err := binaryFiles.ReadDir(binDir)
	if err != nil {
		return fmt.Errorf("ui/static: failed read bin/: %w", err)
	}

	binaryFileChecksums = make(map[string]string, len(dirEntries))
	d := xxhash.New()

	for _, dirEntry := range dirEntries {
		if ctx.Err() != nil {
			return fmt.Errorf("ui/static: break loop over binary files: %w",
				context.Cause(ctx))
		}

		filename := dirEntry.Name()
		f, err := OpenBinaryFile(filename)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(d, f)
		if err != nil {
			return fmt.Errorf("ui/static: copy to digest %q: %w", filename, err)
		}

		hashFileName(filename, strconv.FormatUint(d.Sum64(), 16))
		d.Reset()
	}
	return nil
}

func hashFileName(filename, hash string) string {
	binaryFileChecksums[filename] = hash

	before, after, found := strings.Cut(filename, ".")
	hashedName := before + "-" + hash
	if found {
		hashedName += "." + after
	}

	hashedNames[filename] = hashedName
	unhashedNames[hashedName] = filename
	return hashedName
}

func OpenBinaryFile(filename string) (fs.File, error) {
	if s, ok := unhashedNames[filename]; ok {
		filename = s
	}

	fullName := path.Join(binDir, filename)
	f, err := binaryFiles.Open(fullName)
	if err != nil {
		return nil, fmt.Errorf("ui/static: failed open %q: %w", fullName, err)
	}
	return f, nil
}

func BinaryFileName(filename string) string {
	if hashedName, ok := hashedNames[filename]; ok {
		return hashedName
	}
	return filename
}

// GetBinaryFileChecksum returns a binary file checksum.
func GetBinaryFileChecksum(filename string) (string, error) {
	if _, found := binaryFileChecksums[filename]; !found {
		return "", fmt.Errorf(`static: unable to find checksum for %q`, filename)
	}
	return binaryFileChecksums[filename], nil
}

// GenerateStylesheetsBundles creates CSS bundles.
func GenerateStylesheetsBundles(ctx context.Context) error {
	slog.Info("generate css bundles")
	stylesheets = newBundles(".css")
	err := stylesheets.Generate(ctx, stylesheetFiles, stylesheetManifest)
	if err != nil {
		return fmt.Errorf("ui/static: generate bundles from css.json: %w", err)
	}
	return nil
}

func StylesheetBundle(filename string) []byte {
	return stylesheets.Bundle(filename)
}

func StylesheetNameExt(name string) string {
	return stylesheets.NameExt(name)
}

// GenerateJavascriptBundles creates JS bundles.
func GenerateJavascriptBundles(ctx context.Context) error {
	slog.Info("generate js bundles")
	javascripts = newBundles(".js")
	err := javascripts.Generate(ctx, javascriptFiles, javascriptManifest)
	if err != nil {
		return fmt.Errorf("ui/static: generate bundles from js.json: %w", err)
	}
	return nil
}

func JavascriptBundle(filename string) []byte {
	return javascripts.Bundle(filename)
}

func JavascriptNameExt(name string) string {
	return javascripts.NameExt(name)
}
