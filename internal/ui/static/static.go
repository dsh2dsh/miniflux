// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package static // import "miniflux.app/v2/internal/ui/static"

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"path"
	"strconv"
	"strings"

	"github.com/cespare/xxhash/v2"

	"miniflux.app/v2/internal/crypto"
)

const (
	binDir = "bin"
	cssExt = ".css"
	jsExt  = ".js"
)

// Static assets.
var (
	stylesheetBundles map[string][]byte
	stylesheetHashes  map[string]string

	javascriptBundles map[string][]byte
	javascriptHashes  map[string]string

	hashedNames   = make(map[string]string)
	unhashedNames = make(map[string]string)
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
	var bundles map[string][]string
	if err := json.Unmarshal(stylesheetManifest, &bundles); err != nil {
		return fmt.Errorf("ui/static: unmarshal css.json: %w", err)
	}

	stylesheetBundles = make(map[string][]byte, len(bundles))
	stylesheetHashes = make(map[string]string, len(bundles))

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

		hash := crypto.HashFromBytes(buffer.Bytes())
		filename := bundle + "." + hash + cssExt

		stylesheetBundles[filename] = buffer.Bytes()
		stylesheetHashes[bundle] = filename
	}
	return nil
}

func StylesheetBundle(filename, hashExt string) []byte {
	b, ok := stylesheetBundles[filename]
	if !ok {
		return nil
	}
	return b
}

func StylesheetNameExt(name string) string {
	if filename, ok := stylesheetHashes[name]; ok {
		return filename
	}
	return name + cssExt
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

	javascriptBundles = make(map[string][]byte, len(bundles))
	javascriptHashes = make(map[string]string, len(bundles))

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

		hash := crypto.HashFromBytes(buffer.Bytes())
		filename := bundle + "." + hash + jsExt

		javascriptBundles[filename] = buffer.Bytes()
		javascriptHashes[bundle] = filename
	}
	return nil
}

func JavascriptBundle(filename string) []byte {
	b, ok := javascriptBundles[filename]
	if !ok {
		return nil
	}
	return b
}

func JavascriptNameExt(name string) string {
	if filename, ok := javascriptHashes[name]; ok {
		return filename
	}
	return name + jsExt
}
