package model

import "time"

// RepositoryAccess records whether a User may view a Repository's data and
// whether their token participates in the sync rotation pool for that repo.
//
// Composite key form: userID + "_" + repoID.
type RepositoryAccess struct {
	UserID           string     `json:"userId" datastore:"user_id"`
	RepositoryID     string     `json:"repositoryId" datastore:"repository_id"`
	CanAccess        bool       `json:"canAccess" datastore:"can_access"`
	IsTokenCandidate bool       `json:"isTokenCandidate" datastore:"is_token_candidate"`
	CheckedAt        time.Time  `json:"checkedAt" datastore:"checked_at"`
	LastUsedAt       *time.Time `json:"lastUsedAt,omitempty" datastore:"last_used_at"`
	MarkedInvalidAt  *time.Time `json:"markedInvalidAt,omitempty" datastore:"marked_invalid_at"`
}

// RepositoryAccessKey returns the composite Datastore key for a (user, repo) pair.
func RepositoryAccessKey(userID, repositoryID string) string {
	return userID + "_" + repositoryID
}
