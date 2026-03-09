package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/trilok/dbt-weekly-report/internal/config"
	gh "github.com/trilok/dbt-weekly-report/internal/github"
	"github.com/trilok/dbt-weekly-report/internal/llm"
	"github.com/trilok/dbt-weekly-report/internal/report"
)

func main() {
	configPath  := flag.String("config", "config.yaml", "Path to configuration file")
	promptsPath := flag.String("prompts", "prompts.yaml", "Path to YAML prompts file (system + templates)")
	preset      := flag.String("range", "", "Override time range preset (weekly, biweekly, monthly, quarterly)")
	model       := flag.String("model", "", "Override LLM model (e.g. llama3.1, claude-sonnet-4-20250514, gpt-4o)")
	gatewayURL  := flag.String("url", "", "Override gateway URL (e.g. http://localhost:8090 for ai-gateway-platform)")
	dryRun      := flag.Bool("dry-run", false, "Fetch GitHub data only, skip LLM generation")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *preset != "" {
		cfg.TimeRange.Preset = *preset
	}
	if *model != "" {
		cfg.LLM.Gateway.Model = *model
	}
	if *gatewayURL != "" {
		cfg.LLM.Gateway.URL = *gatewayURL
	}

	// Resolve time range
	since, until, rangeLabel, err := cfg.TimeRange.Resolve()
	if err != nil {
		log.Fatalf("Invalid time range: %v", err)
	}

	// Context with graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
	}()

	// ---- Step 1: Fetch GitHub data ----
	log.Println("=== dbt Report Generator ===")
	log.Printf("Time range: %s (%s to %s)", rangeLabel,
		since.Format("2006-01-02"), until.Format("2006-01-02"))

	ghClient := gh.NewClient(cfg.GitHub)
	stats, err := ghClient.FetchAllRepoStats(ctx, since, until)
	if err != nil {
		log.Fatalf("Failed to fetch GitHub data: %v", err)
	}

	log.Printf("Fetched data for %d repositories", len(stats))

	// Trend window = max(12 weeks, 2× the report period).
	reportDays := int(until.Sub(since).Hours() / 24)
	trendDays := max(reportDays*2, 84)
	trendSince := until.AddDate(0, 0, -trendDays)
	var trendStats []gh.RepoStats
	if !since.After(trendSince) {
		trendStats = stats
	} else {
		log.Printf("Fetching %d-week trend window for charts...", trendDays/7)
		trendStats, err = ghClient.FetchAllRepoStats(ctx, trendSince, until)
		if err != nil {
			log.Printf("Warning: could not fetch trend data, charts will use report data: %v", err)
			trendStats = stats
		}
	}

	// Print quick summary
	for _, s := range stats {
		log.Printf("  %s: %d issues opened, %d closed, %d PRs merged, %d commits",
			s.RepoName, len(s.OpenedIssues), len(s.ClosedIssues), len(s.MergedPRs), len(s.Commits))
	}

	// Show team activity summary
	if len(cfg.Teams) > 0 {
		teamActivities := gh.ExtractTeamActivity(stats, cfg.Teams)
		for _, ta := range teamActivities {
			log.Printf("  Team %q:", ta.TeamName)
			for _, m := range ta.Members {
				ms := ta.MemberStats[m]
				if ms != nil {
					log.Printf("    @%s: %d PRs opened, %d merged, %d commits",
						ms.Username, len(ms.PRsOpened), len(ms.PRsMerged), ms.Commits)
				}
			}
		}
	}

	if *dryRun {
		log.Println("Dry run mode - skipping report generation")
		summaryData := gh.BuildSummaryData(stats)
		fmt.Println(summaryData.ToPromptText())
		if len(cfg.Teams) > 0 {
			teamActivities := gh.ExtractTeamActivity(stats, cfg.Teams)
			fmt.Println(gh.TeamActivityToPromptText(teamActivities))
		}
		return
	}

	// ---- Step 2: Initialize LLM provider ----
	llmProvider, err := llm.NewProvider(cfg.LLM)
	if err != nil {
		log.Fatalf("Failed to initialize LLM provider: %v", err)
	}
	log.Printf("LLM ready: %s", llmProvider.Name())

	prompts := report.LoadPrompts(*promptsPath)
	log.Printf("Prompts loaded from: %s", *promptsPath)

	// ---- Step 3: Generate report ----
	gen := report.NewGenerator(llmProvider, cfg.Report, cfg.Teams, prompts)
	outputPath, err := gen.Generate(ctx, stats, trendStats, rangeLabel)
	if err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}

	log.Printf("Report generated successfully: %s", outputPath)
	fmt.Printf("\n✅ Report saved to: %s\n", outputPath)
}
