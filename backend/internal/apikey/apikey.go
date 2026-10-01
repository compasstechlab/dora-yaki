package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/compasstechlab/dora-yaki/internal/domain/model"
)

const (
	// Prefix : prefix of every API key token.
	Prefix = model.APIKeyPrefix

	idBytes     = 8  // 16 hex chars
	secretBytes = 32 // 43 base64url chars
)

// Generate : creates a new token ("dyk_<id>_<secret>") and returns its id and the secret hash to persist.
func Generate() (token, id string, hash []byte, err error) {
	idRaw := make([]byte, idBytes)
	if _, err := rand.Read(idRaw); err != nil {
		return "", "", nil, fmt.Errorf("generate api key id: %w", err)
	}
	secretRaw := make([]byte, secretBytes)
	if _, err := rand.Read(secretRaw); err != nil {
		return "", "", nil, fmt.Errorf("generate api key secret: %w", err)
	}
	id = hex.EncodeToString(idRaw)
	secret := base64.RawURLEncoding.EncodeToString(secretRaw)
	return Prefix + id + "_" + secret, id, HashSecret(secret), nil
}

// Parse : splits a token into id and secret. ok is false when the token is not a well-formed API key.
func Parse(token string) (id, secret string, ok bool) {
	rest, found := strings.CutPrefix(token, Prefix)
	if !found {
		return "", "", false
	}
	id, secret, found = strings.Cut(rest, "_")
	if !found || secret == "" || len(id) != idBytes*2 {
		return "", "", false
	}
	if _, err := hex.DecodeString(id); err != nil {
		return "", "", false
	}
	return id, secret, true
}

// HashSecret : returns the SHA-256 digest of the secret.
func HashSecret(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

// SecretMatches : compares the secret against a stored hash in constant time.
func SecretMatches(secret string, hash []byte) bool {
	return subtle.ConstantTimeCompare(HashSecret(secret), hash) == 1
}
