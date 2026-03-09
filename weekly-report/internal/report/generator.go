package report

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/trilok/dbt-weekly-report/internal/charts"
	"github.com/trilok/dbt-weekly-report/internal/config"
	gh "github.com/trilok/dbt-weekly-report/internal/github"
	"github.com/trilok/dbt-weekly-report/internal/llm"
)

// Generator orchestrates report creation.
type Generator struct {
	llmProvider llm.Provider
	config      config.ReportConfig
	teams       []config.TeamConfig
	prompts     *PromptsConfig
}

// NewGenerator creates a new report generator.
func NewGenerator(provider llm.Provider, cfg config.ReportConfig, teams []config.TeamConfig, prompts *PromptsConfig) *Generator {
	return &Generator{
		llmProvider: provider,
		config:      cfg,
		teams:       teams,
		prompts:     prompts,
	}
}

// generate resolves a named template and calls the LLM.
func (g *Generator) generate(ctx context.Context, tmpl, data string) (string, error) {
	prompt, err := buildPrompt(g.prompts, tmpl, data)
	if err != nil {
		return "", err
	}
	log.Printf("Generating %s via %s...", tmpl, g.llmProvider.Name())
	return g.llmProvider.GenerateText(ctx, g.prompts.System, prompt)
}

// Generate creates the full report.
func (g *Generator) Generate(ctx context.Context, stats []gh.RepoStats, trendStats []gh.RepoStats, rangeLabel string) (string, error) {
	// Build the structured data
	summaryData := gh.BuildSummaryData(stats)
	rawDataText := summaryData.ToPromptText()

	execSummary, err := g.generate(ctx, tmplExecSummary, rawDataText)
	if err != nil {
		return "", fmt.Errorf("generating executive summary: %w", err)
	}

	detailedAnalysis, err := g.generate(ctx, tmplDetailedAnalysis, rawDataText)
	if err != nil {
		return "", fmt.Errorf("generating detailed analysis: %w", err)
	}

	recommendations, err := g.generate(ctx, tmplRecommendations, rawDataText)
	if err != nil {
		return "", fmt.Errorf("generating recommendations: %w", err)
	}

	// Generate team callout sections
	var teamCallouts []teamCalloutResult
	if len(g.teams) > 0 {
		teamActivities := gh.ExtractTeamActivity(stats, g.teams)
		for _, activity := range teamActivities {
			teamDataText := gh.TeamActivityToPromptText([]gh.TeamActivity{activity})
			calloutText, err := g.generate(ctx, tmplTeamCallout, teamDataText)
			if err != nil {
				log.Printf("Warning: failed to generate team callout for %s: %v", activity.TeamName, err)
				calloutText = "Team callout generation failed. Raw data:\n" + teamDataText
			}
			teamCallouts = append(teamCallouts, teamCalloutResult{
				Name:        activity.TeamName,
				Description: activity.Description,
				Activity:    activity,
				Narrative:   calloutText,
			})
		}
	}

	// Build the metrics table
	metricsTable := buildMetricsTable(summaryData)

	// Generate trend charts (weekly buckets over the full fetch window)
	var chartSVGs map[string]string
	if g.config.IncludeCharts {
		trends := gh.BucketByWeek(trendStats)
		chartSVGs = charts.GenerateTrendSVGs(trends)
	}

	// Assemble the full report
	report := g.assembleReport(summaryData.Period, rangeLabel, execSummary, metricsTable,
		chartSVGs, detailedAnalysis, recommendations, teamCallouts)

	// Write to file
	outputPath, err := g.writeReport(report, summaryData.Period, rangeLabel)
	if err != nil {
		return "", fmt.Errorf("writing report: %w", err)
	}

	return outputPath, nil
}

type teamCalloutResult struct {
	Name        string
	Description string
	Activity    gh.TeamActivity
	Narrative   string
}

func buildMetricsTable(data gh.WeeklySummaryData) string {
	var b strings.Builder
	b.WriteString("| Repository | Issues Opened | Issues Closed | PRs Opened | PRs Merged | Commits | Releases |\n")
	b.WriteString("|------------|:------------:|:------------:|:---------:|:---------:|:-------:|:--------:|\n")

	totalOpened, totalClosed, totalPROpen, totalPRMerge, totalCommits, totalReleases := 0, 0, 0, 0, 0, 0

	for _, r := range data.RepoSummaries {
		b.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %d | %d | %d |\n",
			r.RepoName, r.IssuesOpened, r.IssuesClosed, r.PRsOpened, r.PRsMerged, r.Commits, r.Releases))
		totalOpened += r.IssuesOpened
		totalClosed += r.IssuesClosed
		totalPROpen += r.PRsOpened
		totalPRMerge += r.PRsMerged
		totalCommits += r.Commits
		totalReleases += r.Releases
	}

	b.WriteString(fmt.Sprintf("| **TOTAL** | **%d** | **%d** | **%d** | **%d** | **%d** | **%d** |\n",
		totalOpened, totalClosed, totalPROpen, totalPRMerge, totalCommits, totalReleases))

	return b.String()
}

