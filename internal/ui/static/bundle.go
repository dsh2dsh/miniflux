package static

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"miniflux.app/v2/internal/crypto"
)

func newBundles(ext string) *bundles { return &bundles{ext: ext} }

type bundles struct {
	ext string

	bundles map[string][]byte
	hashes  map[string]string
}

func (self *bundles) Generate(ctx context.Context, fs fs.ReadFileFS,
	manifest []byte,
) error {
	bundleFiles, err := self.unmarshalManifest(manifest)
	if err != nil || len(bundleFiles) == 0 {
		return err
	}

	self.bundles = make(map[string][]byte, len(bundleFiles))
	self.hashes = make(map[string]string, len(bundleFiles))

	for bundleName, srcFiles := range bundleFiles {
		var buffer bytes.Buffer
		zw := gzip.NewWriter(&buffer)

		for _, srcFile := range srcFiles {
			if ctx.Err() != nil {
				return fmt.Errorf("break loop over bundle files (before: %q): %w",
					srcFile, context.Cause(ctx))
			}
			if err := self.copyFile(zw, fs, srcFile); err != nil {
				return err
			}
		}

		if err := zw.Close(); err != nil {
			return fmt.Errorf("closing compressor: %w", err)
		}

		hash := crypto.HashFromBytes(buffer.Bytes())
		filename := bundleName + "." + hash + self.ext

		self.bundles[filename] = buffer.Bytes()
		self.hashes[bundleName] = filename
	}
	return nil
}

func (self *bundles) copyFile(dst io.Writer, fs fs.ReadFileFS, filename string,
) error {
	f, err := fs.Open(filename)
	if err != nil {
		return fmt.Errorf("failed open %q: %w", filename, err)
	}
	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(filename, ".gz") {
		zr, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("gunzip %q: %w", filename, err)
		}
		defer zr.Close()
		r = zr
	}

	if _, err := io.Copy(dst, r); err != nil {
		return fmt.Errorf("compressing %q: %w", filename, err)
	}
	return nil
}

func (self *bundles) unmarshalManifest(manifest []byte) (map[string][]string,
	error,
) {
	var bundles map[string][]string
	if err := json.Unmarshal(manifest, &bundles); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	} else if len(bundles) == 0 {
		return nil, nil
	}
	return bundles, nil
}

func (self *bundles) Bundle(filename string) []byte {
	compressed, ok := self.bundles[filename]
	if !ok {
		return nil
	}
	return compressed
}

func (self *bundles) NameExt(name string) string {
	if filename, ok := self.hashes[name]; ok {
		return filename
	}
	return name + self.ext
}
