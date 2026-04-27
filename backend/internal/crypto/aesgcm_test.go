package crypto

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func newTestEncryptor(t *testing.T) *aesGCMEncryptor {
	t.Helper()
	key := make([]byte, aesKeySize)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	enc, err := newAESGCMEncryptor(base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatalf("newAESGCMEncryptor: %v", err)
	}
	return enc
}

func TestAESGCM_Roundtrip(t *testing.T) {
	enc := newTestEncryptor(t)
	ctx := context.Background()

	cases := []struct {
		name      string
		plaintext []byte
	}{
		{"empty", []byte{}},
		{"short", []byte("hello")},
		{"medium", bytes.Repeat([]byte("a"), 1024)},
		{"large", bytes.Repeat([]byte{0xAB}, 16*1024)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ct, err := enc.Encrypt(ctx, tc.plaintext)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}
			pt, err := enc.Decrypt(ctx, ct)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}
			if !bytes.Equal(pt, tc.plaintext) {
				t.Fatalf("plaintext mismatch: got %q want %q", pt, tc.plaintext)
			}
		})
	}
}

func TestAESGCM_NonceIsRandom(t *testing.T) {
	enc := newTestEncryptor(t)
	ctx := context.Background()
	plaintext := []byte("identical plaintext")

	c1, err := enc.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	c2, err := enc.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if bytes.Equal(c1, c2) {
		t.Fatalf("ciphertexts should differ due to random nonces")
	}
}

func TestAESGCM_TamperDetected(t *testing.T) {
	enc := newTestEncryptor(t)
	ctx := context.Background()

	ct, err := enc.Encrypt(ctx, []byte("secret"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	// Flip a bit somewhere in the body.
	tampered := append([]byte(nil), ct...)
	tampered[len(tampered)-1] ^= 0x01
	if _, err := enc.Decrypt(ctx, tampered); err == nil {
		t.Fatalf("expected error decrypting tampered ciphertext")
	}
}

func TestAESGCM_TruncatedFails(t *testing.T) {
	enc := newTestEncryptor(t)
	ctx := context.Background()

	if _, err := enc.Decrypt(ctx, []byte{0x01, 0x02}); err == nil {
		t.Fatalf("expected error decrypting truncated ciphertext")
	}
}

func TestAESGCM_WrongKeyFails(t *testing.T) {
	ctx := context.Background()
	enc1 := newTestEncryptor(t)
	enc2 := newTestEncryptor(t)

	ct, err := enc1.Encrypt(ctx, []byte("secret"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if _, err := enc2.Decrypt(ctx, ct); err == nil {
		t.Fatalf("expected error decrypting with wrong key")
	}
}

func TestAESGCM_InvalidKeyConfig(t *testing.T) {
	cases := []struct {
		name    string
		keyB64  string
		wantSub string
	}{
		{"invalid base64", "not-base64-!@#$", "invalid ENCRYPTION_KEY_BASE64"},
		{"wrong length", base64.StdEncoding.EncodeToString([]byte("too short")), "must be 32 bytes"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newAESGCMEncryptor(tc.keyB64)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantSub)
			}
		})
	}
}
