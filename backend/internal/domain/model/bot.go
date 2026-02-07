package model

import "strings"

// IsBot determines whether a username is a bot.
func IsBot(username string, customBotUsernames []string) bool {
	if strings.HasSuffix(username, "[bot]") {
		return true
	}
	for _, bot := range customBotUsernames {
		if username == bot {
			return true
		}
	}
	return false
}

// filterByBot is the generic bot filtering logic.
// ボットフィルタリングの共通ロジック
func filterByBot[T any](items []T, customBotUsernames []string, excludeBots, botsOnly bool, getUsername func(T) string) []T {
	if !excludeBots && !botsOnly {
		return items
	}
	result := make([]T, 0, len(items))
	for _, item := range items {
		isBot := IsBot(getUsername(item), customBotUsernames)
		if botsOnly && isBot {
			result = append(result, item)
		} else if excludeBots && !isBot {
			result = append(result, item)
		}
	}
	return result
}

// FilterPullRequestsByBot filters PR list by bot criteria.
func FilterPullRequestsByBot(prs []*PullRequest, customBotUsernames []string, excludeBots, botsOnly bool) []*PullRequest {
	return filterByBot(prs, customBotUsernames, excludeBots, botsOnly, func(pr *PullRequest) string { return pr.Author })
}

// FilterReviewsByBot filters review list by bot criteria.
func FilterReviewsByBot(reviews []*Review, customBotUsernames []string, excludeBots, botsOnly bool) []*Review {
	return filterByBot(reviews, customBotUsernames, excludeBots, botsOnly, func(r *Review) string { return r.Reviewer })
}

// FilterTeamMembersByBot filters team member list by bot criteria.
func FilterTeamMembersByBot(members []*TeamMember, customBotUsernames []string, excludeBots, botsOnly bool) []*TeamMember {
	return filterByBot(members, customBotUsernames, excludeBots, botsOnly, func(m *TeamMember) string { return m.Login })
}
