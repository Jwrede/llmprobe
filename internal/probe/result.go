package probe

import "time"

type Status string

const (
	StatusHealthy  Status = "healthy"
	StatusDegraded Status = "degraded"
	StatusError    Status = "error"
)

type Result struct {
	Provider     string        `json:"provider"`
	Model        string        `json:"model"`
	TTFT         time.Duration `json:"ttft_ms"`
	TotalLatency time.Duration `json:"latency_ms"`
	TokenCount   int           `json:"token_count"`
	TokensPerSec float64       `json:"tokens_per_sec"`
	Status       Status        `json:"status"`
	Error        string        `json:"error,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
}
