package charts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/trilok/dbt-weekly-report/internal/github"
)

// ChartPaths holds paths to generated chart image files (SVG embedded in HTML).
type ChartPaths struct {
	IssuesBarChart  string
	PRsBarChart     string
	CommitsBarChart string
	ActivityPie     string
}

// GenerateCharts creates HTML-based chart files and returns their paths.
func GenerateCharts(stats []github.RepoStats, outputDir string) (*ChartPaths, error) {
	chartsDir := filepath.Join(outputDir, "charts")
	if err := os.MkdirAll(chartsDir, 0755); err != nil {
		return nil, fmt.Errorf("creating charts dir: %w", err)
	}

	paths := &ChartPaths{}
	var err error

	paths.IssuesBarChart, err = generateIssuesChart(stats, chartsDir)
	if err != nil {
		return nil, fmt.Errorf("generating issues chart: %w", err)
	}

	paths.PRsBarChart, err = generatePRsChart(stats, chartsDir)
	if err != nil {
		return nil, fmt.Errorf("generating PRs chart: %w", err)
	}

	paths.CommitsBarChart, err = generateCommitsChart(stats, chartsDir)
	if err != nil {
		return nil, fmt.Errorf("generating commits chart: %w", err)
	}

	paths.ActivityPie, err = generateActivityPieChart(stats, chartsDir)
	if err != nil {
		return nil, fmt.Errorf("generating activity pie chart: %w", err)
	}

	return paths, nil
}

// GenerateTrendSVGs returns inline SVG line charts showing weekly trends over the fetch period.
func GenerateTrendSVGs(trends []github.RepoTrend) map[string]string {
	colors := []string{"#3498db", "#e74c3c"}
	svgs := make(map[string]string)
	svgs["issues_trend"] = generateTrendLineSVG(trends, "Issues Opened — Rolling Trend",
		func(b github.WeekBucket) int { return b.IssuesOpened }, colors)
	svgs["issues_closed_trend"] = generateTrendLineSVG(trends, "Issues Closed — Rolling Trend",
		func(b github.WeekBucket) int { return b.IssuesClosed }, colors)
	svgs["prs_trend"] = generateTrendLineSVG(trends, "PRs Merged — Rolling Trend",
		func(b github.WeekBucket) int { return b.PRsMerged }, colors)
	return svgs
}

