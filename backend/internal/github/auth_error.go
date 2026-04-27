package github

import (
	"errors"
	"net/http"
	"strings"

	gh "github.com/google/go-github/v82/github"
)

// IsAuthError reports whether an error from a go-github call indicates that
// the calling token is invalid or no longer authorized for the resource.
//
// Returns true on:
//   - HTTP 401 (Unauthorized)
//   - HTTP 403 with a message suggesting credentials/permission issues
//
// Returns false on:
//   - HTTP 403 rate-limit responses (those are transient, not auth issues)
//   - any non-HTTP error
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	var er *gh.ErrorResponse
	if !errors.As(err, &er) {
		return false
	}
	if er.Response == nil {
		return false
	}
	switch er.Response.StatusCode {
	case http.StatusUnauthorized:
		return true
	case http.StatusForbidden:
		msg := strings.ToLower(er.Message)
		// Rate-limit responses look like:
		//   "API rate limit exceeded for ..."
		// Auth/permission responses look like:
		//   "Bad credentials" / "Resource not accessible by integration" / "...permission..."
		if strings.Contains(msg, "rate limit") {
			return false
		}
		return strings.Contains(msg, "credentials") ||
			strings.Contains(msg, "not accessible") ||
			strings.Contains(msg, "permission")
	default:
		return false
	}
}
