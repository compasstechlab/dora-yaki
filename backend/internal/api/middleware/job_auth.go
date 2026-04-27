package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
)

const HeaderJobAuthKey = "X-DORA-YAKI-JOB-KEY"

// JobAuth は scheduler/job エンドポイントを共有キーで保護する。
// キーは Basic auth の username または X-DORA-YAKI-JOB-KEY で渡す。
func JobAuth(authKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authKey == "" || authenticateJob(r, authKey) {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid job auth key"})
		})
	}
}

func authenticateJob(r *http.Request, authKey string) bool {
	if username, _, ok := r.BasicAuth(); ok && secureCompare(username, authKey) {
		return true
	}

	headerKey := r.Header.Get(HeaderJobAuthKey)
	return headerKey != "" && secureCompare(headerKey, authKey)
}

func secureCompare(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
