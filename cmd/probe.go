package cmd

import (
	"fmt"
	"os"

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
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	engine := probe.NewEngine(cfg)
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

func checkExitCondition(results []probe.Result) error {
	hasError := false
	hasDegraded := false
	for _, r := range results {
		if r.Status == probe.StatusError {
			hasError = true
		}
		if r.Status == probe.StatusDegraded {
			hasDegraded = true
		}
	}

	switch failOn {
	case "error":
		if hasError {
			fmt.Fprintln(os.Stderr, "Exiting with code 1: probe errors detected")
			os.Exit(1)
		}
	case "degraded":
		if hasError || hasDegraded {
			fmt.Fprintln(os.Stderr, "Exiting with code 1: degraded or errored probes detected")
			os.Exit(1)
		}
	}

	return nil
}
