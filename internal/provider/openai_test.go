package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenAIProbeSuccess(t *testing.T) {
	chunks := []string{
		`{"choices":[{"delta":{"role":"assistant"}}]}`,
		`{"choices":[{"delta":{"content":"Hello"}}]}`,
		`{"choices":[{"delta":{"content":" world"}}]}`,
		`{"choices":[],"usage":{"completion_tokens":2}}`,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header: %q", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		for _, c := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", c)
			flusher.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	client := NewOpenAI("test-key", server.URL)
	result, err := client.Probe(ProviderContext{
		Model:     "gpt-4o",
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
	if result.TotalLatency == 0 {
		t.Error("TotalLatency should be > 0")
	}
}

func TestOpenAIProbeHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewOpenAI("test-key", server.URL)
	_, err := client.Probe(ProviderContext{
		Model:     "gpt-4o",
		Prompt:    "Hello",
		MaxTokens: 10,
		Timeout:   5 * time.Second,
	})
	if err == nil {
		t.Fatal("expected error for 429 response")
	}
}

func TestOpenAIProbeTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	client := NewOpenAI("test-key", server.URL)
	_, err := client.Probe(ProviderContext{
		Model:     "gpt-4o",
		Prompt:    "Hello",
		MaxTokens: 10,
		Timeout:   100 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