func buildTeamMetricsTable(tc teamCalloutResult) string {
	var b strings.Builder
	b.WriteString("| Member | Issues Opened | Issues Closed | PRs Opened | PRs Merged | Repos Active |\n")
	b.WriteString("|--------|:------------:|:------------:|:---------:|:---------:|:------------:|\n")

	totalIO, totalIC, totalPO, totalPM := 0, 0, 0, 0
	for _, member := range tc.Activity.Members {
		lower := strings.ToLower(member)
		ms, ok := tc.Activity.MemberStats[lower]
		if !ok {
			continue
		}
		io, ic, po, pm := len(ms.IssuesOpened), len(ms.IssuesClosed), len(ms.PRsOpened), len(ms.PRsMerged)
		totalIO += io
		totalIC += ic
		totalPO += po
		totalPM += pm
		b.WriteString(fmt.Sprintf("| @%s | %d | %d | %d | %d | %d |\n",
			ms.Username, io, ic, po, pm, len(ms.ReposActive)))
	}
	b.WriteString(fmt.Sprintf("| **TOTAL** | **%d** | **%d** | **%d** | **%d** | |\n",
		totalIO, totalIC, totalPO, totalPM))

	return b.String()
}

func buildTeamMemberDetails(tc teamCalloutResult) string {
	var b strings.Builder
	for _, member := range tc.Activity.Members {
		lower := strings.ToLower(member)
		ms, ok := tc.Activity.MemberStats[lower]
		if !ok {
			continue
		}
		if len(ms.PRsMerged) == 0 && len(ms.PRsOpened) == 0 && len(ms.IssuesOpened) == 0 {
			continue
		}
		repos := make([]string, 0, len(ms.ReposActive))
		for r := range ms.ReposActive {
			repos = append(repos, r)
		}
		repoNote := ""
		if len(repos) > 0 {
			repoNote = fmt.Sprintf(" *(repos: %s)*", strings.Join(repos, ", "))
		}
		b.WriteString(fmt.Sprintf("**@%s**%s\n", ms.Username, repoNote))
		if len(ms.PRsMerged) > 0 {
			b.WriteString(fmt.Sprintf("- Merged PRs (%d):\n", len(ms.PRsMerged)))
			for _, pr := range ms.PRsMerged {
				b.WriteString(fmt.Sprintf("  - #%d %s\n", pr.GetNumber(), pr.GetTitle()))
			}
		}
		if len(ms.PRsOpened) > 0 {
			b.WriteString(fmt.Sprintf("- Open PRs (%d):\n", len(ms.PRsOpened)))
			for _, pr := range ms.PRsOpened {
				b.WriteString(fmt.Sprintf("  - #%d %s\n", pr.GetNumber(), pr.GetTitle()))
			}
		}
		if len(ms.IssuesOpened) > 0 {
			b.WriteString(fmt.Sprintf("- Issues filed (%d):\n", len(ms.IssuesOpened)))
			for _, issue := range ms.IssuesOpened {
				b.WriteString(fmt.Sprintf("  - #%d %s\n", issue.GetNumber(), issue.GetTitle()))
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (g *Generator) assembleReport(period, rangeLabel, execSummary, metricsTable string,
	chartSVGs map[string]string, detailedAnalysis, recommendations string,
	teamCallouts []teamCalloutResult) string {

	if g.config.Format == "html" {
		return buildHTMLReport(period, rangeLabel, execSummary, metricsTable,
			chartSVGs, detailedAnalysis, recommendations, teamCallouts)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# dbt Ecosystem %s Report\n\n", rangeLabel))
	b.WriteString(fmt.Sprintf("**Period:** %s\n\n", period))

	if execSummary != "" {
		b.WriteString(fmt.Sprintf("---\n\n## Executive Summary\n\n%s\n\n", execSummary))
	}

	b.WriteString(fmt.Sprintf("---\n\n## Metrics at a Glance\n\n%s\n\n", metricsTable))

	if len(chartSVGs) > 0 {
		b.WriteString("---\n\n## Charts\n\n")
		for _, key := range []string{"issues_trend", "issues_closed_trend", "prs_trend"} {
			if svg, ok := chartSVGs[key]; ok && svg != "" {
				b.WriteString(svg)
				b.WriteString("\n\n")
			}
		}
	}

	if detailedAnalysis != "" {
		b.WriteString(fmt.Sprintf("---\n\n## Detailed Analysis\n\n%s\n\n", detailedAnalysis))
	}

	for _, tc := range teamCallouts {
		b.WriteString(fmt.Sprintf("---\n\n## Team Callout: %s\n\n", tc.Name))
		if tc.Description != "" {
			b.WriteString(fmt.Sprintf("*%s*\n\n", tc.Description))
		}
		b.WriteString("### Metrics at a Glance\n\n")
		b.WriteString(buildTeamMetricsTable(tc))
		memberDetails := buildTeamMemberDetails(tc)
		if memberDetails != "" {
			b.WriteString("\n### Member Activity\n\n")
			b.WriteString(memberDetails)
		}
		if tc.Narrative != "" {
			b.WriteString(fmt.Sprintf("\n### Analysis\n\n%s\n\n", tc.Narrative))
		}
	}

	if recommendations != "" {
		b.WriteString(fmt.Sprintf("---\n\n## Recommendations & Action Items\n\n%s\n\n", recommendations))
	}

	b.WriteString(fmt.Sprintf("---\n\n*Report generated on %s using dbt-weekly-report*\n",
		time.Now().Format("2006-01-02 15:04:05")))

	return b.String()
}

func (g *Generator) writeReport(content, period, rangeLabel string) (string, error) {
	if err := os.MkdirAll(g.config.OutputDir, 0755); err != nil {
		return "", err
	}

	datePart := strings.ReplaceAll(period, " to ", "_to_")
	datePart = strings.ReplaceAll(datePart, " ", "")

	ext := "md"
	if g.config.Format == "html" {
		ext = "html"
	}

	filename := fmt.Sprintf("dbt_%s_report_%s.%s", strings.ToLower(rangeLabel), datePart, ext)
	path := filepath.Join(g.config.OutputDir, filename)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}

	return path, nil
}

func buildHTMLReport(period, rangeLabel, execSummary, metricsTable string,
	chartSVGs map[string]string, detailedAnalysis, recommendations string,
	teamCallouts []teamCalloutResult) string {

	chartHTML := ""
	if chartSVGs != nil {
		chartHTML = fmt.Sprintf(`
		<div class="charts-grid">
			<div class="chart-container">%s</div>
			<div class="chart-container">%s</div>
			<div class="chart-container">%s</div>
		</div>`, chartSVGs["issues_trend"], chartSVGs["issues_closed_trend"], chartSVGs["prs_trend"])
	}

	htmlMetrics := markdownTableToHTML(metricsTable)

	// Build team callout HTML
	teamHTML := ""
	for _, tc := range teamCallouts {
		teamMetricsHTML := markdownTableToHTML(buildTeamMetricsTable(tc))
		descHTML := ""
		if tc.Description != "" {
			descHTML = fmt.Sprintf(`<p class="team-desc"><em>%s</em></p>`, tc.Description)
		}
		memberDetailsHTML := ""
		if details := buildTeamMemberDetails(tc); details != "" {
			memberDetailsHTML = fmt.Sprintf(`<h3>Member Activity</h3><div>%s</div>`, markdownToBasicHTML(details))
		}
		teamHTML += fmt.Sprintf(`
			<div class="section team-callout">
				<h2>Team Callout: %s</h2>
				%s
				<h3>Metrics at a Glance</h3>
				%s
				%s
				<h3>Analysis</h3>
				<div>%s</div>
			</div>`, tc.Name, descHTML, teamMetricsHTML, memberDetailsHTML, markdownToBasicHTML(tc.Narrative))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>dbt %s Report - %s</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6; color: #333; background: #f5f5f5;
            padding: 20px;
        }
        .container { max-width: 1100px; margin: 0 auto; }
        .header {
            background: linear-gradient(135deg, #ff694a, #e85d3a);
            color: white; padding: 30px 40px; border-radius: 12px 12px 0 0;
        }
        .header h1 { font-size: 28px; margin-bottom: 5px; }
        .header .period { opacity: 0.9; font-size: 16px; }
        .content {
            background: white; padding: 40px;
            border-radius: 0 0 12px 12px; box-shadow: 0 2px 12px rgba(0,0,0,0.08);
        }
        .section { margin-bottom: 36px; }
        .section h2 {
            font-size: 22px; color: #e85d3a; margin-bottom: 16px;
            padding-bottom: 8px; border-bottom: 2px solid #fce4dc;
        }
        .section h3 { font-size: 18px; color: #555; margin: 16px 0 8px; }
        .team-callout {
            background: #fef9f7; border-left: 4px solid #e85d3a;
            padding: 24px; border-radius: 0 8px 8px 0;
        }
        .team-callout h2 { color: #d14b2f; }
        .team-desc { color: #666; font-size: 14px; margin-bottom: 12px; }
        table {
            width: 100%%; border-collapse: collapse; margin: 16px 0;
            font-size: 14px;
        }
        th { background: #f8f8f8; padding: 10px 12px; text-align: center; border: 1px solid #e0e0e0; font-weight: 600; }
        td { padding: 8px 12px; border: 1px solid #e0e0e0; text-align: center; }
        tr:hover td { background: #fef9f7; }
        .charts-grid {
            display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px; margin: 20px 0;
        }
        .chart-container {
            background: #fafafa; border-radius: 8px; padding: 16px;
            border: 1px solid #eee;
        }
        .footer {
            text-align: center; margin-top: 24px; color: #999; font-size: 13px;
        }
        p { margin-bottom: 12px; }
        ul, ol { margin: 12px 0 12px 24px; }
        li { margin-bottom: 6px; }
        strong { color: #333; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; font-size: 13px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>dbt Ecosystem %s Report</h1>
            <div class="period">%s</div>
        </div>
        <div class="content">
            <div class="section">
                <h2>Executive Summary</h2>
                <div>%s</div>
            </div>
            <div class="section">
                <h2>Metrics at a Glance</h2>
                %s
            </div>
            %s
            <div class="section">
                <h2>Detailed Analysis</h2>
                <div>%s</div>
            </div>
            %s
            <div class="section">
                <h2>Recommendations &amp; Action Items</h2>
                <div>%s</div>
            </div>
        </div>
        <div class="footer">
            Report generated on %s using dbt-weekly-report
        </div>
    </div>
</body>
</html>`, rangeLabel, period,
		rangeLabel, period,
		markdownToBasicHTML(execSummary),
		htmlMetrics,
		chartHTML,
		markdownToBasicHTML(detailedAnalysis),
		teamHTML,
		markdownToBasicHTML(recommendations),
		time.Now().Format("2006-01-02 15:04:05"))
}

// markdownToBasicHTML does minimal markdown -> HTML conversion for display.
func markdownToBasicHTML(md string) string {
	lines := strings.Split(md, "\n")
	var result []string
	inList := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "### ") {
			if inList {
				result = append(result, "</ul>")
				inList = false
			}
			result = append(result, fmt.Sprintf("<h3>%s</h3>", strings.TrimPrefix(trimmed, "### ")))
		} else if strings.HasPrefix(trimmed, "## ") {
			if inList {
				result = append(result, "</ul>")
				inList = false
			}
			result = append(result, fmt.Sprintf("<h3>%s</h3>", strings.TrimPrefix(trimmed, "## ")))
		} else if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			if !inList {
				result = append(result, "<ul>")
				inList = true
			}
			content := strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "* ")
			result = append(result, fmt.Sprintf("<li>%s</li>", applyInlineFormatting(content)))
		} else if trimmed == "" {
			if inList {
				result = append(result, "</ul>")
				inList = false
			}
		} else {
			if inList {
				result = append(result, "</ul>")
				inList = false
			}
			result = append(result, fmt.Sprintf("<p>%s</p>", applyInlineFormatting(trimmed)))
		}
	}

	if inList {
		result = append(result, "</ul>")
	}

	return strings.Join(result, "\n")
}

func applyInlineFormatting(s string) string {
	// Bold: **text**
	for {
		start := strings.Index(s, "**")
		if start == -1 {
			break
		}
		end := strings.Index(s[start+2:], "**")
		if end == -1 {
			break
		}
		end += start + 2
		s = s[:start] + "<strong>" + s[start+2:end] + "</strong>" + s[end+2:]
	}
	// Inline code: `text`
	for {
		start := strings.Index(s, "`")
		if start == -1 {
			break
		}
		end := strings.Index(s[start+1:], "`")
		if end == -1 {
			break
		}
		end += start + 1
		s = s[:start] + "<code>" + s[start+1:end] + "</code>" + s[end+1:]
	}
	return s
}

func markdownTableToHTML(md string) string {
	lines := strings.Split(strings.TrimSpace(md), "\n")
	if len(lines) < 2 {
		return md
	}

	var html strings.Builder
	html.WriteString("<table>\n<thead>\n<tr>\n")

	headers := splitTableRow(lines[0])
	for _, h := range headers {
		html.WriteString(fmt.Sprintf("<th>%s</th>\n", strings.TrimSpace(h)))
	}
	html.WriteString("</tr>\n</thead>\n<tbody>\n")

	for _, line := range lines[2:] {
		cells := splitTableRow(line)
		html.WriteString("<tr>\n")
		for _, c := range cells {
			html.WriteString(fmt.Sprintf("<td>%s</td>\n", strings.TrimSpace(c)))
		}
		html.WriteString("</tr>\n")
	}

	html.WriteString("</tbody>\n</table>")
	return html.String()
}

func splitTableRow(row string) []string {
	row = strings.Trim(row, "|")
	return strings.Split(row, "|")
}
