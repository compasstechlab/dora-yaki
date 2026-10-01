package handler

import (
	"reflect"
	"testing"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/domain/model"
)

func TestCalculateAllMemberStats(t *testing.T) {
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	firstCommit := base.Add(-24 * time.Hour)
	merged := base.Add(48 * time.Hour)

	alice := &model.TeamMember{ID: "1", Login: "alice"}
	bob := &model.TeamMember{ID: "2", Login: "bob"}
	carol := &model.TeamMember{ID: "3", Login: "carol"}

	prs := []*model.PullRequest{
		{
			Author: "alice", CreatedAt: base, FirstCommitAt: &firstCommit, MergedAt: &merged,
			Additions: 10, Deletions: 2,
			FileExtStats: []model.FileExtStats{
				{Extension: ".go", Additions: 8, Deletions: 2, Files: 2},
				{Extension: ".md", Additions: 2, Files: 1},
			},
		},
		{Author: "alice", CreatedAt: base, Additions: 5},
		{Author: "bob", CreatedAt: base, MergedAt: &merged, Additions: 1, Deletions: 1},
		{Author: "outsider", CreatedAt: base, Additions: 100},
	}
	reviews := []*model.Review{
		{Reviewer: "bob", State: "APPROVED", CommentsCount: 2},
		{Reviewer: "bob", State: "CHANGES_REQUESTED", CommentsCount: 1},
		{Reviewer: "alice", State: "COMMENTED"},
	}

	tests := []struct {
		name    string
		members []*model.TeamMember
	}{
		{name: "matches per-member calculation", members: []*model.TeamMember{alice, bob, carol}},
		{name: "keeps member order", members: []*model.TeamMember{carol, bob, alice}},
		{name: "no members", members: []*model.TeamMember{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateAllMemberStats(tt.members, prs, reviews)
			if got == nil {
				t.Fatal("calculateAllMemberStats() = nil, want non-nil slice")
			}
			if len(got) != len(tt.members) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.members))
			}
			for i, m := range tt.members {
				want := calculateMemberStats(m, prs, reviews)
				if !reflect.DeepEqual(got[i], want) {
					t.Errorf("stats[%d] (%s) = %+v, want %+v", i, m.Login, got[i], want)
				}
			}
		})
	}
}
