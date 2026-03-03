// Package gateway implements the llm.Provider interface by calling an
// OpenAI-compatible /v1/chat/completions endpoint (e.g. ai-gateway-platform).
package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/trilok/dbt-weekly-report/internal/config"
)

// Provider calls an OpenAI-compatible /v1/chat/completions endpoint.
type Provider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func New(cfg config.GatewayConfig) *Provider {
	// Custom transport with TCP keepalive — prevents Docker's userland proxy
	// from silently dropping long-lived connections (LLM calls can take 30s+).
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 15 * time.Second,
		}).DialContext,
	}
	return &Provider{
		baseURL: cfg.URL,
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		client:  &http.Client{Transport: transport},
	}
}

func (p *Provider) Name() string { return "gateway(" + p.model + ")" }

// GenerateText sends system + user messages to /v1/chat/completions and returns the reply text.
func (p *Provider) GenerateText(ctx context.Context, system, prompt string) (string, error) {
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type request struct {
		Model    string    `json:"model"`
		Messages []message `json:"messages"`
	}

	msgs := []message{}
	if system != "" {
		msgs = append(msgs, message{Role: "system", Content: system})
	}
	msgs = append(msgs, message{Role: "user", Content: prompt})

	raw, err := json.Marshal(request{Model: p.model, Messages: msgs})
	if err != nil {
		return "", fmt.Errorf("gateway: marshal request: %w", err)
	}

	url := p.baseURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("gateway: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("X-API-Key", p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gateway: request to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &errResp)
		msg := errResp.Error.Message
		if msg == "" {
			msg = string(body)
		}
		return "", fmt.Errorf("gateway: HTTP %d: %s", resp.StatusCode, msg)
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("gateway: parse response: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("gateway: empty choices in response")
	}
	return out.Choices[0].Message.Content, nil
}
