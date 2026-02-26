package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level gateway configuration.
type Config struct {
	Providers     map[string]ProviderConfig `yaml:"providers"`
	Profiles      map[string]Profile        `yaml:"profiles"`
	Server        ServerConfig              `yaml:"server"`
	FallbackChain []string                  `yaml:"fallback_chain"`
}

// ProviderConfig holds credentials / base URL for one LLM backend.
type ProviderConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
}

// Profile is a named set of defaults for LLM calls.
type Profile struct {
	Provider    string  `yaml:"provider"`
	Model       string  `yaml:"model"`
	Temperature float64 `yaml:"temperature"`
	MaxTokens   int     `yaml:"max_tokens"`
}

// ServerConfig controls the HTTP server.
type ServerConfig struct {
	Port        int    `yaml:"port"`
	APIKey      string `yaml:"api_key"`       // optional gateway-level auth
	LogRequests bool   `yaml:"log_requests"`
}

// Load reads gateway.yaml (or the given path), expands env vars, and returns the config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	// Expand ${VAR} / $VAR environment variables before parsing.
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Defaults
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	// Ensure a "default" profile always exists.
	if _, ok := cfg.Profiles["default"]; !ok {
		// Pick the first provider we can find as a fallback.
		for name := range cfg.Providers {
			cfg.Profiles["default"] = Profile{Provider: name, MaxTokens: 2048, Temperature: 0.3}
			break
		}
	}

	return &cfg, nil
}
