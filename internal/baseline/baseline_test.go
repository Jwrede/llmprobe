package baseline

import (
	"path/filepath"
	"testing"

	"github.com/Jwrede/llmprobe/internal/report"
)

func TestCreateAndSaveLoad(t *testing.T) {
	records := []report.Record{
		{Provider: "openai", Model: "gpt-4o", Status: "healthy", TTFTMs: 100, LatencyMs: 400},
		{Provider: "openai", Model: "gpt-4o", Status: "healthy", TTFTMs: 120, LatencyMs: 450},
		{Provider: "openai", Model: "gpt-4o", Status: "healthy", TTFTMs: 110, LatencyMs: 420},
		{Provider: "anthropic", Model: "claude", Status: "healthy", TTFTMs: 200, LatencyMs: 800},
		{Provider: "anthropic", Model: "claude", Status: "healthy", TTFTMs: 250, LatencyMs: 900},
	}

	bl := Create(records)

	if len(bl.Endpoints) != 2 {
		t.Fatalf("endpoints = %d, want 2", len(bl.Endpoints))
	}

	ep, ok := bl.Lookup("openai", "gpt-4o")
	if !ok {
		t.Fatal("openai/gpt-4o not found")
	}
	if ep.TTFTP50 < 100 || ep.TTFTP50 > 120 {
		t.Errorf("TTFT p50 = %.1f, want ~110", ep.TTFTP50)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")
	if err := bl.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Endpoints) != 2 {
		t.Fatalf("loaded endpoints = %d, want 2", len(loaded.Endpoints))
	}

	ep2, ok := loaded.Lookup("anthropic", "claude")
	if !ok {
		t.Fatal("anthropic/claude not found in loaded baseline")
	}
	if ep2.LatencyP50 < 800 || ep2.LatencyP50 > 900 {
		t.Errorf("latency p50 = %.1f, want ~850", ep2.LatencyP50)
	}
}

func TestLookupMissing(t *testing.T) {
	bl := &Baseline{Endpoints: make(map[string]EndpointBaseline)}
	_, ok := bl.Lookup("unknown", "model")
	if ok {
		t.Error("expected missing lookup to return false")
	}
}
