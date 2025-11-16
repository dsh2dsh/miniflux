// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package encoding // import "miniflux.app/v2/internal/reader/encoding"

import (
	"bytes"
	"io"
	"os"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReaderWithUTF8Document(t *testing.T) {
	f, err := os.Open("testdata/utf8.html")
	require.NoError(t, err)

	reader, err := NewCharsetReader(f, "text/html; charset=UTF-8")
	require.NoError(t, err)

	data, err := io.ReadAll(reader)
	require.NoError(t, err)

	require.True(t, utf8.Valid(data), "Data is not valid UTF-8")

	expected := "Café"
	assert.True(t, bytes.Contains(data, []byte(expected)),
		"Data does not contain expected unicode string: %s", expected)
}

func TestNewReaderWithUTF8DocumentAndNoContentEncoding(t *testing.T) {
	f, err := os.Open("testdata/utf8.html")
	require.NoError(t, err)

	reader, err := NewCharsetReader(f, "text/html")
	require.NoError(t, err)

	data, err := io.ReadAll(reader)
	require.NoError(t, err)

	require.True(t, utf8.Valid(data), "Data is not valid UTF-8")

	expected := "Café"
	if !bytes.Contains(data, []byte(expected)) {
		t.Fatalf("Data does not contain expected unicode string: %s", expected)
	}
}

func TestNewReaderWithISO88591Document(t *testing.T) {
	f, err := os.Open("testdata/iso-8859-1.xml")
	require.NoError(t, err)

	reader, err := NewCharsetReader(f, "text/html; charset=ISO-8859-1")
	require.NoError(t, err)

	data, err := io.ReadAll(reader)
	require.NoError(t, err)

	require.True(t, utf8.Valid(data), "Data is not valid UTF-8")

	expected := "Café"
	assert.True(t, bytes.Contains(data, []byte(expected)),
		"Data does not contain expected unicode string: %s", expected)
}

func TestNewReaderWithISO88591DocumentAndNoContentType(t *testing.T) {
	f, err := os.Open("testdata/iso-8859-1.xml")
	require.NoError(t, err)

	reader, err := NewCharsetReader(f, "")
	require.NoError(t, err)

	data, err := io.ReadAll(reader)
	require.NoError(t, err)

	require.True(t, utf8.Valid(data), "Data is not valid UTF-8")

	expected := "Café"
	assert.True(t, bytes.Contains(data, []byte(expected)),
		"Data does not contain expected unicode string: %s", expected)
}

func TestNewReaderWithISO88591DocumentWithMetaAfter1024Bytes(t *testing.T) {
	f, err := os.Open("testdata/iso-8859-1-meta-after-1024.html")
	require.NoError(t, err)

	reader, err := NewCharsetReader(f, "text/html")
	require.NoError(t, err)

	data, err := io.ReadAll(reader)
	require.NoError(t, err)

	require.True(t, utf8.Valid(data), "Data is not valid UTF-8")

	expected := "Café"
	assert.True(t, bytes.Contains(data, []byte(expected)),
		"Data does not contain expected unicode string: %s", expected)
}

func TestNewReaderWithUTF8DocumentWithMetaAfter1024Bytes(t *testing.T) {
	f, err := os.Open("testdata/utf8-meta-after-1024.html")
	require.NoError(t, err)

	reader, err := NewCharsetReader(f, "text/html")
	require.NoError(t, err)

	data, err := io.ReadAll(reader)
	require.NoError(t, err)

	require.True(t, utf8.Valid(data), "Data is not valid UTF-8")

	expected := "Café"
	assert.True(t, bytes.Contains(data, []byte(expected)),
		"Data does not contain expected unicode string: %s", expected)
}
