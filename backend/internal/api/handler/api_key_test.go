package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/apikey"
	"github.com/compasstechlab/dora-yaki/internal/auth"
	"github.com/compasstechlab/dora-yaki/internal/domain/model"
)

type fakeAPIKeyStore struct {
	keys    map[string]*model.APIKey
	saved   *model.APIKey
	deleted []string
}

func newFakeAPIKeyStore(keys ...*model.APIKey) *fakeAPIKeyStore {
	s := &fakeAPIKeyStore{keys: map[string]*model.APIKey{}}
	for _, k := range keys {
		s.keys[k.ID] = k
	}
	return s
}

func (s *fakeAPIKeyStore) SaveAPIKey(_ context.Context, k *model.APIKey) error {
	s.saved = k
	s.keys[k.ID] = k
	return nil
}

func (s *fakeAPIKeyStore) GetAPIKey(_ context.Context, id string) (*model.APIKey, error) {
	return s.keys[id], nil
}

// ListAPIKeysByUser : mirrors the datastore contract (caller's keys, newest first).
func (s *fakeAPIKeyStore) ListAPIKeysByUser(_ context.Context, userID string) ([]*model.APIKey, error) {
	var out []*model.APIKey
	for _, k := range s.keys {
		if k.UserID == userID {
			out = append(out, k)
		}
	}
	slices.SortFunc(out, func(a, b *model.APIKey) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return out, nil
}

func (s *fakeAPIKeyStore) DeleteAPIKey(_ context.Context, id string) error {
	s.deleted = append(s.deleted, id)
	delete(s.keys, id)
	return nil
}

var apiKeyTestNow = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

func newTestAPIKeyHandler(store apiKeyStore) *APIKeyHandler {
	h := NewAPIKeyHandler(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	h.now = func() time.Time { return apiKeyTestNow }
	return h
}

func withTestClaims(r *http.Request) *http.Request {
	return r.WithContext(auth.WithClaims(r.Context(), &auth.Claims{UserID: "42", Login: "alice"}))
}

func TestAPIKeyHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		existing   int
		noClaims   bool
		wantStatus int
		wantDays   int
	}{
		{name: "default expiry", body: `{"name":"ci"}`, wantStatus: http.StatusCreated, wantDays: 90},
		{name: "30 days", body: `{"name":"ci","expiresInDays":30}`, wantStatus: http.StatusCreated, wantDays: 30},
		{name: "empty name", body: `{"name":""}`, wantStatus: http.StatusBadRequest},
		{name: "blank name", body: `{"name":"   "}`, wantStatus: http.StatusBadRequest},
		{name: "65 rune name", body: `{"name":"` + strings.Repeat("あ", 65) + `"}`, wantStatus: http.StatusBadRequest},
		{name: "64 rune name", body: `{"name":"` + strings.Repeat("あ", 64) + `"}`, wantStatus: http.StatusCreated, wantDays: 90},
		{name: "disallowed expiry", body: `{"name":"ci","expiresInDays":7}`, wantStatus: http.StatusBadRequest},
		{name: "invalid json", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "one below limit", body: `{"name":"ci"}`, existing: 2, wantStatus: http.StatusCreated, wantDays: 90},
		{name: "limit reached", body: `{"name":"ci"}`, existing: 3, wantStatus: http.StatusConflict},
		{name: "no claims", body: `{"name":"ci"}`, noClaims: true, wantStatus: http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeAPIKeyStore()
			for i := range tt.existing {
				store.keys[strconv.Itoa(i)] = &model.APIKey{ID: strconv.Itoa(i), UserID: "42"}
			}
			h := newTestAPIKeyHandler(store)

			req := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewBufferString(tt.body))
			if !tt.noClaims {
				req = withTestClaims(req)
			}
			rec := httptest.NewRecorder()
			h.Create(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != http.StatusCreated {
				if store.saved != nil {
					t.Fatalf("key must not be saved on failure")
				}
				return
			}
			assertCreatedKey(t, rec, store.saved, tt.wantDays)
		})
	}
}

