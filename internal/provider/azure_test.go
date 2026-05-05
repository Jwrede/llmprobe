package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAzureProbeSuccess(t *testing.T) {
	chunks := []string{
		`{"choices":[{"delta":{"role":"assistant"}}]}`,
		`{"choices":[{"delta":{"content":"Hello"}}]}`,
		`{"choices":[{"delta":{"content":" world"}}]}`,
		`{"choices":[],"usage":{"completion_tokens":2}}`,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("api-key") != "test-key" {
			t.Errorf("unexpected api-key header: %q", r.Header.Get("api-key"))
		}
		if !strings.Contains(r.URL.Path, "/openai/deployments/gpt-4o/chat/completions") {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("api-version") != "2024-10-21" {
			t.Errorf("unexpected api-version: %q", r.URL.Query().Get("api-version"))
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

	client := NewAzure("test-key", server.URL, "")
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
		t.Errorf("token count = %d, want 2", result.TokenCount)
	}
	if result.TTFT == 0 {
		t.Error("TTFT should be > 0")
	}
}
