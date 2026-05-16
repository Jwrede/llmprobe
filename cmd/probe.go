package cmd

import (
	"fmt"
	"os"

	"github.com/Jwrede/llmprobe/internal/baseline"
	"github.com/Jwrede/llmprobe/internal/config"
	"github.com/Jwrede/llmprobe/internal/output"
	"github.com/Jwrede/llmprobe/internal/probe"
	"github.com/spf13/cobra"
)

var failOn string

var probeCmd = &cobra.Command{
	Use:   "probe",
	Short: "Run a one-off health check against configured LLM endpoints",
	RunE:  runProbe,
}

func init() {
	probeCmd.Flags().StringVar(&failOn, "fail-on", "error", "exit 1 on: error, degraded, none")
	rootCmd.AddCommand(probeCmd)
}

func runProbe(cmd *cobra.Command, args []string) error {
	validFailOn := map[string]bool{"error": true, "degraded": true, "none": true}
	if !validFailOn[failOn] {
		return fmt.Errorf("unknown --fail-on %q (supported: error, degraded, none)", failOn)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	engine, err := buildEngine(cfg)
	if err != nil {
		return err
	}

	results, err := engine.RunAll()
	if err != nil {
		return err
	}

	switch outputFmt {
	case "json":
		if err := output.PrintJSON(results); err != nil {
			return err
		}
	default:
		output.PrintTable(results)
		output.PrintSummary(results)
	}

	return checkExitCondition(results)
}

func buildEngine(cfg *config.Config) (*probe.Engine, error) {
	if cfg.BaselinePath == "" {
		return probe.NewEngine(cfg), nil
	}
	bl, err := baseline.Load(cfg.BaselinePath)
	if err != nil {
		return nil, err
	}
	return probe.NewEngineWithBaseline(cfg, bl), nil
}

func checkExitCondition(results []probe.Result) error {
	var failures []probe.Result
	for _, r := range results {
		switch failOn {
		case "error":
			if r.Status == probe.StatusError {
				failures = append(failures, r)
			}
		case "degraded":
			if r.Status == probe.StatusError || r.Status == probe.StatusDegraded {
				failures = append(failures, r)
			}
		}
	}

	if len(failures) == 0 {
		return nil
	}

	fmt.Fprintf(os.Stderr, "\nFailed endpoints (%d/%d):\n", len(failures), len(results))
	for _, r := range failures {
		if r.Status == probe.StatusError {
			fmt.Fprintf(os.Stderr, "  %s/%s  ERROR  %s\n", r.Provider, r.Model, r.Error)
		} else {
			fmt.Fprintf(os.Stderr, "  %s/%s  DEGRADED  TTFT=%dms  Latency=%dms  Tok/s=%.1f\n",
				r.Provider, r.Model,
				r.TTFT.Milliseconds(),
				r.TotalLatency.Milliseconds(),
				r.TokensPerSec,
			)
		}
	}
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
	return nil
}
