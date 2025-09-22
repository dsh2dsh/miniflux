package securecookie

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

const (
	keysNum = 2
	keySize = 32
)

var (
	ErrMsgTooShort = errors.New("http/securecookie: encrypted message too short")
	ErrDecrypt     = errors.New("http/securecookie: decrypt")

	keyLifeTime = time.Hour
)

func New() *SecureCookie { return new(SecureCookie) }

type SecureCookie struct {
	keys       [keysNum][keySize]byte
	lastKeyUse time.Time
}

func (self *SecureCookie) EncryptCookie(plaintext []byte) (string, error) {
	msg, err := self.Encrypt(plaintext)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(msg), nil
}

func (self *SecureCookie) Encrypt(plaintext []byte) ([]byte, error) {
	key, err := self.nextKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf(
			"http/securecookie: new AES cipher for encrypt: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf(
			"http/securecookie: new GCM cipher for encrypt: %w", err)
	}

	nonceSize := aead.NonceSize()
	nonce := make([]byte, nonceSize, nonceSize+len(plaintext)+aead.Overhead())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("http/securecookie: random nonce: %w", err)
	}
	return aead.Seal(nonce, nonce, plaintext, nil), nil
}

func (self *SecureCookie) nextKey() ([]byte, error) {
	if time.Since(self.lastKeyUse) < keyLifeTime {
		return self.keys[0][:], nil
	}

	for i := len(self.keys) - 1; i > 0; i-- {
		self.keys[i] = self.keys[i-1]
	}

	if _, err := rand.Read(self.keys[0][:]); err != nil {
		return nil, fmt.Errorf("http/securecookie: random key: %w", err)
	}
	self.lastKeyUse = time.Now()
	return self.keys[0][:], nil
}

func (self *SecureCookie) DecryptCookie(cookie string) ([]byte, error) {
	msg, err := base64.RawURLEncoding.DecodeString(cookie)
	if err != nil {
		return nil, fmt.Errorf("http/securecookie: base64 decode cookie: %w", err)
	}
	return self.Decrypt(msg)
}

func (self *SecureCookie) Decrypt(message []byte) ([]byte, error) {
	var resultErr error
	for i := range len(self.keys) {
		key := self.keys[i][:]
		plaintext, err := self.decrypt(key, message)
		if err == nil || !errors.Is(err, ErrDecrypt) {
			return plaintext, err
		} else if resultErr != nil {
			resultErr = err
		}
	}
	return nil, resultErr
}

func (self *SecureCookie) decrypt(key, message []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf(
			"http/securecookie: new AES cipher for decrypt: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf(
			"http/securecookie: new GCM cipher for decrypt: %w", err)
	}

	nonceSize := aead.NonceSize()
	wantSize := nonceSize + aead.Overhead()
	if len(message) < wantSize {
		return nil, fmt.Errorf("%w (got %v, want %v or more)", ErrMsgTooShort,
			len(message), wantSize)
	}

	nonce, cipertext := message[:nonceSize], message[nonceSize:]
	plaintext, err := aead.Open(nil, nonce, cipertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecrypt, err)
	}
	return plaintext, nil
}
