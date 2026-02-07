package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
)

// Config holds the application configuration
type Config struct {
	Port                string
	Environment         string
	GCPProjectID        string
	GitHubToken         string
	TZOffset            string // Timezone offset (e.g. "+09:00", "-05:30")
	SyncIntervalMinutes int    // Sync interval in minutes (default: 60)
	SyncLockTTLMinutes  int    // Lock TTL in minutes (default: 10)
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Port:                getEnv("PORT", "7202"),
		Environment:         getEnv("ENVIRONMENT", "development"),
		GCPProjectID:        resolveProjectID(),
		GitHubToken:         getEnv("GITHUB_TOKEN", ""),
		TZOffset:            getEnv("TZ_OFFSET", ""),
		SyncIntervalMinutes: getEnvInt("SYNC_INTERVAL_MINUTES", 60),
		SyncLockTTLMinutes:  getEnvInt("SYNC_LOCK_TTL_MINUTES", 10),
	}
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// Location generates a time.Location from TZOffset.
// TZOffset から time.Location を生成する。空の場合は UTC を返す。
func (c *Config) Location() *time.Location {
	if c.TZOffset == "" {
		return time.UTC
	}
	loc, err := parseTZOffset(c.TZOffset)
	if err != nil {
		return time.UTC
	}
	return loc
}

// parseTZOffset parses an offset in "+09:00" or "-05:30" format.
// "+09:00" や "-05:30" 形式のオフセットをパースする。
func parseTZOffset(offset string) (*time.Location, error) {
	if len(offset) < 5 {
		return nil, fmt.Errorf("invalid TZ_OFFSET format: %s", offset)
	}

	sign := 1
	switch offset[0] {
	case '+':
		// default (positive)
	case '-':
		sign = -1
	default:
		return nil, fmt.Errorf("invalid TZ_OFFSET sign: %s", offset)
	}

	parts := strings.SplitN(offset[1:], ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid TZ_OFFSET format: %s", offset)
	}

	var hours, minutes int
	if _, err := fmt.Sscanf(parts[0], "%d", &hours); err != nil {
		return nil, fmt.Errorf("invalid TZ_OFFSET hours: %w", err)
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &minutes); err != nil {
		return nil, fmt.Errorf("invalid TZ_OFFSET minutes: %w", err)
	}

	totalSeconds := sign * (hours*3600 + minutes*60)
	name := "UTC" + offset
	return time.FixedZone(name, totalSeconds), nil
}

// resolveProjectID resolves the GCP project ID.
// 環境変数 → メタデータサーバー(Cloud Run/GCE) の順で取得を試みる。
func resolveProjectID() string {
	if id := os.Getenv("GCP_PROJECT_ID"); id != "" {
		return id
	}
	if metadata.OnGCE() {
		if id, err := metadata.ProjectIDWithContext(context.Background()); err == nil && id != "" {
			return id
		}
	}
	return ""
}

// SyncInterval returns the sync interval as a time.Duration.
// 同期間隔を time.Duration で返す。
func (c *Config) SyncInterval() time.Duration {
	return time.Duration(c.SyncIntervalMinutes) * time.Minute
}

// SyncLockTTL returns the lock TTL as a time.Duration.
// ロックTTLを time.Duration で返す。
func (c *Config) SyncLockTTL() time.Duration {
	return time.Duration(c.SyncLockTTLMinutes) * time.Minute
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if v, err := strconv.Atoi(value); err == nil {
			return v
		}
	}
	return defaultValue
}
