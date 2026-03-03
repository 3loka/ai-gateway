package github

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/trilok/dbt-weekly-report/internal/config"
	"golang.org/x/oauth2"
)

// RepoStats holds all the weekly stats for a single repository.
type RepoStats struct {
	RepoName        string
	Since           time.Time
	Until           time.Time
	OpenedIssues    []*github.Issue
	ClosedIssues    []*github.Issue
	OpenedPRs       []*github.PullRequest
	MergedPRs       []*github.PullRequest
	ClosedPRs       []*github.PullRequest
	Commits         []*github.RepositoryCommit
	NewReleases     []*github.RepositoryRelease
	Contributors    map[string]int // username -> count of contributions
	TotalStars      int
	TotalOpenIssues int
}

// TeamActivity holds a specific team's contributions across all repos.
type TeamActivity struct {
	TeamName    string
	Description string
	Members     []string
	// Per-member stats
	MemberStats map[string]*MemberStats
}

type MemberStats struct {
	Username     string
	IssuesOpened []*github.Issue
	IssuesClosed []*github.Issue
	PRsOpened    []*github.PullRequest
	PRsMerged    []*github.PullRequest
	PRsReviewed  int
	Commits      int
	ReposActive  map[string]bool // which repos they were active in
}

// Client wraps the GitHub API client.
type Client struct {
	gh     *github.Client
	config config.GitHubConfig
}

// NewClient creates a new GitHub API client.
func NewClient(cfg config.GitHubConfig) *Client {
	ctx := context.Background()
	var client *github.Client

	if cfg.Token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.Token})
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	} else {
		client = github.NewClient(nil)
	}

	return &Client{gh: client, config: cfg}
}

// FetchAllRepoStats fetches stats for all configured repositories.
func (c *Client) FetchAllRepoStats(ctx context.Context, since, until time.Time) ([]RepoStats, error) {
	var allStats []RepoStats
	for _, repo := range c.config.Repositories {
		log.Printf("Fetching stats for %s/%s...", repo.Owner, repo.Name)
		stats, err := c.fetchRepoStats(ctx, repo, since, until)
		if err != nil {
			log.Printf("Warning: failed to fetch stats for %s: %v", repo.FullName(), err)
			continue
		}
		allStats = append(allStats, *stats)
	}

	return allStats, nil
}

// ExtractTeamActivity extracts team-specific data from repo stats.
func ExtractTeamActivity(allStats []RepoStats, teams []config.TeamConfig) []TeamActivity {
	var results []TeamActivity

	for _, team := range teams {
		activity := TeamActivity{
			TeamName:    team.Name,
			Description: team.Description,
			Members:     team.Members,
			MemberStats: make(map[string]*MemberStats),
		}

		// Initialize member stats
		memberSet := make(map[string]bool)
		for _, m := range team.Members {
			lower := strings.ToLower(m)
			memberSet[lower] = true
			activity.MemberStats[lower] = &MemberStats{
				Username:    m,
				ReposActive: make(map[string]bool),
			}
		}

		// Scan all repo stats for team member activity
		for _, repoStats := range allStats {
			for _, issue := range repoStats.OpenedIssues {
				if user := issue.GetUser(); user != nil {
					login := strings.ToLower(user.GetLogin())
					if ms, ok := activity.MemberStats[login]; ok {
						ms.IssuesOpened = append(ms.IssuesOpened, issue)
						ms.ReposActive[repoStats.RepoName] = true
					}
				}
			}

			for _, issue := range repoStats.ClosedIssues {
				if assignee := issue.GetAssignee(); assignee != nil {
					login := strings.ToLower(assignee.GetLogin())
					if ms, ok := activity.MemberStats[login]; ok {
						ms.IssuesClosed = append(ms.IssuesClosed, issue)
						ms.ReposActive[repoStats.RepoName] = true
					}
				}
			}

			for _, pr := range repoStats.OpenedPRs {
				if user := pr.GetUser(); user != nil {
					login := strings.ToLower(user.GetLogin())
					if ms, ok := activity.MemberStats[login]; ok {
						ms.PRsOpened = append(ms.PRsOpened, pr)
						ms.ReposActive[repoStats.RepoName] = true
					}
				}
			}

			for _, pr := range repoStats.MergedPRs {
				if user := pr.GetUser(); user != nil {
					login := strings.ToLower(user.GetLogin())
					if ms, ok := activity.MemberStats[login]; ok {
						ms.PRsMerged = append(ms.PRsMerged, pr)
						ms.ReposActive[repoStats.RepoName] = true
					}
				}
			}

			for username, count := range repoStats.Contributors {
				login := strings.ToLower(username)
				if ms, ok := activity.MemberStats[login]; ok {
					ms.Commits += count
					ms.ReposActive[repoStats.RepoName] = true
				}
			}
		}

		results = append(results, activity)
	}

	return results
}

