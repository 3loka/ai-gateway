package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const anthropicAPIBase = "https://api.anthropic.com/v1"
const anthropicVersion = "2023-06-01"

// Anthropic implements Provider for the Anthropic Messages API.
type Anthropic struct {
	apiKey string
	client *http.Client
}

func NewAnthropic(apiKey string) *Anthropic {
	return &Anthropic{apiKey: apiKey, client: &http.Client{}}
}

func (a *Anthropic) Name() string { return "anthropic" }

func (a *Anthropic) Ping(ctx context.Context) error {
	// A minimal models list call to verify connectivity.
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, anthropicAPIBase+"/models", nil)
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("anthropic ping: HTTP %d", resp.StatusCode)
	}
	return nil
}

func (a *Anthropic) Generate(ctx context.Context, req Request) (Response, error) {
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type body struct {
		Model     string    `json:"model"`
		MaxTokens int       `json:"max_tokens"`
		System    string    `json:"system,omitempty"`
		Messages  []message `json:"messages"`
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}
	model := req.Model
	if model == "" {
		model = "claude-haiku-4-5-20251001"
	}

	payload := body{
		Model:     model,
		MaxTokens: maxTokens,
		System:    req.System,
		Messages:  []message{{Role: "user", Content: req.Prompt}},
	}

	raw, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIBase+"/messages", bytes.NewReader(raw))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	httpReq.Header.Set("content-type", "application/json")

	httpResp, err := a.client.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("anthropic request: %w", err)
	}
	defer httpResp.Body.Close()

	respBytes, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("anthropic error %d: %s", httpResp.StatusCode, string(respBytes))
	}

	var out struct {
		Model   string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return Response{}, fmt.Errorf("anthropic parse: %w", err)
	}

	text := ""
	for _, c := range out.Content {
		if c.Type == "text" {
			text += c.Text
		}
	}

	return Response{
		Text:         text,
		Model:        out.Model,
		InputTokens:  out.Usage.InputTokens,
		OutputTokens: out.Usage.OutputTokens,
	}, nil
}
