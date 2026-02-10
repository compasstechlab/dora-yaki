package handler

import (
	"log/slog"
	"testing"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/config"
	"github.com/compasstechlab/dora-yaki/internal/domain/model"
)

// newTestJobHandler creates a JobHandler with the given sync interval for testing.
func newTestJobHandler(syncIntervalMin int) *JobHandler {
	return &JobHandler{
		logger: slog.Default(),
		cfg: &config.Config{
			SyncIntervalMinutes: syncIntervalMin,
		},
	}
}

func TestPickSyncTarget(t *testing.T) {
	now := time.Now()
	hourAgo := now.Add(-1 * time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)

	tests := []struct {
		name     string
		repos    []*model.Repository
		req      jobSyncRequest
		interval int // SyncInterval in minutes
		wantName string
		wantNil  bool
	}{
		{
			name: "never-synced repository has highest priority",
			repos: []*model.Repository{
				{FullName: "org/synced", LastSyncedAt: &twoHoursAgo},
				{FullName: "org/never-synced"},
			},
			req:      jobSyncRequest{Range: "day"},
			interval: 30,
			wantName: "org/never-synced",
		},
		{
			name: "repository with oldest LastSyncedAt is preferred",
			repos: []*model.Repository{
				{FullName: "org/recent", LastSyncedAt: &hourAgo},
				{FullName: "org/old", LastSyncedAt: &twoHoursAgo},
			},
			req:      jobSyncRequest{Range: "day"},
			interval: 30,
			wantName: "org/old",
		},
		{
			name: "skip repository when interval has not elapsed",
			repos: []*model.Repository{
				{FullName: "org/recent", LastSyncedAt: &now},
			},
			req:      jobSyncRequest{Range: "day"},
			interval: 30,
			wantNil:  true,
		},
		{
			name: "skip repository within processStartGuard",
			repos: []*model.Repository{
				{FullName: "org/processing", ProcessStartAt: &now},
			},
			req:      jobSyncRequest{Range: "day"},
			interval: 0, // interval check passes (LastSyncedAt=nil)
			wantNil:  true,
		},
		{
			name: "repo param matches by FullName",
			repos: []*model.Repository{
				{FullName: "org/repo-a", Name: "repo-a", LastSyncedAt: &twoHoursAgo},
				{FullName: "org/repo-b", Name: "repo-b", LastSyncedAt: &twoHoursAgo},
			},
			req:      jobSyncRequest{Range: "day", Repo: "org/repo-b"},
			interval: 30,
			wantName: "org/repo-b",
		},
		{
			name: "repo param matches by Name",
			repos: []*model.Repository{
				{FullName: "org/repo-a", Name: "repo-a", LastSyncedAt: &twoHoursAgo},
				{FullName: "org/repo-b", Name: "repo-b", LastSyncedAt: &twoHoursAgo},
			},
			req:      jobSyncRequest{Range: "day", Repo: "repo-a"},
			interval: 30,
			wantName: "org/repo-a",
		},
		{
			name: "skip specified repo when interval has not elapsed",
			repos: []*model.Repository{
				{FullName: "org/repo-a", Name: "repo-a", LastSyncedAt: &now},
			},
			req:      jobSyncRequest{Range: "day", Repo: "org/repo-a"},
			interval: 30,
			wantNil:  true,
		},
		{
			name: "force=true bypasses processStartGuard for specified repo",
			repos: []*model.Repository{
				{FullName: "org/repo-a", Name: "repo-a", ProcessStartAt: &now, LastSyncedAt: &twoHoursAgo},
			},
			req:      jobSyncRequest{Range: "day", Repo: "org/repo-a", Force: true},
			interval: 30,
			wantName: "org/repo-a",
		},
		{
			name:     "return nil when repos is empty",
			repos:    []*model.Repository{},
			req:      jobSyncRequest{Range: "day"},
			interval: 30,
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestJobHandler(tt.interval)
			got := h.pickSyncTarget(tt.repos, tt.req)

			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %s", got.FullName)
				}
				return
			}

			if got == nil {
				t.Fatalf("expected %s, got nil", tt.wantName)
			}
			if got.FullName != tt.wantName {
				t.Errorf("expected %s, got %s", tt.wantName, got.FullName)
			}
		})
	}
}
