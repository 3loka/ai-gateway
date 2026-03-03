# dbt Weekly Report Generator

A Go tool that generates activity reports for the dbt ecosystem (dbt-core + adapters) using GitHub data and an LLM for narrative summaries. Designed to be **pluggable** — swap between a local LLM (Ollama), Anthropic Claude, or OpenAI GPT with a single config change.

## What It Does

- Pulls activity data from dbt-core + adapter repos via the GitHub API
- Configurable time range: weekly, biweekly, monthly, quarterly, or custom date range
- Tracks: issues opened/closed, PRs opened/merged, commits, releases, top contributors
- **Team callout sections** — track specific team members' contributions (e.g., the India team ramp-up)
- Generates an executive summary, detailed per-repo analysis, team callouts, and recommendations using an LLM
- Produces an HTML report with inline SVG charts (or plain markdown)
- Fully configurable: repos, time window, LLM provider, teams, output format

## Architecture

```
cmd/main.go                          # Entry point
internal/
  config/config.go                   # YAML config + env var overrides + time range resolution
  github/
    client.go                        # GitHub API data fetcher + team activity extraction
    summary.go                       # Structures data for LLM prompts
  llm/
    provider.go                      # Provider interface + factory
    ollama/ollama.go                 # Local LLM via Ollama API
    anthropic/anthropic.go           # Claude API
    openai/openai.go                 # OpenAI API
  report/
    generator.go                     # Orchestrates report assembly + team callouts
    prompts.go                       # LLM prompt templates
  charts/
    charts.go                        # SVG chart generation (no external deps)
```

The **provider pattern** means adding a new LLM backend is just:
1. Create a new package under `internal/llm/`
2. Implement the `Provider` interface (two methods: `Name()` and `GenerateText()`)
3. Register it in `internal/llm/provider.go`

## Prerequisites

- **Go 1.22+**
- **GitHub token** (optional but recommended — unauthenticated requests are rate-limited to 60/hour)
- **One of the following LLM backends:**

### Option A: Local LLM with Ollama (no API key needed)

```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull a model
ollama pull llama3.1

# Verify it's running
curl http://localhost:11434/api/tags
```

### Option B: Anthropic Claude

Set your API key:
```bash
export ANTHROPIC_API_KEY=sk-ant-your-key-here
```

### Option C: OpenAI GPT

Set your API key:
```bash
export OPENAI_API_KEY=sk-your-key-here
```

## Quick Start

```bash
# Clone and enter the project
git clone <repo-url>
cd dbt-weekly-report

# Install Go dependencies
go mod tidy

# Set your GitHub token (recommended)
export GITHUB_TOKEN=ghp_your_token_here

# Edit config.yaml to set your preferred LLM provider
# Default is "ollama" — make sure Ollama is running

# Build and run
make run
```

## Configuration

All settings live in `config.yaml`. Environment variables override config file values.

### Time Range

Use presets for common intervals, or specify exact dates:

```yaml
time_range:
  # Presets: "weekly" (7d), "biweekly" (14d), "monthly" (30d), "quarterly" (90d), "custom"
  preset: "weekly"

  # Only used when preset is "custom"
  custom_start: "2026-01-01"
  custom_end: "2026-01-31"
```

Override via CLI:
```bash
./bin/dbt-weekly-report --range monthly
./bin/dbt-weekly-report --range quarterly
```

### Switching LLM Providers

**In config.yaml:**
```yaml
llm:
  provider: "ollama"     # Change to "anthropic" or "openai"
```

**Via environment variable:**
```bash
LLM_PROVIDER=anthropic ./bin/dbt-weekly-report
```

**Via CLI flag:**
```bash
./bin/dbt-weekly-report --provider openai
```

### Team Callouts

Track specific team members' contributions in a dedicated report section. Useful for monitoring new team ramp-up, tracking distributed team contributions, etc.

```yaml
teams:
  - name: "India Team"
    description: "New team spun up in January — tracking ramp-up and contributions"
    members:
      - "aahel"
      - "tauhid621"
      - "ash2shukla"
      - "sriramr98"

  # Add more teams as needed
  - name: "Platform Team"
    description: "Core platform engineers"
    members:
      - "user1"
      - "user2"
```

Each team gets its own section in the report with:
- Per-member metrics table (issues, PRs, commits, repos active in)
- LLM-generated narrative assessing ramp-up, contributions, and areas for support

### Adding/Removing Repositories

Edit the `github.repositories` section in `config.yaml`:
```yaml
github:
  repositories:
    - owner: "dbt-labs"
      name: "dbt-core"
    - owner: "dbt-labs"
      name: "dbt-snowflake"
    # Add any GitHub repo here
```

## Usage

```bash
# Full report with default settings (weekly, using Ollama)
make run

# Fetch data only (no LLM needed, useful for testing GitHub connectivity)
make dry-run

# Use a specific provider
make run-ollama
make run-anthropic
make run-openai

# Custom config file
./bin/dbt-weekly-report --config my-config.yaml

# Override provider and time range via flags
./bin/dbt-weekly-report --provider anthropic --range monthly

# Monthly report using OpenAI
./bin/dbt-weekly-report --provider openai --range monthly
```

## Output

Reports are saved to `./reports/` (configurable) as either HTML or Markdown. The HTML report includes:

- Executive summary (LLM-generated)
- Metrics table with totals across all repos
- SVG bar charts (issues, PRs, commits per repo)
- Detailed per-repo analysis (LLM-generated)
- Team callout sections with per-member metrics and LLM narrative
- Recommendations & action items (LLM-generated)

## Environment Variables

| Variable | Description |
|---|---|
| `GITHUB_TOKEN` | GitHub personal access token |
| `LLM_PROVIDER` | Override LLM provider (ollama/anthropic/openai) |
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `OPENAI_API_KEY` | OpenAI API key |

## Adding a New LLM Provider

1. Create `internal/llm/yourprovider/yourprovider.go`
2. Implement the interface:

```go
type Provider interface {
    Name() string
    GenerateText(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}
```

3. Add the provider config struct to `internal/config/config.go`
4. Register it in the `NewProvider` factory in `internal/llm/provider.go`

## Tips

- **Start with `--dry-run`** to verify GitHub data fetching works before hitting your LLM
- **Use a smaller model** (e.g., `llama3.1:8b`) locally for faster iteration
- **Set `GITHUB_TOKEN`** even for public repos to avoid rate limiting (60 req/hr without vs 5000/hr with)
- The tool handles GitHub pagination automatically, so it works even for very active repos
- For monthly/quarterly reports, consider using a more capable model (70B+ or Claude/GPT-4) since there's more data to synthesize
