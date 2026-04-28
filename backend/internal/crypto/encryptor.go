// Package crypto provides encryption primitives used to protect sensitive data
// such as OAuth tokens at rest in Datastore.
package crypto

import (
	"context"
	"errors"
)

// Encryptor encrypts and decrypts opaque byte payloads. Implementations are
// expected to authenticate ciphertexts (e.g., AES-GCM, KMS).
type Encryptor interface {
	Encrypt(ctx context.Context, plaintext []byte) ([]byte, error)
	Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)
}

// FactoryConfig selects which Encryptor implementation to construct.
//
// Selection priority:
//  1. KMSKey (Cloud KMS resource path)
//  2. KeyBase64 (base64-encoded 32-byte AES-256 key for AES-GCM)
type FactoryConfig struct {
	KMSKey    string
	KeyBase64 string
}

// ErrNoEncryptionConfigured is returned when neither KMS nor a local AES key is configured.
var ErrNoEncryptionConfigured = errors.New("no encryption configured (set ENCRYPTION_KMS_KEY or ENCRYPTION_KEY_BASE64)")

// NewFromConfig returns the highest-priority Encryptor configured.
func NewFromConfig(ctx context.Context, cfg FactoryConfig) (Encryptor, error) {
	if cfg.KMSKey != "" {
		return newKMSEncryptor(ctx, cfg.KMSKey)
	}
	if cfg.KeyBase64 != "" {
		return newAESGCMEncryptor(cfg.KeyBase64)
	}
	return nil, ErrNoEncryptionConfigured
}
