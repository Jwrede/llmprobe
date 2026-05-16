package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "1.2.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the llmprobe version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("llmprobe %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
