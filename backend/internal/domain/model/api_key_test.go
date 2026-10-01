package model

import (
	"testing"
	"time"
)

func TestAPIKey_IsExpired(t *testing.T) {
	expiry := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		expiresAt *time.Time
		now       time.Time
		want      bool
	}{
		{name: "no expiry", expiresAt: nil, now: expiry, want: false},
		{name: "before expiry", expiresAt: &expiry, now: expiry.Add(-time.Second), want: false},
		{name: "at expiry", expiresAt: &expiry, now: expiry, want: true},
		{name: "after expiry", expiresAt: &expiry, now: expiry.Add(time.Second), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &APIKey{ExpiresAt: tt.expiresAt}
			if got := k.IsExpired(tt.now); got != tt.want {
				t.Fatalf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIKey_DisplayPrefix(t *testing.T) {
	k := &APIKey{ID: "3f9a1c2b7d4e5f60"}
	if got, want := k.DisplayPrefix(), "dyk_3f9a1c2b7d4e5f60"; got != want {
		t.Fatalf("DisplayPrefix() = %q, want %q", got, want)
	}
}
