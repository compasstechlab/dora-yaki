package github

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v82/github"
	"golang.org/x/oauth2"

	"github.com/compasstechlab/dora-yaki/internal/domain/model"
)

// Client wraps the GitHub API client
type Client struct {
	client *github.Client
}

// NewClient creates a new GitHub API client
func NewClient(token string) *Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		client: github.NewClient(tc),
	}
}

// NewClientWithHTTPClient creates a new GitHub client with a custom HTTP client
func NewClientWithHTTPClient(httpClient *http.Client) *Client {
	return &Client{
		client: github.NewClient(httpClient),
	}
}

// GetRepository fetches repository information
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*model.Repository, error) {
	r, _, err := c.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return &model.Repository{
		ID:        fmt.Sprintf("%d", r.GetID()),
		Owner:     r.GetOwner().GetLogin(),
		Name:      r.GetName(),
		FullName:  r.GetFullName(),
		Private:   r.GetPrivate(),
		CreatedAt: r.GetCreatedAt().Time,
		UpdatedAt: r.GetUpdatedAt().Time,
	}, nil
}

// ListPullRequests fetches pull requests for a repository
func (c *Client) ListPullRequests(ctx context.Context, owner, repo string, opts *PullRequestListOptions) ([]*model.PullRequest, error) {
	ghOpts := &github.PullRequestListOptions{
		State:     opts.State,
		Sort:      opts.Sort,
		Direction: opts.Direction,
		ListOptions: github.ListOptions{
			Page:    opts.Page,
			PerPage: opts.PerPage,
		},
	}

	prs, _, err := c.client.PullRequests.List(ctx, owner, repo, ghOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests: %w", err)
	}

	result := make([]*model.PullRequest, 0, len(prs))
	for _, pr := range prs {
		result = append(result, c.convertPullRequest(pr, owner, repo))
	}

	return result, nil
}

// GetPullRequest fetches a specific pull request
func (c *Client) GetPullRequest(ctx context.Context, owner, repo string, number int) (*model.PullRequest, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}

	return c.convertPullRequest(pr, owner, repo), nil
}

// ListPullRequestCommits fetches commits for a pull request
func (c *Client) ListPullRequestCommits(ctx context.Context, owner, repo string, number int) ([]*github.RepositoryCommit, error) {
	commits, _, err := c.client.PullRequests.ListCommits(ctx, owner, repo, number, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull request commits: %w", err)
	}
	return commits, nil
}

// ListPullRequestReviews fetches reviews for a pull request
func (c *Client) ListPullRequestReviews(ctx context.Context, owner, repo string, number int, repositoryID string) ([]*model.Review, error) {
	reviews, _, err := c.client.PullRequests.ListReviews(ctx, owner, repo, number, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull request reviews: %w", err)
	}

	prID := fmt.Sprintf("%s#%d", repositoryID, number)

	result := make([]*model.Review, 0, len(reviews))
	for _, review := range reviews {
		result = append(result, &model.Review{
			ID:            fmt.Sprintf("%d", review.GetID()),
			PullRequestID: prID,
			RepositoryID:  repositoryID,
			Reviewer:      review.GetUser().GetLogin(),
			State:         review.GetState(),
			Body:          review.GetBody(),
			SubmittedAt:   review.GetSubmittedAt().Time,
		})
	}

	return result, nil
}

// ListReviewComments fetches review comments for a pull request
func (c *Client) ListReviewComments(ctx context.Context, owner, repo string, number int) ([]*github.PullRequestComment, error) {
	comments, _, err := c.client.PullRequests.ListComments(ctx, owner, repo, number, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list review comments: %w", err)
	}
	return comments, nil
}

// ListReleases fetches releases for a repository
func (c *Client) ListReleases(ctx context.Context, owner, repo string, opts *ListOptions) ([]*github.RepositoryRelease, error) {
	ghOpts := &github.ListOptions{
		Page:    opts.Page,
		PerPage: opts.PerPage,
	}

	releases, _, err := c.client.Repositories.ListReleases(ctx, owner, repo, ghOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}

	return releases, nil
}

