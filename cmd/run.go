package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"freeport/internal/lock"
	"freeport/internal/ports"
	"github.com/spf13/cobra"
)

var (
	runPrefer []int
	runRange  string
	runEnvVar string
)

var runCmd = &cobra.Command{
	Use:   "run -- <cmd...>",
	Short: "Run a command with a chosen PORT (best-effort)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dash := cmd.ArgsLenAtDash()
		if dash < 0 {
			return fmt.Errorf("missing -- before command")
		}
		if dash >= len(args) {
			return fmt.Errorf("missing command after --")
		}

		r, err := ports.ParseRange(runRange)
		if err != nil {
			return err
		}

		commandArgs := args[dash:]

		selectedPort, lockHandle, err := lock.PickAndLockTCPPort(runPrefer, r)
		if err != nil {
			return err
		}
		defer lockHandle.Close()

		fmt.Fprintf(os.Stderr, "freeport: using port %d\n", selectedPort)

		child := exec.Command(commandArgs[0], commandArgs[1:]...)
		child.Stdin = os.Stdin
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		child.Env = append(os.Environ(), fmt.Sprintf("%s=%d", runEnvVar, selectedPort))

		return child.Run()
	},
}

func init() {
	runCmd.Flags().IntSliceVar(&runPrefer, "prefer", []int{3000}, "Preferred ports (tries in order)")
	runCmd.Flags().StringVar(&runRange, "range", "3000-3999", "Port range to search (inclusive)")
	runCmd.Flags().StringVar(&runEnvVar, "env", "PORT", "Environment variable name to set")
}
