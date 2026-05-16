package report

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
)

type Record struct {
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	Status       string  `json:"status"`
	TTFTMs       int64   `json:"ttft_ms"`
	LatencyMs    int64   `json:"latency_ms"`
	TokensPerSec float64 `json:"tokens_per_sec"`
	TokenCount   int     `json:"token_count"`
	Error        string  `json:"error,omitempty"`
	Timestamp    string  `json:"timestamp"`
}

type Stats struct {
	Provider     string
	Model        string
	Count        int
	Errors       int
	TTFT         Percentiles
	Latency      Percentiles
	TokensPerSec Percentiles
}

type Percentiles struct {
	P50 float64
	P95 float64
	P99 float64
}

func ParseJSONL(r io.Reader) ([]Record, error) {
	var records []Record
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		var rec Record
		if err := json.Unmarshal([]byte(text), &rec); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		records = append(records, rec)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func Compute(records []Record) []Stats {
	type key struct {
		provider string
		model    string
	}

	grouped := make(map[key][]Record)
	order := []key{}

	for _, r := range records {
		k := key{r.Provider, r.Model}
		if _, exists := grouped[k]; !exists {
			order = append(order, k)
		}
		grouped[k] = append(grouped[k], r)
	}

	var results []Stats
	for _, k := range order {
		recs := grouped[k]
		s := Stats{
			Provider: k.provider,
			Model:    k.model,
			Count:    len(recs),
		}

		var ttfts, latencies, tps []float64
		for _, r := range recs {
			if r.Status == "error" {
				s.Errors++
				continue
			}
			ttfts = append(ttfts, float64(r.TTFTMs))
			latencies = append(latencies, float64(r.LatencyMs))
			if r.TokensPerSec > 0 {
				tps = append(tps, r.TokensPerSec)
			}
		}

		s.TTFT = percentiles(ttfts)
		s.Latency = percentiles(latencies)
		s.TokensPerSec = percentiles(tps)

		results = append(results, s)
	}

	return results
}

func RenderMarkdown(w io.Writer, stats []Stats) {
	fmt.Fprintf(w, "| Provider | Model | Probes | Errors | TTFT p50 | TTFT p95 | TTFT p99 | Latency p50 | Latency p95 | Latency p99 | Tok/s p50 | Tok/s p95 | Tok/s p99 |\n")
	fmt.Fprintf(w, "|----------|-------|--------|--------|----------|----------|----------|-------------|-------------|-------------|-----------|-----------|-----------|\n")
	for _, s := range stats {
		fmt.Fprintf(w, "| %s | %s | %d | %d | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			s.Provider,
			s.Model,
			s.Count,
			s.Errors,
			fmtMs(s.TTFT.P50),
			fmtMs(s.TTFT.P95),
			fmtMs(s.TTFT.P99),
			fmtMs(s.Latency.P50),
			fmtMs(s.Latency.P95),
			fmtMs(s.Latency.P99),
			fmtTps(s.TokensPerSec.P50),
			fmtTps(s.TokensPerSec.P95),
			fmtTps(s.TokensPerSec.P99),
		)
	}
}

func fmtMs(v float64) string {
	if v == 0 {
		return "-"
	}
	return fmt.Sprintf("%.0fms", v)
}

func fmtTps(v float64) string {
	if v == 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f", v)
}

func percentiles(data []float64) Percentiles {
	if len(data) == 0 {
		return Percentiles{}
	}
	sort.Float64s(data)
	return Percentiles{
		P50: percentile(data, 50),
		P95: percentile(data, 95),
		P99: percentile(data, 99),
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	rank := (p / 100) * float64(len(sorted)-1)
	lower := int(math.Floor(rank))
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	frac := rank - float64(lower)
	return sorted[lower]*(1-frac) + sorted[upper]*frac
}
