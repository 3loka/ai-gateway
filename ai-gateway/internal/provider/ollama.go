package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Ollama implements Provider for a local Ollama instance via its chat API.
type Ollama struct {
	baseURL string
	client  *http.Client
}

func NewOllama(baseURL string) *Ollama {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &Ollama{baseURL: baseURL, client: &http.Client{}}
}

func (o *Ollama) Name() string { return "ollama" }

func (o *Ollama) Ping(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, o.baseURL+"/api/tags", nil)
	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("ollama ping: HTTP %d", resp.StatusCode)
	}
	return nil
}

func (o *Ollama) Generate(ctx context.Context, req Request) (Response, error) {
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type options struct {
		Temperature float64 `json:"temperature,omitempty"`
		NumPredict  int     `json:"num_predict,omitempty"`
	}
	type body struct {
		Model    string    `json:"model"`
		Messages []message `json:"messages"`
		Stream   bool      `json:"stream"`
		Options  options   `json:"options,omitempty"`
	}

	model := req.Model
	if model == "" {
		model = "llama3.1"
	}

	msgs := []message{}
	if req.System != "" {
		msgs = append(msgs, message{Role: "system", Content: req.System})
	}
	msgs = append(msgs, message{Role: "user", Content: req.Prompt})

	payload := body{
		Model:    model,
		Messages: msgs,
		Stream:   false,
		Options:  options{Temperature: req.Temperature, NumPredict: req.MaxTokens},
	}

	raw, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/chat", bytes.NewReader(raw))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("content-type", "application/json")

	httpResp, err := o.client.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("ollama request: %w", err)
	}
	defer httpResp.Body.Close()

	respBytes, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("ollama error %d: %s", httpResp.StatusCode, string(respBytes))
	}

	var out struct {
		Model   string `json:"model"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		PromptEvalCount int `json:"prompt_eval_count"`
		EvalCount       int `json:"eval_count"`
	}
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return Response{}, fmt.Errorf("ollama parse: %w", err)
	}

	return Response{
		Text:         out.Message.Content,
		Model:        out.Model,
		InputTokens:  out.PromptEvalCount,
		OutputTokens: out.EvalCount,
	}, nil
}
