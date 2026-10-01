package apikey

import (
	"bytes"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	token, id, hash, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.HasPrefix(token, Prefix) {
		t.Fatalf("token %q does not start with %q", token, Prefix)
	}
	gotID, secret, ok := Parse(token)
	if !ok {
		t.Fatalf("Parse(%q) failed", token)
	}
	if gotID != id {
		t.Fatalf("parsed id = %q, want %q", gotID, id)
	}
	if !bytes.Equal(HashSecret(secret), hash) {
		t.Fatalf("hash mismatch")
	}

	token2, id2, _, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if token2 == token || id2 == id {
		t.Fatalf("two generated keys must differ")
	}
}

func TestParse(t *testing.T) {
	//nolint:gosec // G101: dummy fixtures, not real credentials
	tests := []struct {
		name       string
		token      string
		wantID     string
		wantSecret string
		wantOK     bool
	}{
		{name: "valid", token: "dyk_3f9a1c2b7d4e5f60_abcDEF123", wantID: "3f9a1c2b7d4e5f60", wantSecret: "abcDEF123", wantOK: true},
		{name: "secret with _ and -", token: "dyk_3f9a1c2b7d4e5f60_q8X_w-Zk_2", wantID: "3f9a1c2b7d4e5f60", wantSecret: "q8X_w-Zk_2", wantOK: true},
		{name: "session token", token: "abcd.ef01"},
		{name: "missing prefix", token: "3f9a1c2b7d4e5f60_abc"},
		{name: "missing separator", token: "dyk_3f9a1c2b7d4e5f60abc"},
		{name: "short id", token: "dyk_3f9a1c2b_abc"},
		{name: "non-hex id", token: "dyk_3f9a1c2b7d4e5fzz_abc"},
		{name: "empty secret", token: "dyk_3f9a1c2b7d4e5f60_"},
		{name: "empty", token: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, secret, ok := Parse(tt.token)
			if ok != tt.wantOK || id != tt.wantID || secret != tt.wantSecret {
				t.Fatalf("Parse(%q) = (%q, %q, %v), want (%q, %q, %v)", tt.token, id, secret, ok, tt.wantID, tt.wantSecret, tt.wantOK)
			}
		})
	}
}

func TestSecretMatches(t *testing.T) {
	hash := HashSecret("secret-value")

	tests := []struct {
		name   string
		secret string
		hash   []byte
		want   bool
	}{
		{name: "match", secret: "secret-value", hash: hash, want: true},
		{name: "one char differs", secret: "secret-valuf", hash: hash, want: false},
		{name: "hash length differs", secret: "secret-value", hash: hash[:16], want: false},
		{name: "nil hash", secret: "secret-value", hash: nil, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SecretMatches(tt.secret, tt.hash); got != tt.want {
				t.Fatalf("SecretMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}
