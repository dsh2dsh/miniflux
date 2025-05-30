// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package encoding // import "miniflux.app/v2/internal/reader/encoding"

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"

	"golang.org/x/net/html/charset"
)

// CharsetReader is used when the XML encoding is specified for the input document.
//
// The document is converted in UTF-8 only if a different encoding is specified
// and the document is not already UTF-8.
//
// Several edge cases could exists:
//
// - Feeds with encoding specified only in Content-Type header and not in XML document
// - Feeds with encoding specified in both places
// - Feeds with encoding specified only in XML document and not in HTTP header
// - Feeds with wrong encoding defined and already in UTF-8
func CharsetReader(charsetLabel string, input io.Reader) (io.Reader, error) {
	buffer, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf(`encoding: unable to read input: %w`, err)
	}

	r := bytes.NewReader(buffer)

	// The document is already UTF-8, do not do anything (avoid double-encoding).
	// That means the specified encoding in XML prolog is wrong.
	if utf8.Valid(buffer) {
		return r, nil
	}

	// Transform document to UTF-8 from the specified encoding in XML prolog.
	reader, err := charset.NewReaderLabel(charsetLabel, r)
	if err != nil {
		return nil, fmt.Errorf("reader/encoding: %w", err)
	}
	return reader, nil
}

// NewCharsetReader returns an io.Reader that converts the content of r to UTF-8.
func NewCharsetReader(r io.Reader, contentType string) (io.Reader, error) {
	buffer, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf(`encoding: unable to read input: %w`, err)
	}

	internalReader := bytes.NewReader(buffer)

	// The document is already UTF-8, do not do anything.
	if utf8.Valid(buffer) {
		return internalReader, nil
	}

	// Transform document to UTF-8 from the specified encoding in Content-Type header.
	// Note that only the first 1024 bytes are used to detect the encoding.
	// If the <meta charset> tag is not found in the first 1024 bytes, charset.DetermineEncoding returns "windows-1252" resulting in encoding issues.
	// See https://html.spec.whatwg.org/multipage/parsing.html#determining-the-character-encoding
	reader, err := charset.NewReader(internalReader, contentType)
	if err != nil {
		return nil, fmt.Errorf("reader/encoding: %w", err)
	}
	return reader, nil
}
