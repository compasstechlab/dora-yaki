package apikey

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/auth"
	"github.com/compasstechlab/dora-yaki/internal/domain/model"
)

type fakeStore struct {
	keys        map[string]*model.APIKey
	tokens      map[string]*model.UserGitHubToken
	getKeyErr   error
	updateErr   error
	getKeyCalls int
	updateCalls int
	updatedAt   time.Time
}

func (s *fakeStore) GetAPIKey(_ context.Context, id string) (*model.APIKey, error) {
	s.getKeyCalls++
	if s.getKeyErr != nil {
		return nil, s.getKeyErr
	}
	return s.keys[id], nil
}

func (s *fakeStore) UpdateAPIKeyLastUsed(_ context.Context, _ string, at time.Time) error {
	s.updateCalls++
	s.updatedAt = at
	return s.updateErr
}

func (s *fakeStore) GetUserGitHubToken(_ context.Context, userID string) (*model.UserGitHubToken, error) {
	return s.tokens[userID], nil
}

func TestAuthenticator_VerifyAPIKey(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(24 * time.Hour)
	past := now.Add(-time.Second)
	fiveMinAgo := now.Add(-5 * time.Minute)
	twentyMinAgo := now.Add(-20 * time.Minute)

	token, id, hash, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	_, secret, _ := Parse(token)
	otherToken := Prefix + id + "_" + secret + "x"

	newKey := func(mod func(k *model.APIKey)) *model.APIKey {
		k := &model.APIKey{ID: id, UserID: "42", UserLogin: "alice", SecretHash: hash, ExpiresAt: &future}
		if mod != nil {
			mod(k)
		}
		return k
	}
	validOwner := map[string]*model.UserGitHubToken{"42": {UserID: "42"}}

	tests := []struct {
		name         string
		token        string
		key          *model.APIKey
		owners       map[string]*model.UserGitHubToken
		getKeyErr    error
		updateErr    error
		wantErr      error // sentinel; nil means success
		wantOtherErr bool  // non-sentinel error expected
		wantNoStore  bool
		wantUpdates  int
	}{
		{name: "valid", token: token, key: newKey(nil), owners: validOwner, wantUpdates: 1},
		{name: "not an api key", token: "abcd.ef01", wantErr: auth.ErrNotAPIKey, wantNoStore: true},
		{name: "malformed", token: Prefix + "zz_abc", wantErr: auth.ErrInvalidAPIKey, wantNoStore: true},
		{name: "unknown id", token: token, owners: validOwner, wantErr: auth.ErrInvalidAPIKey},
		{name: "secret mismatch", token: otherToken, key: newKey(nil), owners: validOwner, wantErr: auth.ErrInvalidAPIKey},
		{name: "expired", token: token, key: newKey(func(k *model.APIKey) { k.ExpiresAt = &past }), owners: validOwner, wantErr: auth.ErrInvalidAPIKey},
		{name: "owner token missing", token: token, key: newKey(nil), owners: nil, wantErr: auth.ErrInvalidAPIKey},
		{name: "owner token marked invalid", token: token, key: newKey(nil), owners: map[string]*model.UserGitHubToken{"42": {UserID: "42", MarkedInvalidAt: &past}}, wantErr: auth.ErrInvalidAPIKey},
		{name: "store error", token: token, getKeyErr: errors.New("boom"), wantOtherErr: true},
		{name: "recently used skips update", token: token, key: newKey(func(k *model.APIKey) { k.LastUsedAt = &fiveMinAgo }), owners: validOwner, wantUpdates: 0},
		{name: "stale last used updates", token: token, key: newKey(func(k *model.APIKey) { k.LastUsedAt = &twentyMinAgo }), owners: validOwner, wantUpdates: 1},
		{name: "update failure still succeeds", token: token, key: newKey(nil), owners: validOwner, updateErr: errors.New("boom"), wantUpdates: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{
				keys:      map[string]*model.APIKey{},
				tokens:    tt.owners,
				getKeyErr: tt.getKeyErr,
				updateErr: tt.updateErr,
			}
			if tt.key != nil {
				store.keys[tt.key.ID] = tt.key
			}
			a := NewAuthenticator(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
			a.now = func() time.Time { return now }

			userID, login, err := a.VerifyAPIKey(t.Context(), tt.token)

			switch {
			case tt.wantOtherErr:
				if err == nil || errors.Is(err, auth.ErrNotAPIKey) || errors.Is(err, auth.ErrInvalidAPIKey) {
					t.Fatalf("err = %v, want non-sentinel error", err)
				}
			case tt.wantErr != nil:
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
			default:
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
				if userID != "42" || login != "alice" {
					t.Fatalf("got (%q, %q), want (42, alice)", userID, login)
				}
			}
			if err != nil && strings.Contains(err.Error(), secret) {
				t.Fatalf("error must not contain the secret: %v", err)
			}
			if tt.wantNoStore && store.getKeyCalls != 0 {
				t.Fatalf("store must not be called, got %d calls", store.getKeyCalls)
			}
			if store.updateCalls != tt.wantUpdates {
				t.Fatalf("UpdateAPIKeyLastUsed calls = %d, want %d", store.updateCalls, tt.wantUpdates)
			}
			if tt.wantUpdates > 0 && !store.updatedAt.Equal(now) {
				t.Fatalf("updated at %v, want %v", store.updatedAt, now)
			}
		})
	}
}