// ListDeployments fetches deployments for a repository
func (c *Client) ListDeployments(ctx context.Context, owner, repo string, opts *DeploymentListOptions, repositoryID string) ([]*model.Deployment, error) {
	ghOpts := &github.DeploymentsListOptions{
		Environment: opts.Environment,
		ListOptions: github.ListOptions{
			Page:    opts.Page,
			PerPage: opts.PerPage,
		},
	}

	deployments, _, err := c.client.Repositories.ListDeployments(ctx, owner, repo, ghOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	result := make([]*model.Deployment, 0, len(deployments))
	for _, d := range deployments {
		result = append(result, &model.Deployment{
			ID:           fmt.Sprintf("%d", d.GetID()),
			RepositoryID: repositoryID,
			Environment:  d.GetEnvironment(),
			Ref:          d.GetRef(),
			SHA:          d.GetSHA(),
			Status:       "pending",
			CreatedAt:    d.GetCreatedAt().Time,
		})
	}

	return result, nil
}

// ListContributors fetches contributors for a repository
func (c *Client) ListContributors(ctx context.Context, owner, repo string) ([]*model.TeamMember, error) {
	contributors, _, err := c.client.Repositories.ListContributors(ctx, owner, repo, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list contributors: %w", err)
	}

	result := make([]*model.TeamMember, 0, len(contributors))
	for _, contributor := range contributors {
		result = append(result, &model.TeamMember{
			ID:        fmt.Sprintf("%d", contributor.GetID()),
			Login:     contributor.GetLogin(),
			AvatarURL: contributor.GetAvatarURL(),
		})
	}

	return result, nil
}

// GitHubUser represents authenticated user information.
type GitHubUser struct {
	Login     string   `json:"login"`
	Name      string   `json:"name"`
	AvatarURL string   `json:"avatarUrl"`
	Orgs      []string `json:"orgs"`
}

// GetAuthenticatedUser returns authenticated user info and org list for the token.
func (c *Client) GetAuthenticatedUser(ctx context.Context) (*GitHubUser, error) {
	user, _, err := c.client.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated user: %w", err)
	}

	result := &GitHubUser{
		Login:     user.GetLogin(),
		Name:      user.GetName(),
		AvatarURL: user.GetAvatarURL(),
	}

	// Get org list
	orgs, _, err := c.client.Organizations.List(ctx, "", nil)
	if err != nil {
		// Return user info even if org retrieval fails
		return result, nil
	}
	for _, org := range orgs {
		result.Orgs = append(result.Orgs, org.GetLogin())
	}

	return result, nil
}

// GetRateLimit returns the current rate limit status
func (c *Client) GetRateLimit(ctx context.Context) (*github.RateLimits, error) {
	limits, _, err := c.client.RateLimit.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit: %w", err)
	}
	return limits, nil
}

func (c *Client) convertPullRequest(pr *github.PullRequest, owner, repo string) *model.PullRequest {
	// Use GitHub numeric ID for repository ID (to match Repository entity)
	var repoID string
	if base := pr.GetBase(); base != nil && base.GetRepo() != nil && base.GetRepo().GetID() != 0 {
		repoID = fmt.Sprintf("%d", base.GetRepo().GetID())
	} else {
		// Fallback: owner/repo format (when Base info is missing from List API)
		repoID = fmt.Sprintf("%s/%s", owner, repo)
	}

	result := &model.PullRequest{
		ID:           fmt.Sprintf("%d", pr.GetID()),
		RepositoryID: repoID,
		Number:       pr.GetNumber(),
		Title:        pr.GetTitle(),
		Author:       pr.GetUser().GetLogin(),
		State:        pr.GetState(),
		Draft:        pr.GetDraft(),
		CreatedAt:    pr.GetCreatedAt().Time,
		UpdatedAt:    pr.GetUpdatedAt().Time,
		Additions:    pr.GetAdditions(),
		Deletions:    pr.GetDeletions(),
		ChangedFiles: pr.GetChangedFiles(),
		CommitCount:  pr.GetCommits(),
	}

	if pr.MergedAt != nil {
		t := pr.GetMergedAt().Time
		result.MergedAt = &t
	}

	if pr.ClosedAt != nil {
		t := pr.GetClosedAt().Time
		result.ClosedAt = &t
	}

	return result
}

// PullRequestListOptions options for listing pull requests
type PullRequestListOptions struct {
	State     string
	Sort      string
	Direction string
	Page      int
	PerPage   int
}

// ListOptions generic list options
type ListOptions struct {
	Page    int
	PerPage int
}

// DeploymentListOptions options for listing deployments
type DeploymentListOptions struct {
	Environment string
	Page        int
	PerPage     int
}

