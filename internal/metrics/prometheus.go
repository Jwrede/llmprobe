package metrics

import (
	"fmt"
	"net/http"

	"github.com/Jwrede/llmprobe/internal/probe"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ttftSeconds = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "llmprobe_ttft_seconds",
			Help: "Time to first token in seconds.",
		},
		[]string{"provider", "model"},
	)

	latencySeconds = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "llmprobe_latency_seconds",
			Help: "Total request latency in seconds.",
		},
		[]string{"provider", "model"},
	)

	tokensPerSec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "llmprobe_tokens_per_second",
			Help: "Generation throughput in tokens per second.",
		},
		[]string{"provider", "model"},
	)

	tokenCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "llmprobe_token_count",
			Help: "Number of output tokens from last probe.",
		},
		[]string{"provider", "model"},
	)

	probeStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "llmprobe_status",
			Help: "Probe status: 1=healthy, 0.5=degraded, 0=error.",
		},
		[]string{"provider", "model"},
	)

	probeErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llmprobe_errors_total",
			Help: "Total number of probe errors.",
		},
		[]string{"provider", "model"},
	)

	probeCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llmprobe_probes_total",
			Help: "Total number of probes executed.",
		},
		[]string{"provider", "model"},
	)
)

func init() {
	prometheus.MustRegister(
		ttftSeconds,
		latencySeconds,
		tokensPerSec,
		tokenCount,
		probeStatus,
		probeErrors,
		probeCount,
	)
}

func Record(results []probe.Result) {
	for _, r := range results {
		labels := prometheus.Labels{
			"provider": r.Provider,
			"model":    r.Model,
		}

		probeCount.With(labels).Inc()

		switch r.Status {
		case probe.StatusHealthy:
			probeStatus.With(labels).Set(1)
		case probe.StatusDegraded:
			probeStatus.With(labels).Set(0.5)
		case probe.StatusError:
			probeStatus.With(labels).Set(0)
			probeErrors.With(labels).Inc()
		}

		if r.Status != probe.StatusError {
			ttftSeconds.With(labels).Set(r.TTFT.Seconds())
			latencySeconds.With(labels).Set(r.TotalLatency.Seconds())
			tokensPerSec.With(labels).Set(r.TokensPerSec)
			tokenCount.With(labels).Set(float64(r.TokenCount))
		}
	}
}

func ServeHTTP(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	fmt.Printf("Prometheus metrics at http://%s/metrics\n", addr)
	return http.ListenAndServe(addr, mux)
}
