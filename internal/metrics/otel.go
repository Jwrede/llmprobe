package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/Jwrede/llmprobe/internal/probe"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

type OTelExporter struct {
	provider    *metric.MeterProvider
	ttft        otelmetric.Float64Gauge
	latency     otelmetric.Float64Gauge
	tps         otelmetric.Float64Gauge
	tokens      otelmetric.Int64Gauge
	status      otelmetric.Float64Gauge
	probeCount  otelmetric.Int64Counter
	errorCount  otelmetric.Int64Counter
}

func NewOTelExporter(ctx context.Context, endpoint string) (*OTelExporter, error) {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	}

	exporter, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP exporter: %w", err)
	}

	res := resource.NewSchemaless(
		attribute.String("service.name", "llmprobe"),
	)

	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(exporter, metric.WithInterval(30*time.Second))),
	)

	meter := provider.Meter("llmprobe")

	ttft, _ := meter.Float64Gauge("llmprobe.ttft.seconds",
		otelmetric.WithDescription("Time to first token in seconds"))
	latency, _ := meter.Float64Gauge("llmprobe.latency.seconds",
		otelmetric.WithDescription("Total request latency in seconds"))
	tps, _ := meter.Float64Gauge("llmprobe.tokens_per_second",
		otelmetric.WithDescription("Generation throughput in tokens per second"))
	tokens, _ := meter.Int64Gauge("llmprobe.token_count",
		otelmetric.WithDescription("Number of output tokens from last probe"))
	status, _ := meter.Float64Gauge("llmprobe.status",
		otelmetric.WithDescription("Probe status: 1=healthy, 0.5=degraded, 0=error"))
	probeCount, _ := meter.Int64Counter("llmprobe.probes.total",
		otelmetric.WithDescription("Total number of probes executed"))
	errorCount, _ := meter.Int64Counter("llmprobe.errors.total",
		otelmetric.WithDescription("Total number of probe errors"))

	return &OTelExporter{
		provider:   provider,
		ttft:       ttft,
		latency:    latency,
		tps:        tps,
		tokens:     tokens,
		status:     status,
		probeCount: probeCount,
		errorCount: errorCount,
	}, nil
}

func (o *OTelExporter) Record(ctx context.Context, results []probe.Result) {
	for _, r := range results {
		attrs := otelmetric.WithAttributes(
			attribute.String("provider", r.Provider),
			attribute.String("model", r.Model),
		)

		o.probeCount.Add(ctx, 1, attrs)

		switch r.Status {
		case probe.StatusHealthy:
			o.status.Record(ctx, 1, attrs)
		case probe.StatusDegraded:
			o.status.Record(ctx, 0.5, attrs)
		case probe.StatusError:
			o.status.Record(ctx, 0, attrs)
			o.errorCount.Add(ctx, 1, attrs)
		}

		if r.Status != probe.StatusError {
			o.ttft.Record(ctx, r.TTFT.Seconds(), attrs)
			o.latency.Record(ctx, r.TotalLatency.Seconds(), attrs)
			o.tps.Record(ctx, r.TokensPerSec, attrs)
			o.tokens.Record(ctx, int64(r.TokenCount), attrs)
		}
	}
}

func (o *OTelExporter) Shutdown(ctx context.Context) error {
	return o.provider.Shutdown(ctx)
}
