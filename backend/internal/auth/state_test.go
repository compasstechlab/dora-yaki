package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestEncodeDecodeState_Roundtrip(t *testing.T) {
	s, err := EncodeState(testSecret, "/repositories")
	if err != nil {
		t.Fatalf("EncodeState: %v", err)
	}
	got, err := DecodeState(testSecret, s)
	if err != nil {
		t.Fatalf("DecodeState: %v", err)
	}
	if got != "/repositories" {
		t.Fatalf("returnTo mismatch: got %q want %q", got, "/repositories")
	}
}

func TestDecodeState_Tampered(t *testing.T) {
	s, err := EncodeState(testSecret, "/")
	if err != nil {
		t.Fatalf("EncodeState: %v", err)
	}
	// Flip a single character in the body half.
	parts := strings.SplitN(s, ".", 2)
	tampered := flipFirstHexChar(parts[0]) + "." + parts[1]
	if _, err := DecodeState(testSecret, tampered); err == nil {
		t.Fatalf("expected tamper error")
	}
}

func TestDecodeState_Expired(t *testing.T) {
	// Build a state body whose IssuedAt is in the past.
	payload := statePayload{
		Nonce:    "deadbeef",
		ReturnTo: "/",
		IssuedAt: time.Now().Add(-2 * stateMaxAge).Unix(),
	}
	raw, _ := json.Marshal(payload)
	body := base64.RawURLEncoding.EncodeToString(raw)
	sig := computeHMAC(testSecret, body)
	s := body + "." + sig

	if _, err := DecodeState(testSecret, s); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired error, got %v", err)
	}
}

func TestSanitizeReturnTo(t *testing.T) {
	cases := map[string]string{
		"":                   "/",
		"/":                  "/",
		"/repositories":      "/repositories",
		"http://evil.com":    "/", // protocol-relative not allowed
		"//evil.com/path":    "/", // protocol-relative not allowed
		"javascript:alert()": "/",
	}
	for in, want := range cases {
		if got := SanitizeReturnTo(in); got != want {
			t.Errorf("SanitizeReturnTo(%q) = %q, want %q", in, got, want)
		}
	}
}
