package report

import (
	"bytes"
	"math"
	"os"
	"strings"
	"testing"
)

func TestParseJSONL(t *testing.T) {
	f, err := os.Open("../../testdata/sample.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	records, err := ParseJSONL(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 10 {
		t.Fatalf("got %d records, want 10", len(records))
	}
	if records[0].Provider != "openai" {
		t.Errorf("first record provider = %q, want openai", records[0].Provider)
	}
	if records[0].TTFTMs != 120 {
		t.Errorf("first record ttft_ms = %d, want 120", records[0].TTFTMs)
	}
}

func TestParseJSONLInvalid(t *testing.T) {
	input := `{"provider":"openai","model":"gpt-4o"}
not json
`
	_, err := ParseJSONL(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for invalid JSON line")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("error should reference line 2, got: %v", err)
	}
}

func TestParseJSONLEmpty(t *testing.T) {
	records, err := ParseJSONL(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Fatalf("got %d records, want 0", len(records))
	}
}

func TestCompute(t *testing.T) {
	f, err := os.Open("../../testdata/sample.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	records, err := ParseJSONL(f)
	if err != nil {
		t.Fatal(err)
	}

	stats := Compute(records)
	if len(stats) != 2 {
		t.Fatalf("got %d stat groups, want 2", len(stats))
	}

	openai := stats[0]
	if openai.Provider != "openai" || openai.Model != "gpt-4o" {
		t.Errorf("first group = %s/%s, want openai/gpt-4o", openai.Provider, openai.Model)
	}
	if openai.Count != 5 {
		t.Errorf("openai count = %d, want 5", openai.Count)
	}
	if openai.Errors != 1 {
		t.Errorf("openai errors = %d, want 1", openai.Errors)
	}

	// 4 healthy records: ttft = [95, 110, 120, 200]
	// p50 = interpolated between index 1.5 -> (110+120)/2 = 115
	if !approx(openai.TTFT.P50, 115, 1) {
		t.Errorf("openai TTFT p50 = %.1f, want ~115", openai.TTFT.P50)
	}

	anthropic := stats[1]
	if anthropic.Provider != "anthropic" {
		t.Errorf("second group provider = %q, want anthropic", anthropic.Provider)
	}
	if anthropic.Count != 5 {
		t.Errorf("anthropic count = %d, want 5", anthropic.Count)
	}
	if anthropic.Errors != 1 {
		t.Errorf("anthropic errors = %d, want 1", anthropic.Errors)
	}
}

func TestComputeAllErrors(t *testing.T) {
	records := []Record{
		{Provider: "openai", Model: "gpt-4o", Status: "error"},
		{Provider: "openai", Model: "gpt-4o", Status: "error"},
	}
	stats := Compute(records)
	if len(stats) != 1 {
		t.Fatalf("got %d groups, want 1", len(stats))
	}
	if stats[0].Errors != 2 {
		t.Errorf("errors = %d, want 2", stats[0].Errors)
	}
	if stats[0].TTFT.P50 != 0 {
		t.Errorf("TTFT p50 should be 0 when all errors")
	}
}

func TestRenderMarkdown(t *testing.T) {
	stats := []Stats{
		{
			Provider:     "openai",
			Model:        "gpt-4o",
			Count:        10,
			Errors:       1,
			TTFT:         Percentiles{P50: 120, P95: 200, P99: 250},
			Latency:      Percentiles{P50: 450, P95: 600, P99: 700},
			TokensPerSec: Percentiles{P50: 45.2, P95: 52.0, P99: 55.0},
		},
	}

	var buf bytes.Buffer
	RenderMarkdown(&buf, stats)
	out := buf.String()

	if !strings.Contains(out, "| openai | gpt-4o |") {
		t.Errorf("output missing provider/model row")
	}
	if !strings.Contains(out, "120ms") {
		t.Errorf("output missing TTFT p50 value")
	}
	if !strings.Contains(out, "45.2") {
		t.Errorf("output missing tokens/sec value")
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Errorf("got %d lines, want 3 (header + separator + 1 row)", len(lines))
	}
}

func TestPercentileSingleValue(t *testing.T) {
	p := percentiles([]float64{42})
	if p.P50 != 42 || p.P95 != 42 || p.P99 != 42 {
		t.Errorf("single value percentiles should all be 42, got p50=%.1f p95=%.1f p99=%.1f", p.P50, p.P95, p.P99)
	}
}

func approx(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}
