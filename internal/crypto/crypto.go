// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package crypto // import "miniflux.app/v2/internal/crypto"

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/cespare/xxhash/v2"
	"golang.org/x/crypto/bcrypt"
)

// Sun  6 Jul 2025 22:07:51 CEST
var CompatHashBefore = time.Unix(1751832471, 0).UTC()

// HashFromBytes returns a non-cryptographic checksum of the input.
func HashFromBytes(b []byte) string {
	return strconv.FormatUint(xxhash.Sum64(b), 16)
}

func HashFromStringCompat(s string, t time.Time) string {
	if t.UTC().Before(CompatHashBefore) {
		return SHA256(s)
	}
	return HashFromBytes([]byte(s))
}

// SHA256 returns a SHA-256 checksum of a string.
func SHA256(value string) string {
	h := sha256.Sum256([]byte(value))
	return hex.EncodeToString(h[:])
}

// GenerateRandomBytes returns random bytes.
func GenerateRandomBytes(size int) []byte {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}

// GenerateRandomStringHex returns a random hexadecimal string.
func GenerateRandomStringHex(size int) string {
	return hex.EncodeToString(GenerateRandomBytes(size))
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func GenerateSHA256Hmac(secret string, data []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func GenerateUUID() string {
	b := GenerateRandomBytes(16)
	return fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func ConstantTimeCmp(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
