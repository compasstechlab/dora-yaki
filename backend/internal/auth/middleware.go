package auth

import (
	"encoding/json"
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
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		token := strings.TrimPrefix(h, "Bearer ")
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

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