func assertCreatedKey(t *testing.T, rec *httptest.ResponseRecorder, saved *model.APIKey, wantDays int) {
	t.Helper()
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	raw := rec.Body.String()
	if strings.Contains(raw, "secretHash") || strings.Contains(raw, "userId") {
		t.Fatalf("response leaks internal fields: %s", raw)
	}
	var resp createAPIKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.HasPrefix(resp.Token, apikey.Prefix) {
		t.Fatalf("token %q lacks prefix", resp.Token)
	}
	id, secret, ok := apikey.Parse(resp.Token)
	if !ok || saved == nil || saved.ID != id || resp.APIKey.ID != id {
		t.Fatalf("token/id mismatch: token id=%q saved=%+v resp=%+v", id, saved, resp.APIKey)
	}
	if !apikey.SecretMatches(secret, saved.SecretHash) {
		t.Fatalf("saved hash does not match token secret")
	}
	if saved.UserID != "42" || saved.UserLogin != "alice" {
		t.Fatalf("owner = (%q, %q), want (42, alice)", saved.UserID, saved.UserLogin)
	}
	wantExpiry := apiKeyTestNow.AddDate(0, 0, wantDays)
	if saved.ExpiresAt == nil || !saved.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("ExpiresAt = %v, want %v", saved.ExpiresAt, wantExpiry)
	}
	if resp.APIKey.Prefix != apikey.Prefix+id {
		t.Fatalf("prefix = %q, want %q", resp.APIKey.Prefix, apikey.Prefix+id)
	}
}

func TestAPIKeyHandler_List(t *testing.T) {
	older := &model.APIKey{ID: "aaaaaaaaaaaaaaaa", UserID: "42", Name: "old", CreatedAt: apiKeyTestNow.Add(-time.Hour)}
	newer := &model.APIKey{ID: "bbbbbbbbbbbbbbbb", UserID: "42", Name: "new", CreatedAt: apiKeyTestNow}
	others := &model.APIKey{ID: "cccccccccccccccc", UserID: "7", Name: "other", CreatedAt: apiKeyTestNow}

	t.Run("returns caller keys newest first", func(t *testing.T) {
		h := newTestAPIKeyHandler(newFakeAPIKeyStore(older, newer, others))
		rec := httptest.NewRecorder()
		h.List(rec, withTestClaims(httptest.NewRequest(http.MethodGet, "/api/api-keys", nil)))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		var got []apiKeyResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got) != 2 || got[0].ID != newer.ID || got[1].ID != older.ID {
			t.Fatalf("got %+v, want [new, old]", got)
		}
		if got[0].Prefix != "dyk_"+newer.ID {
			t.Fatalf("prefix = %q", got[0].Prefix)
		}
	})

	t.Run("empty list is []", func(t *testing.T) {
		h := newTestAPIKeyHandler(newFakeAPIKeyStore(others))
		rec := httptest.NewRecorder()
		h.List(rec, withTestClaims(httptest.NewRequest(http.MethodGet, "/api/api-keys", nil)))

		if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
			t.Fatalf("body = %q, want []", body)
		}
	})

	t.Run("no claims", func(t *testing.T) {
		h := newTestAPIKeyHandler(newFakeAPIKeyStore())
		rec := httptest.NewRecorder()
		h.List(rec, httptest.NewRequest(http.MethodGet, "/api/api-keys", nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
}

func TestAPIKeyHandler_Revoke(t *testing.T) {
	mine := &model.APIKey{ID: "aaaaaaaaaaaaaaaa", UserID: "42"}
	others := &model.APIKey{ID: "cccccccccccccccc", UserID: "7"}

	tests := []struct {
		name        string
		id          string
		wantStatus  int
		wantDeleted bool
	}{
		{name: "own key", id: mine.ID, wantStatus: http.StatusNoContent, wantDeleted: true},
		{name: "other user's key", id: others.ID, wantStatus: http.StatusNotFound},
		{name: "unknown key", id: "dddddddddddddddd", wantStatus: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeAPIKeyStore(mine, others)
			h := newTestAPIKeyHandler(store)

			req := httptest.NewRequest(http.MethodDelete, "/api/api-keys/"+tt.id, nil)
			req.SetPathValue("id", tt.id)
			rec := httptest.NewRecorder()
			h.Revoke(rec, withTestClaims(req))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if deleted := len(store.deleted) > 0; deleted != tt.wantDeleted {
				t.Fatalf("deleted = %v, want %v", store.deleted, tt.wantDeleted)
			}
		})
	}
}