// OrgRepo is a lightweight struct for displaying org repositories.
type OrgRepo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"fullName"`
	Owner       string `json:"owner"`
	Private     bool   `json:"private"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Archived    bool   `json:"archived"`
}

// OrgRepoListOptions holds options for listing org repositories.
type OrgRepoListOptions struct {
	Type    string // all, public, private
	PerPage int
}

// ListOwnerRepos lists repositories belonging to an org or user.
func (c *Client) ListOwnerRepos(ctx context.Context, owner string, opts *OrgRepoListOptions) ([]*OrgRepo, error) {
	repos, err := c.listByOrg(ctx, owner, opts)
	if err == nil {
		return repos, nil
	}

	// Fall back to user if org retrieval fails
	repos, userErr := c.listByUser(ctx, owner, opts)
	if userErr != nil {
		return nil, fmt.Errorf("failed to list repos for %q (tried org and user): org=%w, user=%v", owner, err, userErr)
	}
	return repos, nil
}

func (c *Client) listByOrg(ctx context.Context, org string, opts *OrgRepoListOptions) ([]*OrgRepo, error) {
	repoType, perPage := resolveListOpts(opts)

	var allRepos []*OrgRepo
	ghOpts := &github.RepositoryListByOrgOptions{
		Type: repoType,
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: perPage,
		},
	}

	for {
		repos, resp, err := c.client.Repositories.ListByOrg(ctx, org, ghOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to list organization repos: %w", err)
		}

		allRepos = append(allRepos, convertGHRepos(repos)...)

		if resp.NextPage == 0 {
			break
		}
		ghOpts.Page = resp.NextPage
	}

	return allRepos, nil
}

func (c *Client) listByUser(ctx context.Context, user string, opts *OrgRepoListOptions) ([]*OrgRepo, error) {
	repoType, perPage := resolveListOpts(opts)

	var allRepos []*OrgRepo
	ghOpts := &github.RepositoryListByUserOptions{
		Type: repoType,
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: perPage,
		},
	}

	for {
		repos, resp, err := c.client.Repositories.ListByUser(ctx, user, ghOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to list user repos: %w", err)
		}

		allRepos = append(allRepos, convertGHRepos(repos)...)

		if resp.NextPage == 0 {
			break
		}
		ghOpts.Page = resp.NextPage
	}

	return allRepos, nil
}

func resolveListOpts(opts *OrgRepoListOptions) (repoType string, perPage int) {
	repoType = "all"
	perPage = 100
	if opts != nil {
		if opts.Type != "" {
			repoType = opts.Type
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}
	return repoType, perPage
}

func convertGHRepos(repos []*github.Repository) []*OrgRepo {
	result := make([]*OrgRepo, 0, len(repos))
	for _, r := range repos {
		result = append(result, &OrgRepo{
			ID:          r.GetID(),
			Name:        r.GetName(),
			FullName:    r.GetFullName(),
			Owner:       r.GetOwner().GetLogin(),
			Private:     r.GetPrivate(),
			Description: r.GetDescription(),
			Language:    r.GetLanguage(),
			Archived:    r.GetArchived(),
		})
	}
	return result
}

// ListPullRequestFiles retrieves the list of changed files in a PR.
func (c *Client) ListPullRequestFiles(ctx context.Context, owner, repo string, number int) ([]*github.CommitFile, error) {
	var allFiles []*github.CommitFile
	opts := &github.ListOptions{Page: 1, PerPage: 100}

	for {
		files, resp, err := c.client.PullRequests.ListFiles(ctx, owner, repo, number, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list pull request files: %w", err)
		}

		allFiles = append(allFiles, files...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allFiles, nil
}

// GetFirstCommitTime fetches the first commit time for a PR
func (c *Client) GetFirstCommitTime(ctx context.Context, owner, repo string, prNumber int) (*time.Time, error) {
	commits, err := c.ListPullRequestCommits(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, err
	}

	if len(commits) == 0 {
		return nil, nil
	}

	var firstCommitTime *time.Time
	for _, commit := range commits {
		if commit.Commit != nil && commit.Commit.Author != nil {
			t := commit.Commit.Author.GetDate().Time
			if firstCommitTime == nil || t.Before(*firstCommitTime) {
				firstCommitTime = &t
			}
		}
	}

	return firstCommitTime, nil
}
