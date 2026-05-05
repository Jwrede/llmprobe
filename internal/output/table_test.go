package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Jwrede/llmprobe/internal/probe"
)

func TestWriteTableHealthy(t *testing.T) {
	results := []probe.Result{
		{
			Provider:     "openai",
			Model:        "gpt-4o",
			TTFT:         312 * time.Millisecond,
			TotalLatency: 2100 * time.Millisecond,
			TokenCount:   42,
			TokensPerSec: 68.4,
			Status:       probe.StatusHealthy,
		},
	}

	var buf bytes.Buffer
	WriteTable(&buf, results)
	out := buf.String()

	if !strings.Contains(out, "openai") {
		t.Error("missing provider name")
	}
	if !strings.Contains(out, "gpt-4o") {
		t.Error("missing model name")
	}
	if !strings.Contains(out, "healthy") {
		t.Error("missing status")
	}
	if !strings.Contains(out, "312ms") {
		t.Error("missing TTFT")
	}

	for _, r := range out {
		if r > 127 {
			t.Errorf("non-ASCII character in output: %U", r)
		}
	}
}

func TestWriteTableError(t *testing.T) {
	results := []probe.Result{
		{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
			Status:   probe.StatusError,
			Error:    "connection refused",
		},
	}

	var buf bytes.Buffer
	WriteTable(&buf, results)
	out := buf.String()

	if !strings.Contains(out, "error") {
		t.Error("missing error status")
	}
	if !strings.Contains(out, "connection refused") {
		t.Error("missing error message")
	}
}
