package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/datastore"
)

// CacheEntry represents an in-memory cache entry.
type CacheEntry struct {
	body        []byte
	contentType string
	statusCode  int
	createdAt   time.Time
}

// ResponseCache is a 3-tier cache: in-memory → Datastore → handler (live query).
type ResponseCache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	ttl     time.Duration
	ttlSec  int
	ds      *datastore.Client
	logger  *slog.Logger
}

// NewResponseCache creates a new 3-tier response cache.
func NewResponseCache(ttl time.Duration, ds *datastore.Client, logger *slog.Logger) *ResponseCache {
	rc := &ResponseCache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
		ttlSec:  int(ttl.Seconds()),
		ds:      ds,
		logger:  logger,
	}
	go rc.cleanup()
	return rc
}

// cleanup periodically removes expired in-memory entries.
func (rc *ResponseCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rc.mu.Lock()
		now := time.Now()
		for key, entry := range rc.entries {
			if now.Sub(entry.createdAt) > rc.ttl {
				delete(rc.entries, key)
			}
		}
		rc.mu.Unlock()
	}
}

// Invalidate clears both in-memory and Datastore caches.
func (rc *ResponseCache) Invalidate() {
	rc.mu.Lock()
	rc.entries = make(map[string]*CacheEntry)
	rc.mu.Unlock()

	// Delete Datastore cache asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := rc.ds.DeleteAllMetricsCache(ctx); err != nil {
			rc.logger.Warn("failed to delete datastore cache", "error", err)
		} else {
			rc.logger.Info("datastore metrics cache invalidated")
		}
	}()
}

// getFromMemory retrieves an entry from the in-memory cache.
func (rc *ResponseCache) getFromMemory(key string) (*CacheEntry, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	entry, ok := rc.entries[key]
	if !ok || time.Since(entry.createdAt) > rc.ttl {
		return nil, false
	}
	return entry, true
}

// getFromDatastore retrieves from Datastore cache and promotes to in-memory on hit.
func (rc *ResponseCache) getFromDatastore(ctx context.Context, key string) (*CacheEntry, bool) {
	body, err := rc.ds.GetMetricsCache(ctx, key)
	if err != nil {
		return nil, false
	}

	// Restore from Datastore and promote to in-memory
	entry := &CacheEntry{
		body:        body,
		contentType: "application/json",
		statusCode:  http.StatusOK,
		createdAt:   time.Now(),
	}

	rc.mu.Lock()
	rc.entries[key] = entry
	rc.mu.Unlock()

	return entry, true
}

// storeAll stores in both in-memory and Datastore caches.
func (rc *ResponseCache) storeAll(ctx context.Context, key string, cw *cacheWriter) {
	bodyBytes := cw.body.Bytes()
	contentType := cw.Header().Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}

	// Store in memory
	rc.mu.Lock()
	rc.entries[key] = &CacheEntry{
		body:        bodyBytes,
		contentType: contentType,
		statusCode:  cw.statusCode,
		createdAt:   time.Now(),
	}
	rc.mu.Unlock()

	// Store in Datastore asynchronously
	go func() {
		dsCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := rc.ds.PutMetricsCache(dsCtx, key, bodyBytes, rc.ttlSec); err != nil {
			rc.logger.Warn("failed to store datastore cache", "key", key, "error", err)
		}
	}()
}

// Middleware returns a 3-tier cache middleware.
func (rc *ResponseCache) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only cache GET requests
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			// Bypass cache when refresh=true
			if r.URL.Query().Get("refresh") == "true" {
				q := r.URL.Query()
				q.Del("refresh")
				r.URL.RawQuery = q.Encode()

				cw := &cacheWriter{ResponseWriter: w, body: &bytes.Buffer{}}
				next.ServeHTTP(cw, r)
				if cw.statusCode >= 200 && cw.statusCode < 300 {
					rc.storeAll(r.Context(), r.URL.RequestURI(), cw)
				}
				w.Header().Set("X-Cache", "BYPASS")
				return
			}

			key := r.URL.RequestURI()

			// Stage 1: in-memory cache
			if entry, ok := rc.getFromMemory(key); ok {
				w.Header().Set("Content-Type", entry.contentType)
				w.Header().Set("X-Cache", "HIT-MEMORY")
				w.WriteHeader(entry.statusCode)
				_, _ = w.Write(entry.body)
				return
			}

			// Stage 2: Datastore cache
			if entry, ok := rc.getFromDatastore(r.Context(), key); ok {
				w.Header().Set("Content-Type", entry.contentType)
				w.Header().Set("X-Cache", "HIT-DATASTORE")
				w.WriteHeader(entry.statusCode)
				_, _ = w.Write(entry.body)
				return
			}

			// Stage 3: handler (live Datastore query)
			cw := &cacheWriter{ResponseWriter: w, body: &bytes.Buffer{}}
			next.ServeHTTP(cw, r)

			// Only cache 2xx responses in both tiers
			if cw.statusCode >= 200 && cw.statusCode < 300 {
				rc.storeAll(r.Context(), key, cw)
			}

			w.Header().Set("X-Cache", "MISS")
		})
	}
}

// cacheWriter captures responses while also writing to the original ResponseWriter.
type cacheWriter struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (cw *cacheWriter) WriteHeader(code int) {
	cw.statusCode = code
	cw.ResponseWriter.WriteHeader(code)
}

func (cw *cacheWriter) Write(b []byte) (int, error) {
	cw.body.Write(b)
	return cw.ResponseWriter.Write(b)
}
