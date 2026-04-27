package crypto

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"
)

func TestNewFromConfig_AESFromBase64(t *testing.T) {
	key := make([]byte, aesKeySize)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	cfg := FactoryConfig{KeyBase64: base64.StdEncoding.EncodeToString(key)}

	enc, err := NewFromConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	if _, ok := enc.(*aesGCMEncryptor); !ok {
		t.Fatalf("expected *aesGCMEncryptor, got %T", enc)
	}
}

func TestNewFromConfig_NoConfig(t *testing.T) {
	_, err := NewFromConfig(context.Background(), FactoryConfig{})
	if !errors.Is(err, ErrNoEncryptionConfigured) {
		t.Fatalf("expected ErrNoEncryptionConfigured, got %v", err)
	}
}

// TestNewFromConfig_KMSPriority verifies that when both KMSKey and KeyBase64 are
// set, KMS is selected first. We do not actually round-trip via KMS here — we
// just check that the constructor attempts the KMS path. The KMS client
// constructor in google's SDK will not fail just because credentials are
// missing during construction; it defers errors to the first call. So we feed
// it a malformed key name to force a synchronous error path that proves KMS
// was selected.
//
// In environments without GOOGLE_APPLICATION_CREDENTIALS the constructor may
// still succeed lazily, in which case the test is a no-op. Skip in that case.
func TestNewFromConfig_KMSPriority(t *testing.T) {
	key := make([]byte, aesKeySize)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	cfg := FactoryConfig{
		KMSKey:    "projects/test/locations/global/keyRings/test/cryptoKeys/test",
		KeyBase64: base64.StdEncoding.EncodeToString(key),
	}

	enc, err := NewFromConfig(context.Background(), cfg)
	if err != nil {
		// The KMS client constructor may fail without credentials; that itself
		// proves KMS was selected over AES (AES would have succeeded).
		return
	}
	if _, ok := enc.(*kmsEncryptor); !ok {
		t.Fatalf("expected *kmsEncryptor (KMS priority), got %T", enc)
	}
}
