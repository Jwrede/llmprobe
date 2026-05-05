package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Jwrede/llmprobe/internal/probe"
)

type jsonResult struct {
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

func PrintJSON(results []probe.Result) error {
	return WriteJSON(os.Stdout, results)
}

func WriteJSON(w io.Writer, results []probe.Result) error {
	out := make([]jsonResult, len(results))
	for i, r := range results {
		out[i] = jsonResult{
			Provider:     r.Provider,
			Model:        r.Model,
			Status:       string(r.Status),
			TTFTMs:       r.TTFT.Milliseconds(),
			LatencyMs:    r.TotalLatency.Milliseconds(),
			TokensPerSec: r.TokensPerSec,
			TokenCount:   r.TokenCount,
			Error:        r.Error,
			Timestamp:    r.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	return nil
}

func WriteJSONLine(w io.Writer, r probe.Result) error {
	out := jsonResult{
		Provider:     r.Provider,
		Model:        r.Model,
		Status:       string(r.Status),
		TTFTMs:       r.TTFT.Milliseconds(),
		LatencyMs:    r.TotalLatency.Milliseconds(),
		TokensPerSec: r.TokensPerSec,
		TokenCount:   r.TokenCount,
		Error:        r.Error,
		Timestamp:    r.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
	}
	data, err := json.Marshal(out)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}