func (c *Client) fetchRepoStats(ctx context.Context, repo config.RepoConfig, since, until time.Time) (*RepoStats, error) {
	stats := &RepoStats{
		RepoName:     repo.FullName(),
		Since:        since,
		Until:        until,
		Contributors: make(map[string]int),
	}

	// Fetch repository info for star count and open issues
	repoInfo, _, err := c.gh.Repositories.Get(ctx, repo.Owner, repo.Name)
	if err != nil {
		return nil, fmt.Errorf("fetching repo info: %w", err)
	}
	stats.TotalStars = repoInfo.GetStargazersCount()
	stats.TotalOpenIssues = repoInfo.GetOpenIssuesCount()

	// Fetch issues created in the time window
	createdIssues, err := c.listIssues(ctx, repo, "created", since)
	if err != nil {
		return nil, fmt.Errorf("fetching created issues: %w", err)
	}
	for _, issue := range createdIssues {
		if issue.IsPullRequest() {
			continue
		}
		// Only count issues created within [since, until)
		t := issue.GetCreatedAt().Time
		if !t.Before(since) && t.Before(until) {
			stats.OpenedIssues = append(stats.OpenedIssues, issue)
		}
	}

	// Fetch issues closed/updated in the time window
	closedIssues, err := c.listIssues(ctx, repo, "updated", since)
	if err != nil {
		return nil, fmt.Errorf("fetching closed issues: %w", err)
	}
	for _, issue := range closedIssues {
		if issue.IsPullRequest() {
			continue
		}
		// Only count issues closed within [since, until)
		t := issue.GetClosedAt().Time
		if issue.GetState() == "closed" && !t.Before(since) && t.Before(until) {
			stats.ClosedIssues = append(stats.ClosedIssues, issue)
		}
	}

	// Fetch PRs
	if err := c.fetchPRs(ctx, repo, since, until, stats); err != nil {
		return nil, fmt.Errorf("fetching PRs: %w", err)
	}

	// Fetch commits on default branch
	commits, err := c.listCommits(ctx, repo, since)
	if err != nil {
		log.Printf("Warning: failed to fetch commits for %s: %v", repo.FullName(), err)
	} else {
		stats.Commits = commits
		for _, commit := range commits {
			if author := commit.GetAuthor(); author != nil {
				stats.Contributors[author.GetLogin()]++
			}
		}
	}

	// Fetch releases
	releases, err := c.listReleases(ctx, repo, since, until)
	if err != nil {
		log.Printf("Warning: failed to fetch releases for %s: %v", repo.FullName(), err)
	} else {
		stats.NewReleases = releases
	}

	return stats, nil
}

func (c *Client) listIssues(ctx context.Context, repo config.RepoConfig, sort string, since time.Time) ([]*github.Issue, error) {
	var allIssues []*github.Issue
	opts := &github.IssueListByRepoOptions{
		Since:     since,
		State:     "all",
		Sort:      sort,
		Direction: "desc",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		issues, resp, err := c.gh.Issues.ListByRepo(ctx, repo.Owner, repo.Name, opts)
		if err != nil {
			return nil, err
		}
		allIssues = append(allIssues, issues...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allIssues, nil
}

func (c *Client) fetchPRs(ctx context.Context, repo config.RepoConfig, since, until time.Time, stats *RepoStats) error {
	opts := &github.PullRequestListOptions{
		State:     "all",
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		prs, resp, err := c.gh.PullRequests.List(ctx, repo.Owner, repo.Name, opts)
		if err != nil {
			return err
		}

		reachedOlder := false
		for _, pr := range prs {
			if pr.GetUpdatedAt().Before(since) {
				reachedOlder = true
				break
			}
			ct := pr.GetCreatedAt().Time
			if !ct.Before(since) && ct.Before(until) {
				stats.OpenedPRs = append(stats.OpenedPRs, pr)
			}
			mt := pr.GetMergedAt().Time
			if !mt.IsZero() && !mt.Before(since) && mt.Before(until) {
				stats.MergedPRs = append(stats.MergedPRs, pr)
			} else if pr.GetState() == "closed" {
				clt := pr.GetClosedAt().Time
				if !clt.Before(since) && clt.Before(until) {
					stats.ClosedPRs = append(stats.ClosedPRs, pr)
				}
			}
		}

		if reachedOlder || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return nil
}

func (c *Client) listCommits(ctx context.Context, repo config.RepoConfig, since time.Time) ([]*github.RepositoryCommit, error) {
	var allCommits []*github.RepositoryCommit
	opts := &github.CommitsListOptions{
		Since: since,
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		commits, resp, err := c.gh.Repositories.ListCommits(ctx, repo.Owner, repo.Name, opts)
		if err != nil {
			return nil, err
		}
		allCommits = append(allCommits, commits...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allCommits, nil
}

func (c *Client) listReleases(ctx context.Context, repo config.RepoConfig, since, until time.Time) ([]*github.RepositoryRelease, error) {
	var recent []*github.RepositoryRelease
	opts := &github.ListOptions{PerPage: 20}

	releases, _, err := c.gh.Repositories.ListReleases(ctx, repo.Owner, repo.Name, opts)
	if err != nil {
		return nil, err
	}

	for _, r := range releases {
		t := r.GetPublishedAt().Time
		if !t.Before(since) && t.Before(until) {
			recent = append(recent, r)
		}
	}

	return recent, nil
}
