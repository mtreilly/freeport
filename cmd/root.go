package cmd

import (
	"os"

	"freeport/internal/ui"
	"github.com/spf13/cobra"
)

var jsonOutput bool
var noColor bool

var rootCmd = &cobra.Command{
	Use:   "freeport",
	Short: "Local dev port helpers (list/who/kill/pick/run)",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		ui.Configure(noColor)
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable ANSI colors")
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(whoCmd)
	rootCmd.AddCommand(killCmd)
	rootCmd.AddCommand(pickCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(checkCmd)
}
