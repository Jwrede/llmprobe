package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Bedrock struct {
	accessKey string
	secretKey string
	region    string
	baseURL   string
	client    *http.Client
}

func NewBedrock(accessKey, secretKey, region, baseURL string) *Bedrock {
	if baseURL == "" && region != "" {
		baseURL = fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com", region)
	}
	return &Bedrock{
		accessKey: accessKey,
		secretKey: secretKey,
		region:    region,
		baseURL:   baseURL,
		client:    &http.Client{},
	}
}

func (b *Bedrock) Name() string { return "bedrock" }

func (b *Bedrock) Probe(pc ProviderContext) (Result, error) {
	body := map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{"text": pc.Prompt},
				},
			},
		},
		"inferenceConfig": map[string]interface{}{
			"maxTokens": pc.MaxTokens,
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return Result{}, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/model/%s/converse-stream", b.baseURL, pc.Model)

	ctx, cancel := context.WithTimeout(context.Background(), pc.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return Result{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	signV4(req, payload, b.region, "bedrock", b.accessKey, b.secretKey)

	start := time.Now()
	resp, err := b.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Result{}, httpError(resp, "Bedrock")
	}

	var result Result
	var ttftRecorded bool
	tokenCount := 0
	usageTokens := 0

	err = parseEventStream(resp.Body, func(eventType string, payload []byte) error {
		switch eventType {
		case "contentBlockDelta":
			var delta struct {
				Delta struct {
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal(payload, &delta); err != nil {
				return nil
			}
			if delta.Delta.Text != "" {
				if !ttftRecorded {
					result.TTFT = time.Since(start)
					ttftRecorded = true
				}
				tokenCount++
			}

		case "metadata":
			var meta struct {
				Usage struct {
					OutputTokens int `json:"outputTokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal(payload, &meta); err != nil {
				return nil
			}
			if meta.Usage.OutputTokens > 0 {
				usageTokens = meta.Usage.OutputTokens
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
