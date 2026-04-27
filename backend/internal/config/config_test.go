package config

import (
	"strings"
	"testing"
)

func TestMustValidateRequiresJobAuthKeyInProduction(t *testing.T) {
	cfg := validTestConfig()
	cfg.Environment = "production"
	cfg.JobAuthKey = ""

	err := cfg.MustValidate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "JOB_AUTH_KEY") {
		t.Fatalf("error = %q, want JOB_AUTH_KEY", err.Error())
	}
}

func TestMustValidateAllowsEmptyJobAuthKeyInDevelopment(t *testing.T) {
	cfg := validTestConfig()
	cfg.Environment = "development"
	cfg.JobAuthKey = ""

	if err := cfg.MustValidate(); err != nil {
		t.Fatalf("MustValidate() error = %v", err)
	}
}

func validTestConfig() *Config {
	return &Config{
		Environment:             "production",
		GitHubOAuthClientID:     "client-id",
		GitHubOAuthClientSecret: "client-secret",
		AuthJWTSecret:           "jwt-secret",
		JobAuthKey:              "job-secret",
		EncryptionKeyBase64:     "encryption-key",
		SyncIntervalMinutes:     60,
		SyncLockTTLMinutes:      10,
	}
}
