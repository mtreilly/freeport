package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"fp/internal/scan"
	"fp/internal/ui"
	"github.com/spf13/cobra"
)

var whoCmd = &cobra.Command{
	Use:   "who <port>",
	Short: "Show what is listening on a port",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		port, err := strconv.Atoi(args[0])
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid port: %q", args[0])
		}

		listeners, err := scan.ListTCPListeners(context.Background())
		if err != nil {
			return err
		}

		var matches []scan.Listener
		for _, l := range listeners {
			if l.Port == port {
				matches = append(matches, l)
			}
		}

		scan.EnrichListenersWithProcessInfo(context.Background(), matches)

		if jsonOutput {
			return scan.WriteJSON(os.Stdout, matches)
		}

		if len(matches) == 0 {
			fmt.Fprintf(ui.Stdout(), "port %d: %s (no TCP listeners found)\n", port, ui.Success(ui.Stdout(), "free"))
			return nil
		}

		suffix := "listeners"
		if len(matches) == 1 {
			suffix = "listener"
		}
		fmt.Fprintf(ui.Stdout(), "%s %s\n", ui.Header(ui.Stdout(), fmt.Sprintf("port %d", port)), ui.Muted(ui.Stdout(), fmt.Sprintf("(%d %s)", len(matches), suffix)))
		for _, m := range matches {
			fmt.Fprintf(ui.Stdout(), "  %s %d\n", ui.Info(ui.Stdout(), "pid:"), m.PID)
			if m.PPID > 0 {
				fmt.Fprintf(ui.Stdout(), "  %s %d\n", ui.Info(ui.Stdout(), "ppid:"), m.PPID)
			}
			if m.User != "" {
				fmt.Fprintf(ui.Stdout(), "  %s %s\n", ui.Info(ui.Stdout(), "user:"), m.User)
			}
			if m.Command != "" {
				fmt.Fprintf(ui.Stdout(), "  %s %s\n", ui.Info(ui.Stdout(), "cmd:"), ui.Emphasis(ui.Stdout(), m.Command))
			}
			if m.CommandLine != "" {
				fmt.Fprintf(ui.Stdout(), "  %s %q\n", ui.Info(ui.Stdout(), "args:"), m.CommandLine)
			}
			if m.Executable != "" {
				fmt.Fprintf(ui.Stdout(), "  %s %q\n", ui.Info(ui.Stdout(), "exe:"), m.Executable)
			}
			if m.CWD != "" {
				fmt.Fprintf(ui.Stdout(), "  %s %q\n", ui.Info(ui.Stdout(), "cwd:"), m.CWD)
			}
			if m.Address != "" {
				fmt.Fprintf(ui.Stdout(), "  %s %s\n", ui.Info(ui.Stdout(), "addr:"), m.Address)
			}
		}
		return nil
	},
}
