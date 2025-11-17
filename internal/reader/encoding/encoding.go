// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package encoding // import "miniflux.app/v2/internal/reader/encoding"

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

// NewCharsetReader returns an io.Reader that converts the content of r to
// UTF-8.
//
// Transform document to UTF-8 from the specified encoding in Content-Type
// header. Note that only the first 1024 bytes are used to detect the
// encoding. If the <meta charset> tag is not found in the first 1024 bytes,
// charset.DetermineEncoding returns "windows-1252" resulting in encoding
// issues. See
// https://html.spec.whatwg.org/multipage/parsing.html#determining-the-character-encoding
func NewCharsetReader(r io.Reader, contentType string) (io.Reader, error) {
	preview := make([]byte, 1024)
	n, err := io.ReadFull(r, preview)
	switch {
	case errors.Is(err, io.EOF):
		return r, nil
	case errors.Is(err, io.ErrUnexpectedEOF):
		preview = preview[:n]
		r = bytes.NewReader(preview)
	case err != nil:
		return nil, fmt.Errorf("reader/encoding: read preview: %w", err)
	default:
		r = io.MultiReader(bytes.NewReader(preview), r)
	}

	e, name, certain := charset.DetermineEncoding(preview, contentType)
	if e == encoding.Nop || name == "utf-8" {
		return r, nil
	}

	if certain || name != "windows-1252" || !utf8.Valid(preview) {
		return transform.NewReader(r, e.NewDecoder()), nil
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf(
			"reader/encoding: read all for detecting utf8: %w", err)
	}

	if utf8.Valid(b) {
		return bytes.NewReader(b), nil
	}
	return transform.NewReader(bytes.NewReader(b), e.NewDecoder()), nil
}
