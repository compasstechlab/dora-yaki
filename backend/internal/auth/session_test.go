package auth

import (
	"strings"
	"testing"
	"time"
)

var testSecret = []byte("test-secret-do-not-use-in-prod-32!")

func TestSignVerifyToken_Roundtrip(t *testing.T) {
	claims := NewClaims("12345", "octocat")
	tok, err := SignToken(testSecret, claims)
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	got, err := VerifyToken(testSecret, tok)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if got.UserID != claims.UserID || got.Login != claims.Login {
		t.Fatalf("claims mismatch: got=%+v want=%+v", got, claims)
	}
}

func TestVerifyToken_Expired(t *testing.T) {
	claims := Claims{
		UserID:    "1",
		Login:     "x",
		IssuedAt:  time.Now().Add(-2 * TokenTTL).Unix(),
		ExpiresAt: time.Now().Add(-time.Hour).Unix(),
	}
	tok, err := SignToken(testSecret, claims)
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	if _, err := VerifyToken(testSecret, tok); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired error, got %v", err)
	}
}

func TestVerifyToken_TamperedPayload(t *testing.T) {
	tok, err := SignToken(testSecret, NewClaims("1", "x"))
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	// Mutate one hex character of the payload.
	parts := strings.SplitN(tok, ".", 2)
	if len(parts) != 2 {
		t.Fatalf("unexpected token shape: %q", tok)
	}
	flipped := flipFirstHexChar(parts[0])
	bad := flipped + "." + parts[1]
	if _, err := VerifyToken(testSecret, bad); err == nil {
		t.Fatalf("expected error for tampered payload")
	}
}

func TestVerifyToken_WrongSecret(t *testing.T) {
	tok, err := SignToken(testSecret, NewClaims("1", "x"))
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	if _, err := VerifyToken([]byte("different-secret"), tok); err == nil {
		t.Fatalf("expected error for wrong secret")
	}
}

func TestVerifyToken_BadFormat(t *testing.T) {
	cases := []string{"", "no-dot-here", "only.one.dot.too.many"}
	for _, tc := range cases {
		if _, err := VerifyToken(testSecret, tc); err == nil {
			t.Fatalf("expected error for %q", tc)
		}
	}
}

func flipFirstHexChar(s string) string {
	if s == "" {
		return s
	}
	b := []byte(s)
	if b[0] == '0' {
		b[0] = '1'
	} else {
		b[0] = '0'
	}
	return string(b)
}
