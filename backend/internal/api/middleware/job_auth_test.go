package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJobAuthAllowsHeaderKey(t *testing.T) {
	handler := JobAuth("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPut, "/api/job/sync", nil)
	req.Header.Set(HeaderJobAuthKey, "secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestJobAuthAllowsBasicUsername(t *testing.T) {
	handler := JobAuth("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPut, "/api/job/sync", nil)
	req.SetBasicAuth("secret", "")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestJobAuthRejectsInvalidKey(t *testing.T) {
	handler := JobAuth("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("inner handler should not run")
	}))

	req := httptest.NewRequest(http.MethodPut, "/api/job/sync", nil)
	req.Header.Set(HeaderJobAuthKey, "wrong")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestJobAuthBypassesWhenKeyIsEmpty(t *testing.T) {
	handler := JobAuth("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPut, "/api/job/sync", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
