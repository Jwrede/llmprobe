package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAnthropicProbeSuccess(t *testing.T) {
	events := []struct {
		event string
		data  string
	}{
		{"message_start", `{"type":"message_start"}`},
		{"content_block_start", `{"type":"content_block_start"}`},
		{"content_block_delta", `{"delta":{"type":"text_delta","text":"Hello"}}`},
		{"content_block_delta", `{"delta":{"type":"text_delta","text":" there"}}`},
		{"message_delta", `{"usage":{"output_tokens":2}}`},
		{"message_stop", `{"type":"message_stop"}`},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("unexpected api key header: %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("unexpected anthropic-version: %q", r.Header.Get("anthropic-version"))
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		for _, e := range events {
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", e.event, e.data)
			flusher.Flush()
		}
	}))
	defer server.Close()

	client := NewAnthropic("test-key", server.URL)
	result, err := client.Probe(ProviderContext{
		Model:     "claude-sonnet-4-20250514",
		Prompt:    "Hello",
		MaxTokens: 10,
		Timeout:   5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.TokenCount != 2 {
		t.Errorf("token count = %d, want 2 (from usage)", result.TokenCount)
	}
	if result.TTFT == 0 {
		t.Error("TTFT should be > 0")
	}
}

func TestAnthropicProbeHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewAnthropic("bad-key", server.URL)
	_, err := client.Probe(ProviderContext{
		Model:     "claude-sonnet-4-20250514",
		Prompt:    "Hello",
		MaxTokens: 10,
		Timeout:   5 * time.Second,
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}
