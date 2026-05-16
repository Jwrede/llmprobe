package cmd

import (
	"fmt"
	"os"

	"github.com/Jwrede/llmprobe/internal/baseline"
	"github.com/Jwrede/llmprobe/internal/report"
	"github.com/spf13/cobra"
)

var baselineOutput string

var baselineCmd = &cobra.Command{
	Use:   "baseline [file.jsonl]",
	Short: "Create a baseline file from JSONL probe data",
	Long:  "Reads a JSONL file and computes p50 TTFT and latency per endpoint.\nThe output file can be referenced in probes.yml to enable multiplier-based thresholds.",
	Args:  cobra.ExactArgs(1),
	RunE:  runBaseline,
}

func init() {
	baselineCmd.Flags().StringVarP(&baselineOutput, "output", "o", "baseline.json", "output file path")
	rootCmd.AddCommand(baselineCmd)
}

func runBaseline(cmd *cobra.Command, args []string) error {
	f, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("open %s: %w", args[0], err)
	}
	defer f.Close()

	records, err := report.ParseJSONL(f)
	if err != nil {
		return fmt.Errorf("parse %s: %w", args[0], err)
	}

	if len(records) == 0 {
		return fmt.Errorf("no records in %s", args[0])
	}

	bl := baseline.Create(records)
	if err := bl.Save(baselineOutput); err != nil {
		return err
	}

	fmt.Printf("Baseline written to %s (%d endpoints)\n", baselineOutput, len(bl.Endpoints))
	return nil
}
