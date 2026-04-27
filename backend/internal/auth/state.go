package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// stateMaxAge limits how long an OAuth state parameter is considered fresh.
// Long enough for slow GitHub redirects, short enough to limit replay window.
const stateMaxAge = 10 * time.Minute

// statePayload is the inner JSON of the OAuth state parameter.
type statePayload struct {
	Nonce    string `json:"n"`
	ReturnTo string `json:"r"`
	IssuedAt int64  `json:"iat"`
}

// EncodeState builds an HMAC-signed, base64-encoded state value tying together a
// random nonce, the originating return path, and the issuance time. Verifies
// the redirect was initiated by us (CSRF protection) and lets the callback
// know where to send the user back.
func EncodeState(secret []byte, returnTo string) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("rand.Read: %w", err)
	}
	payload := statePayload{
		Nonce:    base64.RawURLEncoding.EncodeToString(nonce),
		ReturnTo: returnTo,
		IssuedAt: time.Now().Unix(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal state: %w", err)
	}
	body := base64.RawURLEncoding.EncodeToString(raw)
	sig := computeHMAC(secret, body)
	return body + "." + sig, nil
}

// DecodeState verifies the signature/freshness and returns the embedded ReturnTo.
func DecodeState(secret []byte, state string) (string, error) {
	parts := strings.SplitN(state, ".", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid state format")
	}
	body, sig := parts[0], parts[1]
	expected := computeHMAC(secret, body)
	// Constant-time comparison to avoid timing side-channels on the signature.
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", fmt.Errorf("invalid state signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return "", fmt.Errorf("decode state body: %w", err)
	}
	var payload statePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("parse state payload: %w", err)
	}
	if time.Since(time.Unix(payload.IssuedAt, 0)) > stateMaxAge {
		return "", fmt.Errorf("state expired")
	}
	return SanitizeReturnTo(payload.ReturnTo), nil
}

// SanitizeReturnTo restricts post-login redirect targets to relative paths so
// the callback cannot be turned into an open redirect.
func SanitizeReturnTo(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		return "/"
	}
	if strings.HasPrefix(p, "//") {
		return "/"
	}
	return p
}
