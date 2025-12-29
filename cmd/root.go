package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var jsonOutput bool

var rootCmd = &cobra.Command{
	Use:   "freeport",
	Short: "Local dev port helpers (list/who/kill/pick/run)",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(whoCmd)
	rootCmd.AddCommand(killCmd)
	rootCmd.AddCommand(pickCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(checkCmd)
}
