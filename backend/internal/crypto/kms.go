package crypto

import (
	"context"
	"fmt"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
)

type kmsEncryptor struct {
	client  *kms.KeyManagementClient
	keyName string
}

// newKMSEncryptor connects to Cloud KMS and verifies access by issuing a no-op
// GetCryptoKey call would be ideal, but to keep startup cheap we defer the first
// real round-trip to the first Encrypt/Decrypt call.
func newKMSEncryptor(ctx context.Context, keyName string) (*kmsEncryptor, error) {
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("kms.NewKeyManagementClient: %w", err)
	}
	return &kmsEncryptor{client: client, keyName: keyName}, nil
}

func (e *kmsEncryptor) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	resp, err := e.client.Encrypt(ctx, &kmspb.EncryptRequest{
		Name:      e.keyName,
		Plaintext: plaintext,
	})
	if err != nil {
		return nil, fmt.Errorf("kms encrypt: %w", err)
	}
	return resp.GetCiphertext(), nil
}

func (e *kmsEncryptor) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	resp, err := e.client.Decrypt(ctx, &kmspb.DecryptRequest{
		Name:       e.keyName,
		Ciphertext: ciphertext,
	})
	if err != nil {
		return nil, fmt.Errorf("kms decrypt: %w", err)
	}
	return resp.GetPlaintext(), nil
}
