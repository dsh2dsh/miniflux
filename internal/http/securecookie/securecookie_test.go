package securecookie

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	s := New()
	require.NotNil(t, s)

	plaintext := []byte("OrpheanBeholderScryDoubt")
	msg, err := s.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, msg)

	decrypted, err := s.Decrypt(msg)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptDecrypt_nextKey(t *testing.T) {
	s := New()
	require.NotNil(t, s)

	plaintext := []byte("OrpheanBeholderScryDoubt")
	msg, err := s.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, msg)

	s.lastKeyUse = time.Now().Add(-keyLifeTime)
	key, err := s.nextKey()
	require.NoError(t, err)
	assert.NotEmpty(t, key)
	assert.Equal(t, key, s.keys[0][:])
	assert.NotEqual(t, s.keys[0][:], s.keys[1][:])

	decrypted, err := s.Decrypt(msg)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptDecrypt_Cookie(t *testing.T) {
	s := New()
	require.NotNil(t, s)

	plaintext := []byte("OrpheanBeholderScryDoubt")
	cookie, err := s.EncryptCookie(plaintext)
	t.Log(cookie)
	require.NoError(t, err)
	assert.NotEmpty(t, cookie)

	decrypted, err := s.DecryptCookie(cookie)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}
