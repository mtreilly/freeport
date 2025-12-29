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

		if jsonOutput {
			return scan.WriteJSON(os.Stdout, matches)
		}

		if len(matches) == 0 {
			fmt.Fprintf(ui.Stdout(), "port %d: %s (no TCP listeners found)\n", port, ui.Success(ui.Stdout(), "free"))
			return nil
		}

		for _, m := range matches {
			fmt.Fprintf(ui.Stdout(), "port %d: pid=%d user=%s cmd=%s addr=%s\n", port, m.PID, m.User, m.Command, m.Address)
		}
		return nil
	},
}