func generateTrendLineSVG(trends []github.RepoTrend, title string, valFn func(github.WeekBucket) int, colors []string) string {
	if len(trends) == 0 || len(trends[0].Weeks) == 0 {
		return ""
	}
	nWeeks := len(trends[0].Weeks)
	W, H := 700, 300
	mL, mR, mT, mB := 45, 20, 50, 65
	plotW := W - mL - mR
	plotH := H - mT - mB

	maxVal := 1
	for _, tr := range trends {
		for _, b := range tr.Weeks {
			if v := valFn(b); v > maxVal {
				maxVal = v
			}
		}
	}

	xAt := func(i int) int {
		if nWeeks <= 1 {
			return mL + plotW/2
		}
		return mL + i*plotW/(nWeeks-1)
	}
	yAt := func(v int) int {
		return mT + plotH - int(float64(v)/float64(maxVal)*float64(plotH))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" style="max-width:700px;width:100%%">`, W, H))
	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" fill="#fafafa" rx="8"/>`, W, H))
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="28" text-anchor="middle" font-size="15" font-weight="bold" fill="#333">%s</text>`, W/2, title))

	// Y gridlines + labels (5 steps)
	for i := 0; i <= 4; i++ {
		yVal := maxVal * i / 4
		y := yAt(yVal)
		sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#e8e8e8" stroke-width="1"/>`, mL, y, W-mR, y))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="end" font-size="11" fill="#aaa">%d</text>`, mL-5, y+4, yVal))
	}

	// X axis baseline
	sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#ccc" stroke-width="1"/>`, mL, mT+plotH, W-mR, mT+plotH))

	// X axis labels (rotated)
	for i, week := range trends[0].Weeks {
		x := xAt(i)
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="end" font-size="10" fill="#888" transform="rotate(-35 %d %d)">%s</text>`,
			x, mT+plotH+14, x, mT+plotH+14, week.Label))
	}

	// One line + dots per repo
	for ri, tr := range trends {
		color := colors[ri%len(colors)]

		var pts []string
		for i, b := range tr.Weeks {
			pts = append(pts, fmt.Sprintf("%d,%d", xAt(i), yAt(valFn(b))))
		}
		sb.WriteString(fmt.Sprintf(`<polyline points="%s" fill="none" stroke="%s" stroke-width="2.5" stroke-linejoin="round" stroke-linecap="round"/>`,
			strings.Join(pts, " "), color))

		for i, b := range tr.Weeks {
			v := valFn(b)
			x, y := xAt(i), yAt(v)
			sb.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="3.5" fill="%s"/>`, x, y, color))
			if v > 0 {
				sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" font-size="10" fill="%s">%d</text>`, x, y-9, color, v))
			}
		}

		// Legend entry
		lx := mL + ri*200
		ly := H - 18
		sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="2.5"/>`, lx, ly, lx+18, ly, color))
		sb.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="3.5" fill="%s"/>`, lx+9, ly, color))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-size="11" fill="#555">%s</text>`, lx+22, ly+4, shortName(tr.RepoName)))
	}

	sb.WriteString(`</svg>`)
	return sb.String()
}

// GenerateInlineSVGs returns inline SVG strings for embedding directly in HTML reports.
func GenerateInlineSVGs(stats []github.RepoStats) map[string]string {
	svgs := make(map[string]string)

	svgs["issues"] = generateIssuesSVG(stats)
	svgs["prs"] = generatePRsSVG(stats)
	svgs["commits"] = generateCommitsSVG(stats)

	return svgs
}

func shortName(fullName string) string {
	for i := len(fullName) - 1; i >= 0; i-- {
		if fullName[i] == '/' {
			return fullName[i+1:]
		}
	}
	return fullName
}

// ---- SVG-based chart generators (no external dependencies) ----

func generateIssuesSVG(stats []github.RepoStats) string {
	return generateGroupedBarSVG(stats, "Issues: Opened vs Closed", func(s github.RepoStats) (int, int) {
		return len(s.OpenedIssues), len(s.ClosedIssues)
	}, "#e74c3c", "#2ecc71", "Opened", "Closed")
}

func generatePRsSVG(stats []github.RepoStats) string {
	return generateGroupedBarSVG(stats, "Pull Requests: Opened vs Merged", func(s github.RepoStats) (int, int) {
		return len(s.OpenedPRs), len(s.MergedPRs)
	}, "#3498db", "#9b59b6", "Opened", "Merged")
}

func generateCommitsSVG(stats []github.RepoStats) string {
	n := len(stats)
	if n == 0 {
		return ""
	}

	width := 600
	height := 350
	margin := 80
	barAreaW := width - 2*margin
	barAreaH := height - 2*margin

	maxVal := 0
	for _, s := range stats {
		if len(s.Commits) > maxVal {
			maxVal = len(s.Commits)
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	barW := barAreaW / n
	if barW > 80 {
		barW = 80
	}

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" style="max-width:600px;width:100%%">`, width, height)
	svg += fmt.Sprintf(`<rect width="%d" height="%d" fill="#fafafa" rx="8"/>`, width, height)
	svg += fmt.Sprintf(`<text x="%d" y="30" text-anchor="middle" font-size="16" font-weight="bold" fill="#333">Commits This Week</text>`, width/2)

	for i, s := range stats {
		val := len(s.Commits)
		barH := int(float64(val) / float64(maxVal) * float64(barAreaH))
		x := margin + i*(barAreaW/n) + (barAreaW/n-barW)/2
		y := margin + barAreaH - barH

		svg += fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="#f39c12" rx="4"/>`, x, y, barW, barH)
		svg += fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" font-size="12" fill="#333">%d</text>`, x+barW/2, y-5, val)
		svg += fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" font-size="11" fill="#666" transform="rotate(-30 %d %d)">%s</text>`,
			x+barW/2, height-margin+40, x+barW/2, height-margin+40, shortName(s.RepoName))
	}

	svg += `</svg>`
	return svg
}

