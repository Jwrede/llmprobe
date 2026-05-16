package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Azure struct {
	apiKey     string
	baseURL    string
	apiVersion string
	client     *http.Client
}

func NewAzure(apiKey, baseURL, apiVersion string) *Azure {
	if apiVersion == "" {
		apiVersion = "2024-10-21"
	}
	return &Azure{
		apiKey:     apiKey,
		baseURL:    baseURL,
		apiVersion: apiVersion,
		client:     &http.Client{},
	}
}

func (a *Azure) Name() string { return "azure" }

func (a *Azure) Probe(pc ProviderContext) (Result, error) {
	body := map[string]interface{}{
		"max_tokens": pc.MaxTokens,
		"stream":     true,
		"stream_options": map[string]interface{}{
			"include_usage": true,
		},
		"messages": []map[string]string{
			{"role": "user", "content": pc.Prompt},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return Result{}, fmt.Errorf("marshal request: %w", err)
	}

	// Azure URL: {base_url}/openai/deployments/{model}/chat/completions?api-version={version}
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		a.baseURL, pc.Model, a.apiVersion)

	ctx, cancel := context.WithTimeout(context.Background(), pc.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return Result{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", a.apiKey)

	start := time.Now()
	resp, err := a.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Result{}, httpError(resp, "Azure OpenAI")
	}

	var result Result
	var ttftRecorded bool
	tokenCount := 0
	usageTokens := 0

	err = ParseSSEStream(resp.Body, func(eventType string, data []byte) error {
		if string(data) == "[DONE]" {
			return nil
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(data, &chunk); err != nil {
			return nil
		}

		if chunk.Usage != nil && chunk.Usage.CompletionTokens > 0 {
			usageTokens = chunk.Usage.CompletionTokens
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			if !ttftRecorded {
				result.TTFT = time.Since(start)
				ttftRecorded = true
			}
			tokenCount++
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
