package llm

import (
	"context"
	"fmt"

	"github.com/trilok/dbt-weekly-report/internal/config"
	"github.com/trilok/dbt-weekly-report/internal/llm/gateway"
)

// Provider is the interface all LLM backends must implement.
type Provider interface {
	Name() string
	// GenerateText sends a system prompt + user prompt and returns the reply text.
	GenerateText(ctx context.Context, system, prompt string) (string, error)
}

// NewProvider creates the LLM provider. All model routing is handled by the
// ai-gateway service — configure it in ../ai-gateway/gateway.yaml.
func NewProvider(cfg config.LLMConfig) (Provider, error) {
	if cfg.Gateway.URL == "" {
		return nil, fmt.Errorf("gateway URL is required (set llm.gateway.url in config.yaml)")
	}
	return gateway.New(cfg.Gateway), nil
}
