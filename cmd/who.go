package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"freeport/internal/scan"
	"freeport/internal/ui"
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

		for _, m := range matches {
			line := fmt.Sprintf("port %d: pid=%d", port, m.PID)
			if m.PPID > 0 {
				line += fmt.Sprintf(" ppid=%d", m.PPID)
			}
			if m.User != "" {
				line += fmt.Sprintf(" user=%s", m.User)
			}
			if m.Command != "" {
				line += fmt.Sprintf(" cmd=%s", m.Command)
			}
			if m.Address != "" {
				line += fmt.Sprintf(" addr=%s", m.Address)
			}
			fmt.Fprintf(ui.Stdout(), "%s\n", line)
			if m.CommandLine != "" {
				fmt.Fprintf(ui.Stdout(), "  args=%q\n", m.CommandLine)
			}
			if m.Executable != "" {
				fmt.Fprintf(ui.Stdout(), "  exe=%q\n", m.Executable)
			}
			if m.CWD != "" {
				fmt.Fprintf(ui.Stdout(), "  cwd=%q\n", m.CWD)
			}
		}
		return nil
	},
}
