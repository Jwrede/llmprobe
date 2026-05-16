package metrics

import (
	"testing"
	"time"

	"github.com/Jwrede/llmprobe/internal/probe"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestRecord(t *testing.T) {
	results := []probe.Result{
		{
			Provider:     "openai",
			Model:        "gpt-4o",
			Status:       probe.StatusHealthy,
			TTFT:         150 * time.Millisecond,
			TotalLatency: 500 * time.Millisecond,
			TokensPerSec: 42.5,
			TokenCount:   18,
			Timestamp:    time.Now(),
		},
		{
			Provider:  "anthropic",
			Model:     "claude-sonnet-4-20250514",
			Status:    probe.StatusError,
			Error:     "HTTP 500",
			Timestamp: time.Now(),
		},
	}

	Record(results)

	// Check ttft gauge for openai
	var m dto.Metric
	ttftGauge, err := ttftSeconds.GetMetricWith(prometheus.Labels{"provider": "openai", "model": "gpt-4o"})
	if err != nil {
		t.Fatal(err)
	}
	ttftGauge.Write(&m)
	if got := m.GetGauge().GetValue(); got != 0.15 {
		t.Errorf("ttft = %f, want 0.15", got)
	}

	// Check error counter for anthropic
	var mc dto.Metric
	errCounter, err := probeErrors.GetMetricWith(prometheus.Labels{"provider": "anthropic", "model": "claude-sonnet-4-20250514"})
	if err != nil {
		t.Fatal(err)
	}
	errCounter.Write(&mc)
	if got := mc.GetCounter().GetValue(); got != 1 {
		t.Errorf("errors = %f, want 1", got)
	}

	// Check status gauge for error case
	var ms dto.Metric
	statusGauge, err := probeStatus.GetMetricWith(prometheus.Labels{"provider": "anthropic", "model": "claude-sonnet-4-20250514"})
	if err != nil {
		t.Fatal(err)
	}
	statusGauge.Write(&ms)
	if got := ms.GetGauge().GetValue(); got != 0 {
		t.Errorf("status = %f, want 0 for error", got)
	}
}
