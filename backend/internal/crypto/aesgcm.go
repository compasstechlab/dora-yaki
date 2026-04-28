package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// aesKeySize is the required AES-256 key length in bytes.
const aesKeySize = 32

type aesGCMEncryptor struct {
	aead cipher.AEAD
}

// newAESGCMEncryptor builds an AES-256-GCM encryptor from a base64-encoded 32-byte key.
func newAESGCMEncryptor(b64 string) (*aesGCMEncryptor, error) {
	key, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		// Tolerate URL-safe alphabet too.
		key, err = base64.URLEncoding.DecodeString(b64)
		if err != nil {
			return nil, fmt.Errorf("invalid ENCRYPTION_KEY_BASE64: %w", err)
		}
	}
	if len(key) != aesKeySize {
		return nil, fmt.Errorf("encryption key must be %d bytes (AES-256), got %d", aesKeySize, len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes.NewCipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cipher.NewGCM: %w", err)
	}
	return &aesGCMEncryptor{aead: aead}, nil
}

// Encrypt seals plaintext with a fresh random nonce. Output layout: nonce || ciphertext-with-tag.
func (e *aesGCMEncryptor) Encrypt(_ context.Context, plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("rand.Read: %w", err)
	}
	return e.aead.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt verifies and opens a ciphertext produced by Encrypt.
func (e *aesGCMEncryptor) Decrypt(_ context.Context, ciphertext []byte) ([]byte, error) {
	ns := e.aead.NonceSize()
	if len(ciphertext) < ns {
		return nil, fmt.Errorf("ciphertext too short: %d < %d", len(ciphertext), ns)
	}
	nonce, ct := ciphertext[:ns], ciphertext[ns:]
	plaintext, err := e.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("aead.Open: %w", err)
	}
	return plaintext, nil
}
