package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

// ExtractClaims attempts to read a session token from the request (cookie first,
// then Authorization: Bearer) and returns the verified claims. Returns false if
// either source yields a valid token.
func ExtractClaims(r *http.Request, secret []byte) (*Claims, bool) {
	if cookie, err := r.Cookie(CookieName); err == nil {
		if claims, err := VerifyToken(secret, cookie.Value); err == nil {
			return claims, true
		}
	}
	if token, ok := bearerToken(r); ok {
		if claims, err := VerifyToken(secret, token); err == nil {
			return claims, true
		}
	}
	return nil, false
}

// RequireAuth wraps next in a middleware that requires a valid session token.
// On failure it responds with 401 + JSON error. On success the verified Claims
// are attached to the request context (retrievable via FromContext).
func RequireAuth(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ExtractClaims(r, secret)
			if !ok {
				writeUnauthorized(w)
				return
			}
			ctx := WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuthOrAPIKey : like RequireAuth, but also accepts an API key bearer token. Verifier failures other than rejection become 500.
func RequireAuthOrAPIKey(secret []byte, v APIKeyVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if claims, ok := ExtractClaims(r, secret); ok {
				next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
				return
			}
			token, ok := bearerToken(r)
			if !ok {
				writeUnauthorized(w)
				return
			}
			userID, login, err := v.VerifyAPIKey(r.Context(), token)
			switch {
			case errors.Is(err, ErrNotAPIKey), errors.Is(err, ErrInvalidAPIKey):
				writeUnauthorized(w)
				return
			case err != nil:
				slog.ErrorContext(r.Context(), "failed to verify api key", "error", err)
				writeJSONError(w, http.StatusInternalServerError, "internal")
				return
			}
			ctx := WithClaims(r.Context(), &Claims{UserID: userID, Login: login})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// bearerToken : returns the token of an "Authorization: Bearer <token>" header.
func bearerToken(r *http.Request) (string, bool) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return token, ok && token != ""
}

func writeUnauthorized(w http.ResponseWriter) {
	writeJSONError(w, http.StatusUnauthorized, "unauthorized")
}

// writeJSONError : writes {"error": msg} with the given status.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
