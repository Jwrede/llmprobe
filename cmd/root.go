package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	configPath  string
	outputFmt   string
	verbose     bool
)

var rootCmd = &cobra.Command{
	Use:   "llmprobe",
	Short: "Probe LLM API endpoints for health and performance",
	Long:  "llmprobe measures TTFT, latency, and throughput of LLM API endpoints.\nUse it as a one-off check, a continuous monitor, or a CI gate.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		validFormats := map[string]bool{"table": true, "json": true}
		if !validFormats[outputFmt] {
			return fmt.Errorf("unknown --format %q (supported: table, json)", outputFmt)
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "probes.yml", "path to config file")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "format", "f", "table", "output format (table, json)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
