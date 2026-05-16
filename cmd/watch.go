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
	useTUI         bool
	loadPath       string
	prometheusAddr string
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Continuously probe LLM endpoints on an interval",
	RunE:  runWatch,
}

func init() {
	watchCmd.Flags().DurationVar(&watchInterval, "interval", 60*time.Second, "probe interval (e.g. 30s, 5m)")
	watchCmd.Flags().BoolVar(&useTUI, "tui", false, "show live terminal dashboard")
	watchCmd.Flags().StringVar(&loadPath, "load", "", "load historical JSONL data into the dashboard")
	watchCmd.Flags().StringVar(&prometheusAddr, "prometheus", "", "expose Prometheus metrics on this address (e.g. :9090)")
	rootCmd.AddCommand(watchCmd)
}

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

	if useTUI {
		return runWatchTUI(engine)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Printf("Watching %d endpoints every %s (Ctrl+C to stop)\n\n", countEndpoints(cfg), watchInterval)

	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()

	runIteration(engine)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nShutting down.")
			return nil
		case <-ticker.C:
			runIteration(engine)
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

	go func() {
		ticker := time.NewTicker(watchInterval)
		defer ticker.Stop()

		results, err := engine.RunAll()
		if err == nil {
			dash.Update(results)
			if prometheusAddr != "" {
				metrics.Record(results)
			}
		}

		for {
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
				}
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
