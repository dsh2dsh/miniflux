// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package static // import "miniflux.app/v2/internal/ui/static"

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"

	"miniflux.app/v2/internal/crypto"
)

// Static assets.
var (
	StylesheetBundleChecksums map[string]string
	StylesheetBundles         map[string][]byte
	JavascriptBundleChecksums map[string]string
	JavascriptBundles         map[string][]byte
)

//go:embed bin/*
var binaryFiles embed.FS

//go:embed css/*.css
var stylesheetFiles embed.FS

//go:embed css.json
var stylesheetManifest []byte

//go:embed js/*.js
var javascriptFiles embed.FS

//go:embed js.json
var javascriptManifest []byte

var binaryFileChecksums map[string]string

// CalculateBinaryFileChecksums generates hash of embed binary files.
func CalculateBinaryFileChecksums(ctx context.Context) error {
	slog.Info("calculate binary file hashes")
	dirEntries, err := binaryFiles.ReadDir("bin")
	if err != nil {
		return fmt.Errorf("ui/static: failed read bin/: %w", err)
	}

	binaryFileChecksums = make(map[string]string, len(dirEntries))
	for _, dirEntry := range dirEntries {
		if ctx.Err() != nil {
			return fmt.Errorf("ui/static: break loop over binary files: %w",
				context.Cause(ctx))
		}
		data, err := LoadBinaryFile(dirEntry.Name())
		if err != nil {
			return err
		}
		binaryFileChecksums[dirEntry.Name()] = crypto.HashFromBytes(data)
	}
	return nil
}

// LoadBinaryFile loads an embed binary file.
func LoadBinaryFile(filename string) ([]byte, error) {
	fullName := `bin/` + filename
	b, err := binaryFiles.ReadFile(fullName)
	if err != nil {
		return nil, fmt.Errorf("ui/static: failed read %q: %w", fullName, err)
	}
	return b, nil
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
	var bundles map[string][]string
	if err := json.Unmarshal(stylesheetManifest, &bundles); err != nil {
		return fmt.Errorf("ui/static: unmarshal css.json: %w", err)
	}

	StylesheetBundles = make(map[string][]byte, len(bundles))
	StylesheetBundleChecksums = make(map[string]string, len(bundles))

	for bundle, srcFiles := range bundles {
		var buffer bytes.Buffer
		for _, srcFile := range srcFiles {
			if ctx.Err() != nil {
				return fmt.Errorf(
					"ui/static: break loop over css files(before: %q): %w",
					srcFile, context.Cause(ctx))
			}
			fileData, err := stylesheetFiles.ReadFile(srcFile)
			if err != nil {
				return fmt.Errorf("ui/static: failed read %q: %w", srcFile, err)
			}
			buffer.Write(fileData)
		}

		StylesheetBundles[bundle] = buffer.Bytes()
		StylesheetBundleChecksums[bundle] = crypto.HashFromBytes(buffer.Bytes())
	}
	return nil
}

// GenerateJavascriptBundles creates JS bundles.
func GenerateJavascriptBundles(ctx context.Context) error {
	slog.Info("generate js bundles")
	var bundles map[string][]string
	if err := json.Unmarshal(javascriptManifest, &bundles); err != nil {
		return fmt.Errorf("ui/static: unmarshal js.json: %w", err)
	}

	prefixes := map[string]string{"app": "(function(){'use strict';"}
	suffixes := map[string]string{"app": "})();"}

	JavascriptBundles = make(map[string][]byte, len(bundles))
	JavascriptBundleChecksums = make(map[string]string, len(bundles))

	for bundle, srcFiles := range bundles {
		var buffer bytes.Buffer
		if prefix, ok := prefixes[bundle]; ok {
			buffer.WriteString(prefix)
		}

		for _, srcFile := range srcFiles {
			if ctx.Err() != nil {
				return fmt.Errorf(
					"ui/static: break loop over js files(before: %q): %w",
					srcFile, context.Cause(ctx))
			}
			fileData, err := javascriptFiles.ReadFile(srcFile)
			if err != nil {
				return fmt.Errorf("ui/static: failed read %q: %w", srcFile, err)
			}
			buffer.Write(fileData)
		}

		if suffix, ok := suffixes[bundle]; ok {
			buffer.WriteString(suffix)
		}

		JavascriptBundles[bundle] = buffer.Bytes()
		JavascriptBundleChecksums[bundle] = crypto.HashFromBytes(buffer.Bytes())
	}
	return nil
}
