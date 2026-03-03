package github

import (
	"fmt"
	"strings"
	"time"
)

// WeeklySummaryData is a structured representation of the week's activity
// that gets passed to the LLM for narrative generation.
type WeeklySummaryData struct {
	Period       string
	RepoSummaries []RepoSummaryData
}

type RepoSummaryData struct {
	RepoName        string
	IssuesOpened    int
	IssuesClosed    int
	PRsOpened       int
	PRsMerged       int
	PRsClosed       int
	Commits         int
	Releases        int
	TopContributors []string
	TotalStars      int
	TotalOpenIssues int
	// Key issue/PR titles for context
	NotableIssues []string
	NotablePRs    []string
	ReleaseNames  []string
}

// BuildSummaryData converts raw stats into structured summary data.
func BuildSummaryData(allStats []RepoStats) WeeklySummaryData {
	if len(allStats) == 0 {
		return WeeklySummaryData{}
	}

	summary := WeeklySummaryData{
		Period: fmt.Sprintf("%s to %s",
			allStats[0].Since.Format("2006-01-02"),
			allStats[0].Until.Format("2006-01-02")),
	}

	for _, s := range allStats {
		repo := RepoSummaryData{
			RepoName:        s.RepoName,
			IssuesOpened:    len(s.OpenedIssues),
			IssuesClosed:    len(s.ClosedIssues),
			PRsOpened:       len(s.OpenedPRs),
			PRsMerged:       len(s.MergedPRs),
			PRsClosed:       len(s.ClosedPRs),
			Commits:         len(s.Commits),
			Releases:        len(s.NewReleases),
			TotalStars:      s.TotalStars,
			TotalOpenIssues: s.TotalOpenIssues,
		}

		// Top contributors (up to 5)
		type kv struct {
			Key   string
			Value int
		}
		var sorted []kv
		for k, v := range s.Contributors {
			sorted = append(sorted, kv{k, v})
		}
		// Simple sort
		for i := 0; i < len(sorted); i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[j].Value > sorted[i].Value {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
		for i, kv := range sorted {
			if i >= 5 {
				break
			}
			repo.TopContributors = append(repo.TopContributors,
				fmt.Sprintf("%s (%d commits)", kv.Key, kv.Value))
		}

		// Notable issues (up to 10)
		for i, issue := range s.OpenedIssues {
			if i >= 10 {
				break
			}
			labels := []string{}
			for _, l := range issue.Labels {
				labels = append(labels, l.GetName())
			}
			labelStr := ""
			if len(labels) > 0 {
				labelStr = fmt.Sprintf(" [%s]", strings.Join(labels, ", "))
			}
			repo.NotableIssues = append(repo.NotableIssues,
				fmt.Sprintf("#%d: %s%s", issue.GetNumber(), issue.GetTitle(), labelStr))
		}

		// Notable PRs (up to 10)
		for i, pr := range s.MergedPRs {
			if i >= 10 {
				break
			}
			repo.NotablePRs = append(repo.NotablePRs,
				fmt.Sprintf("#%d: %s (by @%s)", pr.GetNumber(), pr.GetTitle(), pr.GetUser().GetLogin()))
		}

		// Releases
		for _, r := range s.NewReleases {
			repo.ReleaseNames = append(repo.ReleaseNames, r.GetTagName())
		}

		summary.RepoSummaries = append(summary.RepoSummaries, repo)
	}

	return summary
}

// ToPromptText converts the summary data into a text block suitable for an LLM prompt.
func (s WeeklySummaryData) ToPromptText() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("=== dbt Ecosystem Activity Report ===\n"))
	b.WriteString(fmt.Sprintf("Period: %s\n\n", s.Period))

	for _, repo := range s.RepoSummaries {
		b.WriteString(fmt.Sprintf("--- %s ---\n", repo.RepoName))
		b.WriteString(fmt.Sprintf("Stars: %d | Total Open Issues: %d\n", repo.TotalStars, repo.TotalOpenIssues))
		b.WriteString(fmt.Sprintf("Issues opened: %d | Issues closed: %d\n", repo.IssuesOpened, repo.IssuesClosed))
		b.WriteString(fmt.Sprintf("PRs opened: %d | PRs merged: %d | PRs closed without merge: %d\n",
			repo.PRsOpened, repo.PRsMerged, repo.PRsClosed))
		b.WriteString(fmt.Sprintf("Commits: %d | Releases: %d\n", repo.Commits, repo.Releases))

		if len(repo.TopContributors) > 0 {
			b.WriteString(fmt.Sprintf("Top contributors: %s\n", strings.Join(repo.TopContributors, ", ")))
		}
		if len(repo.ReleaseNames) > 0 {
			b.WriteString(fmt.Sprintf("New releases: %s\n", strings.Join(repo.ReleaseNames, ", ")))
		}
		if len(repo.NotableIssues) > 0 {
			b.WriteString("Notable new issues:\n")
			for _, issue := range repo.NotableIssues {
				b.WriteString(fmt.Sprintf("  - %s\n", issue))
			}
		}
		if len(repo.NotablePRs) > 0 {
			b.WriteString("Notable merged PRs:\n")
			for _, pr := range repo.NotablePRs {
				b.WriteString(fmt.Sprintf("  - %s\n", pr))
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// WeekBucket holds aggregated PR and issue counts for a single week.
type WeekBucket struct {
	WeekStart    time.Time
	Label        string
	IssuesOpened int
	IssuesClosed int
	PRsOpened    int
	PRsMerged    int
}

// RepoTrend holds weekly-bucketed activity for a single repository.
type RepoTrend struct {
	RepoName string
	Weeks    []WeekBucket
}

// BucketByWeek slices allStats into weekly buckets for trend charts.
func BucketByWeek(allStats []RepoStats) []RepoTrend {
	if len(allStats) == 0 {
		return nil
	}

	since := allStats[0].Since
	until := allStats[0].Until

	var weekStarts []time.Time
	for t := since; t.Before(until); t = t.AddDate(0, 0, 7) {
		weekStarts = append(weekStarts, t)
	}
	if len(weekStarts) == 0 {
		return nil
	}

	inBucket := func(t time.Time, ws time.Time) bool {
		return !t.IsZero() && !t.Before(ws) && t.Before(ws.AddDate(0, 0, 7))
	}

	var trends []RepoTrend
	for _, stats := range allStats {
		trend := RepoTrend{RepoName: stats.RepoName}
		for _, ws := range weekStarts {
			b := WeekBucket{WeekStart: ws, Label: ws.Format("Jan 2")}
			for _, issue := range stats.OpenedIssues {
				if inBucket(issue.GetCreatedAt().Time, ws) {
					b.IssuesOpened++
				}
			}
			for _, issue := range stats.ClosedIssues {
				if inBucket(issue.GetClosedAt().Time, ws) {
					b.IssuesClosed++
				}
			}
			for _, pr := range stats.OpenedPRs {
				if inBucket(pr.GetCreatedAt().Time, ws) {
					b.PRsOpened++
				}
			}
			for _, pr := range stats.MergedPRs {
				if inBucket(pr.GetMergedAt().Time, ws) {
					b.PRsMerged++
				}
			}
			trend.Weeks = append(trend.Weeks, b)
		}
		trends = append(trends, trend)
	}
	return trends
}

// TeamActivityToPromptText converts team activity data into text for LLM prompts.
func TeamActivityToPromptText(teams []TeamActivity) string {
	var b strings.Builder

	for _, team := range teams {
		b.WriteString(fmt.Sprintf("=== Team Callout: %s ===\n", team.TeamName))
		if team.Description != "" {
			b.WriteString(fmt.Sprintf("Context: %s\n", team.Description))
		}
		b.WriteString(fmt.Sprintf("Members: %s\n\n", strings.Join(team.Members, ", ")))

		for _, member := range team.Members {
			lower := strings.ToLower(member)
			ms, ok := team.MemberStats[lower]
			if !ok {
				continue
			}

			b.WriteString(fmt.Sprintf("  @%s:\n", ms.Username))
			b.WriteString(fmt.Sprintf("    Issues opened: %d | Issues closed: %d\n",
				len(ms.IssuesOpened), len(ms.IssuesClosed)))
			b.WriteString(fmt.Sprintf("    PRs opened: %d | PRs merged: %d\n",
				len(ms.PRsOpened), len(ms.PRsMerged)))
			b.WriteString(fmt.Sprintf("    Commits: %d\n", ms.Commits))

			if len(ms.ReposActive) > 0 {
				repos := make([]string, 0, len(ms.ReposActive))
				for r := range ms.ReposActive {
					repos = append(repos, r)
				}
				b.WriteString(fmt.Sprintf("    Active in: %s\n", strings.Join(repos, ", ")))
			}

			// List PR titles
			for _, pr := range ms.PRsOpened {
				b.WriteString(fmt.Sprintf("    - PR opened: #%d %s\n", pr.GetNumber(), pr.GetTitle()))
			}
			for _, pr := range ms.PRsMerged {
				b.WriteString(fmt.Sprintf("    - PR merged: #%d %s\n", pr.GetNumber(), pr.GetTitle()))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}
