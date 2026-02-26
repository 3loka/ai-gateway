package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/trilok/ai-gateway/internal/config"
	"github.com/trilok/ai-gateway/internal/provider"
)

// GenerateRequest is the inbound API request body.
type GenerateRequest struct {
	System      string  `json:"system"`
	Prompt      string  `json:"prompt"`
	Profile     string  `json:"profile"`      // named profile (optional)
	Provider    string  `json:"provider"`     // override profile provider (optional)
	Model       string  `json:"model"`        // override profile model (optional)
	Temperature float64 `json:"temperature"`  // override (optional)
	MaxTokens   int     `json:"max_tokens"`   // override (optional)
}

// GenerateResponse is what the API returns.
type GenerateResponse struct {
	Text      string `json:"text"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Usage     Usage  `json:"usage"`
	LatencyMs int64  `json:"latency_ms"`
}

// Usage mirrors provider token counts.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Gateway routes LLM requests to the right provider after resolving profiles.
type Gateway struct {
	cfg       *config.Config
	providers map[string]provider.Provider
}

// New constructs a Gateway and wires up all configured providers.
func New(cfg *config.Config, providers map[string]provider.Provider) *Gateway {
	return &Gateway{cfg: cfg, providers: providers}
}

// Generate resolves the request profile, selects the provider, and executes.
func (g *Gateway) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	resolved := g.resolveRequest(req)

	p, ok := g.providers[resolved.provider]
	if !ok {
		return nil, fmt.Errorf("provider %q is not configured", resolved.provider)
	}

	start := time.Now()
	resp, err := p.Generate(ctx, provider.Request{
		System:      resolved.system,
		Prompt:      resolved.prompt,
		Model:       resolved.model,
		Temperature: resolved.temperature,
		MaxTokens:   resolved.maxTokens,
	})
	if err != nil {
		// Try fallback chain if configured.
		if fallback, fbErr := g.tryFallback(ctx, resolved, err); fbErr == nil {
			resp = fallback
		} else {
			return nil, fmt.Errorf("provider %q failed: %w (fallback also failed: %v)", resolved.provider, err, fbErr)
		}
	}

	return &GenerateResponse{
		Text:      resp.Text,
		Provider:  resolved.provider,
		Model:     resp.Model,
		Usage:     Usage{InputTokens: resp.InputTokens, OutputTokens: resp.OutputTokens},
		LatencyMs: time.Since(start).Milliseconds(),
	}, nil
}

// Health returns a map of provider name → error (nil = reachable).
func (g *Gateway) Health(ctx context.Context) map[string]string {
	result := make(map[string]string)
	for name, p := range g.providers {
		if err := p.Ping(ctx); err != nil {
			result[name] = err.Error()
		} else {
			result[name] = "ok"
		}
	}
	return result
}

// Profiles returns the configured profile names and their settings.
func (g *Gateway) Profiles() map[string]config.Profile {
	return g.cfg.Profiles
}

// resolved holds the final, merged request fields after profile resolution.
type resolved struct {
	system      string
	prompt      string
	provider    string
	model       string
	temperature float64
	maxTokens   int
}

// resolveRequest merges: default profile → named profile → per-request overrides.
func (g *Gateway) resolveRequest(req GenerateRequest) resolved {
	// Start from the "default" profile.
	base := g.cfg.Profiles["default"]

	// Overlay named profile if specified.
	profileName := req.Profile
	if profileName == "" {
		profileName = "default"
	}
	if profileName != "default" {
		if p, ok := g.cfg.Profiles[profileName]; ok {
			if p.Provider != "" {
				base.Provider = p.Provider
			}
			if p.Model != "" {
				base.Model = p.Model
			}
			if p.Temperature != 0 {
				base.Temperature = p.Temperature
			}
			if p.MaxTokens != 0 {
				base.MaxTokens = p.MaxTokens
			}
		}
	}

	// Apply per-request overrides.
	r := resolved{
		system:      req.System,
		prompt:      req.Prompt,
		provider:    base.Provider,
		model:       base.Model,
		temperature: base.Temperature,
		maxTokens:   base.MaxTokens,
	}
	if req.Provider != "" {
		r.provider = req.Provider
	}
	if req.Model != "" {
		r.model = req.Model
	}
	if req.Temperature != 0 {
		r.temperature = req.Temperature
	}
	if req.MaxTokens != 0 {
		r.maxTokens = req.MaxTokens
	}

	return r
}

// tryFallback attempts each provider in the fallback chain (excluding the one that just failed).
func (g *Gateway) tryFallback(ctx context.Context, r resolved, originalErr error) (provider.Response, error) {
	for _, name := range g.cfg.FallbackChain {
		if name == r.provider {
			continue // skip the one that already failed
		}
		p, ok := g.providers[name]
		if !ok {
			continue
		}
		resp, err := p.Generate(ctx, provider.Request{
			System:      r.system,
			Prompt:      r.prompt,
			Model:       "",  // use provider's default model on fallback
			Temperature: r.temperature,
			MaxTokens:   r.maxTokens,
		})
		if err == nil {
			return resp, nil
		}
	}
	return provider.Response{}, fmt.Errorf("all fallbacks exhausted (original: %v)", originalErr)
}
