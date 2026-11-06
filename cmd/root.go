package cmd

import (
	"errors"
	"os"
	"os/exec"
	"syscall"

	"fp/internal/ui"
	"github.com/spf13/cobra"
)

var jsonOutput bool
var noColor bool

var rootCmd = &cobra.Command{
	Use:   "fp",
	Short: "Local dev port helpers (list/who/kill/pick/run)",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		ui.Configure(noColor)
		return nil
	},
}

func Execute() {
	cmd, err := rootCmd.ExecuteC()
	if err != nil {
		if cmd == checkCmd {
			os.Exit(2)
		}
		var exitErr *exec.ExitError
		if cmd == runCmd && errors.As(err, &exitErr) {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
				os.Exit(128 + int(status.Signal()))
			}
			os.Exit(exitErr.ExitCode())
		}
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
	checkCmd.MarkFlagsMutuallyExclusive("quiet", "json")
}
