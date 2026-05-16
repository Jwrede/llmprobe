package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Google struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewGoogle(apiKey, baseURL string) *Google {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	return &Google{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (g *Google) Name() string { return "google" }

func (g *Google) Probe(pc ProviderContext) (Result, error) {
	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": pc.Prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": pc.MaxTokens,
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return Result{}, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s",
		g.baseURL, pc.Model, g.apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), pc.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return Result{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := g.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Result{}, httpError(resp, "Google")
	}

	var result Result
	var ttftRecorded bool
	tokenCount := 0
	usageTokens := 0

	err = ParseSSEStream(resp.Body, func(eventType string, data []byte) error {
		var chunk struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
			UsageMetadata *struct {
				CandidatesTokenCount int `json:"candidatesTokenCount"`
			} `json:"usageMetadata"`
		}
		if err := json.Unmarshal(data, &chunk); err != nil {
			return nil
		}

		if chunk.UsageMetadata != nil && chunk.UsageMetadata.CandidatesTokenCount > 0 {
			usageTokens = chunk.UsageMetadata.CandidatesTokenCount
		}

		if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
			text := chunk.Candidates[0].Content.Parts[0].Text
			if text != "" {
				if !ttftRecorded {
					result.TTFT = time.Since(start)
					ttftRecorded = true
				}
				tokenCount++
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
