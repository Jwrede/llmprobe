package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type OpenAI struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewOpenAI(apiKey, baseURL string) *OpenAI {
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	return &OpenAI{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (o *OpenAI) Name() string { return "openai" }

func (o *OpenAI) Probe(pc ProviderContext) (Result, error) {
	body := map[string]interface{}{
		"model":      pc.Model,
		"max_tokens": pc.MaxTokens,
		"stream":     true,
		"stream_options": map[string]interface{}{
			"include_usage": true,
		},
		"messages": []map[string]string{
			{"role": "user", "content": pc.Prompt},
		},
	}
	if pc.ResponseFormat == "json" {
		body["response_format"] = map[string]string{"type": "json_object"}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return Result{}, fmt.Errorf("marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), pc.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return Result{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	start := time.Now()
	resp, err := o.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Result{}, httpError(resp, "OpenAI")
	}

	var result Result
	var ttftRecorded bool
	var contentBuf strings.Builder
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
			contentBuf.WriteString(chunk.Choices[0].Delta.Content)
			tokenCount++
		}
		return nil
	})
	if err != nil {
		return Result{}, fmt.Errorf("parse stream: %w", err)
	}

	result.TotalLatency = time.Since(start)
	result.Content = contentBuf.String()
	if usageTokens > 0 {
		result.TokenCount = usageTokens
	} else {
		result.TokenCount = tokenCount
	}

	return result, nil
}
