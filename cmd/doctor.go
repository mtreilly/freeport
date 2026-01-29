package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"fp/internal/scan"
	"fp/internal/ui"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system dependencies and configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := ui.Stdout()
		fmt.Fprintf(out, "%s\n\n", ui.Header(out, "fp doctor"))

		// System info
		fmt.Fprintf(out, "%s\n", ui.Info(out, "System"))
		fmt.Fprintf(out, "  OS:       %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Fprintf(out, "  Go:       %s\n\n", runtime.Version())

		// Check port listing tools
		fmt.Fprintf(out, "%s\n", ui.Info(out, "Port listing tools"))

		hasLsof := checkTool("lsof", out)
		hasSS := checkTool("ss", out)

		if !hasLsof && !hasSS {
			fmt.Fprintf(out, "\n  %s No port listing tool found. Install lsof or ss.\n", ui.LabelErr(out))
		}
		fmt.Fprintln(out)

		// Check process tools
		fmt.Fprintf(out, "%s\n", ui.Info(out, "Process tools"))
		checkTool("ps", out)
		checkTool("kill", out)
		fmt.Fprintln(out)

		// Test port scanning
		fmt.Fprintf(out, "%s\n", ui.Info(out, "Port scanning"))
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		start := time.Now()
		listeners, err := scan.ListTCPListeners(ctx)
		elapsed := time.Since(start)

		if err != nil {
			fmt.Fprintf(out, "  %s %v\n", ui.LabelErr(out), err)
		} else {
			fmt.Fprintf(out, "  %s Found %d listeners in %v\n", ui.LabelOK(out), len(listeners), elapsed.Round(time.Millisecond))
		}
		fmt.Fprintln(out)

		// Summary
		fmt.Fprintf(out, "%s\n", ui.Info(out, "Status"))
		if (hasLsof || hasSS) && err == nil {
			fmt.Fprintf(out, "  %s fp is ready to use\n", ui.LabelOK(out))
		} else {
			fmt.Fprintf(out, "  %s Some issues detected (see above)\n", ui.LabelWarn(out))
		}

		return nil
	},
}

func checkTool(name string, out *termenv.Output) bool {
	path, err := exec.LookPath(name)
	if err != nil {
		fmt.Fprintf(out, "  %s %s not found\n", ui.LabelWarn(out), name)
		return false
	}
	fmt.Fprintf(out, "  %s %s (%s)\n", ui.LabelOK(out), name, path)
	return true
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
