package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"

	"freeport/internal/scan"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List listening TCP ports (best-effort)",
	RunE: func(cmd *cobra.Command, args []string) error {
		listeners, err := scan.ListTCPListeners(context.Background())
		if err != nil {
			return err
		}

		sort.Slice(listeners, func(i, j int) bool {
			if listeners[i].Port != listeners[j].Port {
				return listeners[i].Port < listeners[j].Port
			}
			return listeners[i].PID < listeners[j].PID
		})

		if jsonOutput {
			return scan.WriteJSON(os.Stdout, listeners)
		}

		fmt.Fprintf(os.Stdout, "PORT\tPID\tUSER\tCOMMAND\tADDR\n")
		for _, l := range listeners {
			fmt.Fprintf(os.Stdout, "%d\t%d\t%s\t%s\t%s\n", l.Port, l.PID, l.User, l.Command, l.Address)
		}
		return nil
	},
}
