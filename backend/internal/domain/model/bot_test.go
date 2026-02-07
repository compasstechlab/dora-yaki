package model

import "testing"

func TestIsBot(t *testing.T) {
	customBots := []string{"renovate", "snyk-bot"}

	tests := []struct {
		name     string
		username string
		want     bool
	}{
		{"末尾[bot]はbot判定", "dependabot[bot]", true},
		{"末尾[bot]はbot判定2", "github-actions[bot]", true},
		{"通常ユーザーは非bot", "morikawa", false},
		{"カスタムbotリスト一致", "renovate", true},
		{"カスタムbotリスト一致2", "snyk-bot", true},
		{"カスタムbotリスト不一致", "unknown-user", false},
		{"空文字列は非bot", "", false},
		{"部分一致ではマッチしない", "renovate-extra", false},
		{"[bot]が途中にある場合は非bot", "user[bot]name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBot(tt.username, customBots)
			if got != tt.want {
				t.Errorf("IsBot(%q) = %v, want %v", tt.username, got, tt.want)
			}
		})
	}
}

func TestIsBot_EmptyCustomList(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     bool
	}{
		{"[bot]サフィックスのみで判定", "dependabot[bot]", true},
		{"通常ユーザー", "morikawa", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBot(tt.username, nil)
			if got != tt.want {
				t.Errorf("IsBot(%q, nil) = %v, want %v", tt.username, got, tt.want)
			}
		})
	}
}

func TestFilterPullRequestsByBot(t *testing.T) {
	customBots := []string{"renovate"}
	prs := []*PullRequest{
		{Author: "morikawa"},
		{Author: "dependabot[bot]"},
		{Author: "renovate"},
		{Author: "tanaka"},
	}

	tests := []struct {
		name        string
		excludeBots bool
		botsOnly    bool
		wantCount   int
		wantAuthors []string
	}{
		{"フィルタなし", false, false, 4, []string{"morikawa", "dependabot[bot]", "renovate", "tanaka"}},
		{"bot除外", true, false, 2, []string{"morikawa", "tanaka"}},
		{"botのみ", false, true, 2, []string{"dependabot[bot]", "renovate"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterPullRequestsByBot(prs, customBots, tt.excludeBots, tt.botsOnly)
			if len(got) != tt.wantCount {
				t.Errorf("got %d PRs, want %d", len(got), tt.wantCount)
			}
			for i, pr := range got {
				if i < len(tt.wantAuthors) && pr.Author != tt.wantAuthors[i] {
					t.Errorf("got author %q at index %d, want %q", pr.Author, i, tt.wantAuthors[i])
				}
			}
		})
	}
}

func TestFilterReviewsByBot(t *testing.T) {
	customBots := []string{"renovate"}
	reviews := []*Review{
		{Reviewer: "morikawa"},
		{Reviewer: "dependabot[bot]"},
		{Reviewer: "renovate"},
	}

	tests := []struct {
		name        string
		excludeBots bool
		botsOnly    bool
		wantCount   int
	}{
		{"フィルタなし", false, false, 3},
		{"bot除外", true, false, 1},
		{"botのみ", false, true, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterReviewsByBot(reviews, customBots, tt.excludeBots, tt.botsOnly)
			if len(got) != tt.wantCount {
				t.Errorf("got %d reviews, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestFilterTeamMembersByBot(t *testing.T) {
	customBots := []string{"renovate"}
	members := []*TeamMember{
		{Login: "morikawa"},
		{Login: "dependabot[bot]"},
		{Login: "renovate"},
	}

	tests := []struct {
		name        string
		excludeBots bool
		botsOnly    bool
		wantCount   int
	}{
		{"フィルタなし", false, false, 3},
		{"bot除外", true, false, 1},
		{"botのみ", false, true, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterTeamMembersByBot(members, customBots, tt.excludeBots, tt.botsOnly)
			if len(got) != tt.wantCount {
				t.Errorf("got %d members, want %d", len(got), tt.wantCount)
			}
		})
	}
}
