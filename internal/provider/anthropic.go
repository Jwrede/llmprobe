package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Anthropic struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewAnthropic(apiKey, baseURL string) *Anthropic {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	return &Anthropic{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (a *Anthropic) Name() string { return "anthropic" }

func (a *Anthropic) Probe(pc ProviderContext) (Result, error) {
	body := map[string]interface{}{
		"model":      pc.Model,
		"max_tokens": pc.MaxTokens,
		"stream":     true,
		"messages": []map[string]string{
			{"role": "user", "content": pc.Prompt},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return Result{}, fmt.Errorf("marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), pc.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return Result{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	start := time.Now()
	resp, err := a.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("HTTP %d from Anthropic", resp.StatusCode)
	}

	var result Result
	var ttftRecorded bool
	tokenCount := 0
	usageTokens := 0

	err = ParseSSEStream(resp.Body, func(eventType string, data []byte) error {
		switch eventType {
		case "content_block_delta":
			var delta struct {
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal(data, &delta); err != nil {
				return nil
			}
			if delta.Delta.Type == "text_delta" && delta.Delta.Text != "" {
				if !ttftRecorded {
					result.TTFT = time.Since(start)
					ttftRecorded = true
				}
				tokenCount++
			}

		case "message_delta":
			var msg struct {
				Usage struct {
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal(data, &msg); err != nil {
				return nil
			}
			if msg.Usage.OutputTokens > 0 {
				usageTokens = msg.Usage.OutputTokens
			}
		}
		return nil
	})
	if err != nil {
		return Result{}, fmt.Errorf("parse stream: %w", err)
	}

	result.TotalLatency = time.Since(start)
	if usageTokens > 0 {
		result.TokenCount = usageTokens
	} else {
		result.TokenCount = tokenCount
	}

	return result, nil
}
