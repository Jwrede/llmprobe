package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "probes.yml")
	os.WriteFile(path, []byte(`
defaults:
  prompt: "Hello"
  max_tokens: 10
  timeout: 5s
  concurrency: 3
providers:
  - name: openai
    api_key: "test-key"
    models:
      - name: gpt-4o
        thresholds:
          max_ttft: 2s
`), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Defaults.Prompt != "Hello" {
		t.Errorf("prompt = %q, want %q", cfg.Defaults.Prompt, "Hello")
	}
	if cfg.Defaults.MaxTokens != 10 {
		t.Errorf("max_tokens = %d, want 10", cfg.Defaults.MaxTokens)
	}
	if cfg.Defaults.Timeout.Duration != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", cfg.Defaults.Timeout.Duration)
	}
	if cfg.Defaults.Concurrency != 3 {
		t.Errorf("concurrency = %d, want 3", cfg.Defaults.Concurrency)
	}
	if len(cfg.Providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(cfg.Providers))
	}
	if cfg.Providers[0].Models[0].Thresholds.MaxTTFT.Duration != 2*time.Second {
		t.Errorf("max_ttft = %v, want 2s", cfg.Providers[0].Models[0].Thresholds.MaxTTFT.Duration)
	}
}

func TestLoadEnvExpansion(t *testing.T) {
	t.Setenv("TEST_LLM_KEY", "expanded-key")
	dir := t.TempDir()
	path := filepath.Join(dir, "probes.yml")
	os.WriteFile(path, []byte(`
providers:
  - name: openai
    api_key: ${TEST_LLM_KEY}
    models:
      - name: gpt-4o
`), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Providers[0].APIKey != "expanded-key" {
		t.Errorf("api_key = %q, want %q", cfg.Providers[0].APIKey, "expanded-key")
	}
}

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "probes.yml")
	os.WriteFile(path, []byte(`
providers:
  - name: openai
    api_key: "key"
    models:
      - name: gpt-4o
`), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Defaults.MaxTokens != 50 {
		t.Errorf("default max_tokens = %d, want 50", cfg.Defaults.MaxTokens)
	}
	if cfg.Providers[0].Models[0].MaxTokens != 50 {
		t.Errorf("model max_tokens = %d, want 50", cfg.Providers[0].Models[0].MaxTokens)
	}
}

func TestLoadUnknownProvider(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "probes.yml")
	os.WriteFile(path, []byte(`
providers:
  - name: mistral
    api_key: "key"
    models:
      - name: mistral-large
`), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestLoadMissingAPIKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "probes.yml")
	os.WriteFile(path, []byte(`
providers:
  - name: openai
    api_key: ""
    models:
      - name: gpt-4o
`), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty api_key")
	}
}

func TestLoadNoProviders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "probes.yml")
	os.WriteFile(path, []byte(`providers: []`), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for no providers")
	}
}
