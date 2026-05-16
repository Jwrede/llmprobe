package baseline

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Jwrede/llmprobe/internal/report"
)

type Baseline struct {
	Endpoints map[string]EndpointBaseline `json:"endpoints"`
}

type EndpointBaseline struct {
	TTFTP50    float64 `json:"ttft_p50_ms"`
	LatencyP50 float64 `json:"latency_p50_ms"`
}

func key(provider, model string) string {
	return provider + "/" + model
}

func Create(records []report.Record) *Baseline {
	stats := report.Compute(records)
	b := &Baseline{Endpoints: make(map[string]EndpointBaseline)}
	for _, s := range stats {
		b.Endpoints[key(s.Provider, s.Model)] = EndpointBaseline{
			TTFTP50:    s.TTFT.P50,
			LatencyP50: s.Latency.P50,
		}
	}
	return b
}

func Load(path string) (*Baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading baseline: %w", err)
	}
	var b Baseline
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("parsing baseline: %w", err)
	}
	return &b, nil
}

func (b *Baseline) Save(path string) error {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (b *Baseline) Lookup(provider, model string) (EndpointBaseline, bool) {
	ep, ok := b.Endpoints[key(provider, model)]
	return ep, ok
}
