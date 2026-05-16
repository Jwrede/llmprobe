package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestHandleProbeAllWithConfig(t *testing.T) {
	server := mockOpenAIServer(t)
	defer server.Close()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "probes.yml")
	os.WriteFile(cfgPath, []byte(`
providers:
  - name: openai
    api_key: "test-key"
    base_url: "`+server.URL+`"
    models:
      - name: gpt-4o
`), 0644)

	result, _, err := handleProbeAll(context.Background(), nil, probeAllArgs{Config: cfgPath})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "gpt-4o") {
		t.Error("result should contain model name")
	}
	if !strings.Contains(text, "healthy") {
		t.Error("result should show healthy status")
	}
}

func TestHandleProbeAllMissingConfig(t *testing.T) {
	result, _, _ := handleProbeAll(context.Background(), nil, probeAllArgs{Config: "/nonexistent/probes.yml"})
	if !result.IsError {
		t.Fatal("expected error for missing config")
	}
}

func TestHandleProbeModelMissingProvider(t *testing.T) {
	result, _, _ := handleProbeModel(context.Background(), nil, probeModelArgs{
		Model:     "gpt-4o",
		APIKeyEnv: "TEST_KEY",
	})
	if !result.IsError {
		t.Fatal("expected error for missing provider")
	}
	if !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "provider is required") {
		t.Error("error message should mention provider")
	}
}

func TestHandleProbeModelMissingModel(t *testing.T) {
	result, _, _ := handleProbeModel(context.Background(), nil, probeModelArgs{
		Provider:  "openai",
		APIKeyEnv: "TEST_KEY",
	})
	if !result.IsError {
		t.Fatal("expected error for missing model")
	}
	if !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "model is required") {
		t.Error("error message should mention model")
	}
}

func TestHandleProbeModelMissingAPIKeyEnv(t *testing.T) {
	result, _, _ := handleProbeModel(context.Background(), nil, probeModelArgs{
		Provider: "openai",
		Model:    "gpt-4o",
	})
	if !result.IsError {
		t.Fatal("expected error for missing api_key_env")
	}
	if !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "api_key_env is required") {
		t.Error("error message should mention api_key_env")
	}
}

func TestHandleProbeModelEmptyEnvVar(t *testing.T) {
	t.Setenv("LLMPROBE_TEST_EMPTY", "")
	result, _, _ := handleProbeModel(context.Background(), nil, probeModelArgs{
		Provider:  "openai",
		Model:     "gpt-4o",
		APIKeyEnv: "LLMPROBE_TEST_EMPTY",
	})
	if !result.IsError {
		t.Fatal("expected error for empty env var")
	}
	if !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "not set or empty") {
		t.Error("error message should mention env var is empty")
	}
}

func TestHandleProbeModelWithBaseURL(t *testing.T) {
	server := mockOpenAIServer(t)
	defer server.Close()

	t.Setenv("LLMPROBE_TEST_KEY", "test-key")

	result, _, err := handleProbeModel(context.Background(), nil, probeModelArgs{
		Provider:  "openai",
		Model:     "gpt-4o",
		APIKeyEnv: "LLMPROBE_TEST_KEY",
		BaseURL:   server.URL,
		Label:     "my-vllm",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "my-vllm") {
		t.Error("result should show label as provider name")
	}
}

func TestHandleGetConfigNoSecretLeak(t *testing.T) {
	server := mockOpenAIServer(t)
	defer server.Close()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "probes.yml")
	os.WriteFile(cfgPath, []byte(`
providers:
  - name: openai
    api_key: "super-secret-key-12345"
    base_url: "`+server.URL+`"
    models:
      - name: gpt-4o
`), 0644)

	result, _, err := handleGetConfig(context.Background(), nil, getConfigArgs{Config: cfgPath})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if strings.Contains(text, "super-secret-key-12345") {
		t.Error("get_config must not expose raw API key values")
	}
	if !strings.Contains(text, "has_api_key") {
		t.Error("get_config should indicate whether api_key is set")
	}
}

func mockOpenAIServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunks := []map[string]interface{}{
			{
				"choices": []map[string]interface{}{
					{"delta": map[string]string{"content": "Hello"}},
				},
			},
			{
				"choices": []map[string]interface{}{
					{"delta": map[string]string{"content": " world"}},
				},
			},
			{
				"usage": map[string]interface{}{"completion_tokens": 2},
			},
		}

		flusher, _ := w.(http.Flusher)
		for _, chunk := range chunks {
			data, _ := json.Marshal(chunk)
			w.Write([]byte("data: " + string(data) + "\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
		}
		w.Write([]byte("data: [DONE]\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
	}))
}
