package auth

import "context"

type ctxKey int

const claimsKey ctxKey = 0

// WithClaims returns a derived context that carries the given Claims.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// FromContext retrieves Claims attached by RequireAuth, if any.
func FromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	return c, ok && c != nil
}
