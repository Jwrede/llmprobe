package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGoogleProbeSuccess(t *testing.T) {
	chunks := []string{
		`{"candidates":[{"content":{"parts":[{"text":"Hello"}]}}]}`,
		`{"candidates":[{"content":{"parts":[{"text":" world"}]}}],"usageMetadata":{"candidatesTokenCount":5}}`,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "key=test-key") {
			t.Errorf("missing api key in query: %q", r.URL.RawQuery)
		}
		if !strings.Contains(r.URL.RawQuery, "alt=sse") {
			t.Errorf("missing alt=sse in query: %q", r.URL.RawQuery)
		}
		if !strings.Contains(r.URL.Path, "gemini-2.0-flash") {
			t.Errorf("model not in URL path: %q", r.URL.Path)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		for _, c := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", c)
			flusher.Flush()
		}
	}))
	defer server.Close()

	client := NewGoogle("test-key", server.URL)
	result, err := client.Probe(ProviderContext{
		Model:     "gemini-2.0-flash",
		Prompt:    "Hello",
		MaxTokens: 10,
		Timeout:   5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.TokenCount != 5 {
		t.Errorf("token count = %d, want 5 (from usageMetadata)", result.TokenCount)
	}
	if result.TTFT == 0 {
		t.Error("TTFT should be > 0")
	}
}
