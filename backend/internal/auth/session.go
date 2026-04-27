// Package auth provides session token signing/verification, GitHub OAuth helpers,
// and HTTP middleware for authenticating requests.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	// CookieName is the session cookie name set on authenticated browsers.
	CookieName = "dora_yaki_session"
	// TokenTTL is the lifetime of a session token (and its cookie).
	TokenTTL = 14 * 24 * time.Hour
)

// Claims is the payload of a session token.
type Claims struct {
	UserID    string `json:"user_id"`
	Login     string `json:"login"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// SignToken serializes claims and signs them with HMAC-SHA256.
// Output format: hex(payload) + "." + hex(sig).
func SignToken(secret []byte, claims Claims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}
	payloadHex := hex.EncodeToString(payload)
	sig := computeHMAC(secret, payloadHex)
	return payloadHex + "." + sig, nil
}

// VerifyToken parses, verifies, and returns the claims of a session token.
// Returns an error if the format, signature, or expiry are invalid.
func VerifyToken(secret []byte, token string) (*Claims, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid token format")
	}
	payloadHex, sig := parts[0], parts[1]
	expected := computeHMAC(secret, payloadHex)
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return nil, fmt.Errorf("invalid token signature")
	}
	payload, err := hex.DecodeString(payloadHex)
	if err != nil {
		return nil, fmt.Errorf("invalid token payload: %w", err)
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("token expired")
	}
	return &claims, nil
}

// NewClaims builds a fresh Claims with IssuedAt=now and ExpiresAt=now+TokenTTL.
func NewClaims(userID, login string) Claims {
	now := time.Now()
	return Claims{
		UserID:    userID,
		Login:     login,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(TokenTTL).Unix(),
	}
}

func computeHMAC(secret []byte, data string) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
