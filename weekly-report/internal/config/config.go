package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GitHub    GitHubConfig    `yaml:"github"`
	TimeRange TimeRangeConfig `yaml:"time_range"`
	LLM       LLMConfig       `yaml:"llm"`
	Teams     []TeamConfig    `yaml:"teams"`
	Report    ReportConfig    `yaml:"report"`
}

type GitHubConfig struct {
	Token        string       `yaml:"token"`
	Repositories []RepoConfig `yaml:"repositories"`
}

type RepoConfig struct {
	Owner string `yaml:"owner"`
	Name  string `yaml:"name"`
}

func (r RepoConfig) FullName() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

type TimeRangeConfig struct {
	Preset      string `yaml:"preset"`       // weekly | biweekly | monthly | quarterly | custom
	CustomStart string `yaml:"custom_start"` // YYYY-MM-DD
	CustomEnd   string `yaml:"custom_end"`   // YYYY-MM-DD
}

// Resolve returns the actual since/until times based on the preset or custom range.
func (t TimeRangeConfig) Resolve() (since, until time.Time, label string, err error) {
	until = time.Now()

	switch t.Preset {
	case "weekly", "":
		since = until.AddDate(0, 0, -7)
		label = "Weekly"
	case "biweekly":
		since = until.AddDate(0, 0, -14)
		label = "Biweekly"
	case "monthly":
		since = until.AddDate(0, -1, 0)
		label = "Monthly"
	case "quarterly":
		since = until.AddDate(0, -3, 0)
		label = "Quarterly"
	case "custom":
		if t.CustomStart == "" || t.CustomEnd == "" {
			err = fmt.Errorf("custom time range requires both custom_start and custom_end (YYYY-MM-DD)")
			return
		}
		since, err = time.Parse("2006-01-02", t.CustomStart)
		if err != nil {
			err = fmt.Errorf("invalid custom_start date %q: %w", t.CustomStart, err)
			return
		}
		until, err = time.Parse("2006-01-02", t.CustomEnd)
		if err != nil {
			err = fmt.Errorf("invalid custom_end date %q: %w", t.CustomEnd, err)
			return
		}
		until = until.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		label = "Custom"
	default:
		err = fmt.Errorf("unknown time range preset %q (use: weekly, biweekly, monthly, quarterly, custom)", t.Preset)
		return
	}

	return
}

type TeamConfig struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Members     []string `yaml:"members"` // GitHub usernames
}

type LLMConfig struct {
	Gateway GatewayConfig `yaml:"gateway"`
}

// GatewayConfig points to a running ai-gateway-platform instance.
// The platform uses the standard OpenAI /v1/chat/completions API.
type GatewayConfig struct {
	URL    string `yaml:"url"`     // e.g. "http://localhost:8080"
	APIKey string `yaml:"api_key"` // X-API-Key team key (see teams section in gateway config.yaml)
	Model  string `yaml:"model"`   // model name (e.g. "claude-sonnet-4-20250514", "llama3.1")
}

type ReportConfig struct {
	OutputDir     string `yaml:"output_dir"`
	Format        string `yaml:"format"`
	IncludeCharts bool   `yaml:"include_charts"`
}

// Load reads the config file and applies environment variable overrides.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Environment variable overrides
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		cfg.GitHub.Token = token
	}

	// Defaults
	if cfg.TimeRange.Preset == "" {
		cfg.TimeRange.Preset = "weekly"
	}
	if cfg.LLM.Gateway.URL == "" {
		cfg.LLM.Gateway.URL = "http://localhost:8090"
	}
	if cfg.LLM.Gateway.Model == "" {
		cfg.LLM.Gateway.Model = "llama3.1"
	}
	if cfg.Report.OutputDir == "" {
		cfg.Report.OutputDir = "./reports"
	}
	if cfg.Report.Format == "" {
		cfg.Report.Format = "markdown"
	}

	return &cfg, nil
}