func generateGroupedBarSVG(stats []github.RepoStats, title string, valFn func(github.RepoStats) (int, int), color1, color2, label1, label2 string) string {
	n := len(stats)
	if n == 0 {
		return ""
	}

	width := 600
	height := 350
	margin := 80
	barAreaW := width - 2*margin
	barAreaH := height - 2*margin

	maxVal := 0
	for _, s := range stats {
		v1, v2 := valFn(s)
		if v1 > maxVal {
			maxVal = v1
		}
		if v2 > maxVal {
			maxVal = v2
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	groupW := barAreaW / n
	barW := groupW / 3
	if barW > 35 {
		barW = 35
	}

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" style="max-width:600px;width:100%%">`, width, height)
	svg += fmt.Sprintf(`<rect width="%d" height="%d" fill="#fafafa" rx="8"/>`, width, height)
	svg += fmt.Sprintf(`<text x="%d" y="30" text-anchor="middle" font-size="16" font-weight="bold" fill="#333">%s</text>`, width/2, title)

	// Legend
	svg += fmt.Sprintf(`<rect x="%d" y="45" width="12" height="12" fill="%s"/>`, width/2-80, color1)
	svg += fmt.Sprintf(`<text x="%d" y="56" font-size="11" fill="#666">%s</text>`, width/2-64, label1)
	svg += fmt.Sprintf(`<rect x="%d" y="45" width="12" height="12" fill="%s"/>`, width/2+10, color2)
	svg += fmt.Sprintf(`<text x="%d" y="56" font-size="11" fill="#666">%s</text>`, width/2+26, label2)

	for i, s := range stats {
		v1, v2 := valFn(s)
		barH1 := int(float64(v1) / float64(maxVal) * float64(barAreaH))
		barH2 := int(float64(v2) / float64(maxVal) * float64(barAreaH))

		x1 := margin + i*groupW + (groupW-2*barW-4)/2
		x2 := x1 + barW + 4
		y1 := margin + barAreaH - barH1
		y2 := margin + barAreaH - barH2

		svg += fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="%s" rx="3"/>`, x1, y1, barW, barH1, color1)
		svg += fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" font-size="11" fill="#333">%d</text>`, x1+barW/2, y1-4, v1)

		svg += fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="%s" rx="3"/>`, x2, y2, barW, barH2, color2)
		svg += fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" font-size="11" fill="#333">%d</text>`, x2+barW/2, y2-4, v2)

		labelX := (x1 + x2 + barW) / 2
		svg += fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" font-size="11" fill="#666" transform="rotate(-30 %d %d)">%s</text>`,
			labelX, height-margin+40, labelX, height-margin+40, shortName(s.RepoName))
	}

	svg += `</svg>`
	return svg
}

// File-based chart generators (write HTML files with embedded SVGs)
func generateIssuesChart(stats []github.RepoStats, dir string) (string, error) {
	return writeChartHTML(dir, "issues.html", generateIssuesSVG(stats))
}

func generatePRsChart(stats []github.RepoStats, dir string) (string, error) {
	return writeChartHTML(dir, "prs.html", generatePRsSVG(stats))
}

func generateCommitsChart(stats []github.RepoStats, dir string) (string, error) {
	return writeChartHTML(dir, "commits.html", generateCommitsSVG(stats))
}

func generateActivityPieChart(stats []github.RepoStats, dir string) (string, error) {
	return writeChartHTML(dir, "activity.html", generateCommitsSVG(stats))
}

func writeChartHTML(dir, filename, svgContent string) (string, error) {
	path := filepath.Join(dir, filename)
	html := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Chart</title></head>
<body style="display:flex;justify-content:center;padding:20px">
%s
</body></html>`, svgContent)

	if err := os.WriteFile(path, []byte(html), 0644); err != nil {
		return "", err
	}
	return path, nil
}
