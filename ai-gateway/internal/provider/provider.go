package provider

import "context"

// Request is the normalised input sent to any LLM provider.
type Request struct {
	System      string
	Prompt      string
	Model       string
	Temperature float64
	MaxTokens   int
}

// Response is the normalised output from any LLM provider.
type Response struct {
	Text         string
	Model        string
	InputTokens  int
	OutputTokens int
}

// Provider is the common interface every LLM backend must satisfy.
type Provider interface {
	// Name returns the provider identifier (e.g. "anthropic", "openai", "ollama").
	Name() string
	// Generate sends a request and returns the completed response.
	Generate(ctx context.Context, req Request) (Response, error)
	// Ping checks whether the provider is reachable (used by /health).
	Ping(ctx context.Context) error
}
