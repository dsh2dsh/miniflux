// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package encoding // import "miniflux.app/v2/internal/reader/encoding"

import (
	"errors"
	"fmt"
	"io"

	"golang.org/x/net/html/charset"
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
	reader, err := charset.NewReader(r, contentType)
	switch {
	case errors.Is(err, io.EOF):
		return r, nil
	case err != nil:
		return nil, fmt.Errorf(
			"reader/encoding: new charset reader with contentType=%q: %w",
			contentType, err)
	}
	return reader, nil
}
