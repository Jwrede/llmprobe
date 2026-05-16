package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Jwrede/llmprobe/internal/config"
	"github.com/Jwrede/llmprobe/internal/metrics"
	"github.com/Jwrede/llmprobe/internal/output"
	"github.com/Jwrede/llmprobe/internal/probe"
	"github.com/Jwrede/llmprobe/internal/tui"
	"github.com/spf13/cobra"
)

var (
	watchInterval  time.Duration
	watchDuration  time.Duration
	watchCount     int
	useTUI         bool
	loadPath       string
	prometheusAddr string
	otelEndpoint   string
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Continuously probe LLM endpoints on an interval",
	RunE:  runWatch,
}

func init() {
	watchCmd.Flags().DurationVar(&watchInterval, "interval", 60*time.Second, "probe interval (e.g. 30s, 5m)")
	watchCmd.Flags().DurationVar(&watchDuration, "duration", 0, "stop after this duration (e.g. 30s, 5m); 0 means run forever")
	watchCmd.Flags().IntVar(&watchCount, "count", 0, "stop after this many probe cycles; 0 means run forever")
	watchCmd.Flags().BoolVar(&useTUI, "tui", false, "show live terminal dashboard")
	watchCmd.Flags().StringVar(&loadPath, "load", "", "load historical JSONL data into the dashboard")
	watchCmd.Flags().StringVar(&prometheusAddr, "prometheus", "", "expose Prometheus metrics on this address (e.g. :9090)")
	watchCmd.Flags().StringVar(&otelEndpoint, "otel", "", "send OpenTelemetry metrics to this gRPC endpoint (e.g. localhost:4317)")
	rootCmd.AddCommand(watchCmd)
}

var otel *metrics.OTelExporter

func runWatch(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	engine := probe.NewEngine(cfg)

	if prometheusAddr != "" {
		go func() {
			if err := metrics.ServeHTTP(prometheusAddr); err != nil {
				fmt.Fprintf(os.Stderr, "prometheus server: %v\n", err)
			}
		}()
	}

	if otelEndpoint != "" {
		ctx := context.Background()
		var otelErr error
		otel, otelErr = metrics.NewOTelExporter(ctx, otelEndpoint)
		if otelErr != nil {
			return fmt.Errorf("otel setup: %w", otelErr)
		}
		defer otel.Shutdown(ctx)
		fmt.Printf("OpenTelemetry metrics exporting to %s\n", otelEndpoint)
	}

	if useTUI {
		return runWatchTUI(engine)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if watchDuration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, watchDuration)
		defer cancel()
	}

	fmt.Printf("Watching %d endpoints every %s (Ctrl+C to stop)\n\n", countEndpoints(cfg), watchInterval)

	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()

	iterations := 0
	runIteration(engine)
	iterations++

	for {
		if watchCount > 0 && iterations >= watchCount {
			return nil
		}
		select {
		case <-ctx.Done():
			fmt.Println("\nShutting down.")
			return nil
		case <-ticker.C:
			runIteration(engine)
			iterations++
		}
	}
}

func runWatchTUI(engine *probe.Engine) error {
	dash, err := tui.New()
	if err != nil {
		return err
	}

	if loadPath != "" {
		if err := dash.LoadJSONL(loadPath); err != nil {
			return fmt.Errorf("loading %s: %w", loadPath, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	if watchDuration > 0 {
		var timeoutCancel context.CancelFunc
		ctx, timeoutCancel = context.WithTimeout(ctx, watchDuration)
		defer timeoutCancel()
	}

	go func() {
		ticker := time.NewTicker(watchInterval)
		defer ticker.Stop()

		iterations := 0

		results, err := engine.RunAll()
		if err == nil {
			dash.Update(results)
			if prometheusAddr != "" {
				metrics.Record(results)
			}
			if otel != nil {
				otel.Record(ctx, results)
			}
		}
		iterations++

		for {
			if watchCount > 0 && iterations >= watchCount {
				cancel()
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				results, err := engine.RunAll()
				if err == nil {
					dash.Update(results)
					if prometheusAddr != "" {
						metrics.Record(results)
					}
					if otel != nil {
						otel.Record(ctx, results)
					}
				}
				iterations++
			}
		}
	}()

	return dash.Run(ctx, cancel)
}

func runIteration(engine *probe.Engine) {
	results, err := engine.RunAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "probe error: %v\n", err)
		return
	}

	if prometheusAddr != "" {
		metrics.Record(results)
	}
	if otel != nil {
		otel.Record(context.Background(), results)
	}

	timestamp := time.Now().Format("15:04:05")

	switch outputFmt {
	case "json":
		for _, r := range results {
			output.WriteJSONLine(os.Stdout, r)
		}
	default:
		fmt.Printf("[%s] ", timestamp)
		healthy := 0
		degraded := 0
		errors := 0
		for _, r := range results {
			switch r.Status {
			case probe.StatusHealthy:
				healthy++
			case probe.StatusDegraded:
				degraded++
			case probe.StatusError:
				errors++
			}
		}

		if errors == 0 && degraded == 0 {
			fmt.Printf("All %d endpoints healthy.", healthy)
		} else {
			fmt.Printf("%d healthy, %d degraded, %d errors.", healthy, degraded, errors)
		}

		for _, r := range results {
			if r.Status == probe.StatusDegraded {
				fmt.Printf(" DEGRADED: %s/%s (TTFT %dms)", r.Provider, r.Model, r.TTFT.Milliseconds())
			}
			if r.Status == probe.StatusError {
				fmt.Printf(" ERROR: %s/%s (%s)", r.Provider, r.Model, r.Error)
			}
		}
		fmt.Println()
	}
}

func countEndpoints(cfg *config.Config) int {
	n := 0
	for _, p := range cfg.Providers {
		n += len(p.Models)
	}
	return n
}
