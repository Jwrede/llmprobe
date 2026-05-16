package cmd

import (
	"fmt"
	"os"

	"github.com/Jwrede/llmprobe/internal/report"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report [file.jsonl]",
	Short: "Generate a Markdown summary from JSONL probe data",
	Long:  "Reads a JSONL file produced by 'llmprobe watch --format json' and outputs\na Markdown table with p50/p95/p99 percentiles for TTFT, latency, and throughput.",
	Args:  cobra.ExactArgs(1),
	RunE:  runReport,
}

func init() {
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) error {
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

	stats := report.Compute(records)
	report.RenderMarkdown(os.Stdout, stats)
	return nil
}
