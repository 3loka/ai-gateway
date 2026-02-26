package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const openAIAPIBase = "https://api.openai.com/v1"

// OpenAI implements Provider for the OpenAI Chat Completions API.
type OpenAI struct {
	apiKey string
	client *http.Client
}

func NewOpenAI(apiKey string) *OpenAI {
	return &OpenAI{apiKey: apiKey, client: &http.Client{}}
}

func (o *OpenAI) Name() string { return "openai" }

func (o *OpenAI) Ping(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, openAIAPIBase+"/models", nil)
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("openai ping: HTTP %d", resp.StatusCode)
	}
	return nil
}

func (o *OpenAI) Generate(ctx context.Context, req Request) (Response, error) {
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type body struct {
		Model       string    `json:"model"`
		Temperature float64   `json:"temperature"`
		MaxTokens   int       `json:"max_tokens"`
		Messages    []message `json:"messages"`
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}
	model := req.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	msgs := []message{}
	if req.System != "" {
		msgs = append(msgs, message{Role: "system", Content: req.System})
	}
	msgs = append(msgs, message{Role: "user", Content: req.Prompt})

	payload := body{
		Model:       model,
		Temperature: req.Temperature,
		MaxTokens:   maxTokens,
		Messages:    msgs,
	}

	raw, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIAPIBase+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	httpReq.Header.Set("content-type", "application/json")

	httpResp, err := o.client.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("openai request: %w", err)
	}
	defer httpResp.Body.Close()

	respBytes, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("openai error %d: %s", httpResp.StatusCode, string(respBytes))
	}

	var out struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return Response{}, fmt.Errorf("openai parse: %w", err)
	}

	text := ""
	if len(out.Choices) > 0 {
		text = out.Choices[0].Message.Content
	}

	return Response{
		Text:         text,
		Model:        out.Model,
		InputTokens:  out.Usage.PromptTokens,
		OutputTokens: out.Usage.CompletionTokens,
	}, nil
}
