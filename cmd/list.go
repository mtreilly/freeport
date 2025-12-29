package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"

	"freeport/internal/scan"
	"freeport/internal/ui"
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

		if listPort > 0 {
			filtered := listeners[:0]
			for _, l := range listeners {
				if l.Port == listPort {
					filtered = append(filtered, l)
				}
			}
			listeners = filtered
		}

		if listUnique {
			seen := make(map[string]bool)
			filtered := listeners[:0]
			for _, l := range listeners {
				key := fmt.Sprintf("%d:%d", l.Port, l.PID)
				if seen[key] {
					continue
				}
				seen[key] = true
				filtered = append(filtered, l)
			}
			listeners = filtered
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

		fmt.Fprintf(ui.Stdout(), "%s\n", ui.Header(ui.Stdout(), "PORT\tPID\tUSER\tCOMMAND\tADDR"))
		for _, l := range listeners {
			port := ui.Emphasis(ui.Stdout(), fmt.Sprintf("%d", l.Port))
			command := ui.Emphasis(ui.Stdout(), l.Command)
			fmt.Fprintf(ui.Stdout(), "%s\t%d\t%s\t%s\t%s\n", port, l.PID, l.User, command, l.Address)
		}
		return nil
	},
}

var (
	listPort   int
	listUnique bool
)

func init() {
	listCmd.Flags().IntVar(&listPort, "port", 0, "Filter by port")
	listCmd.Flags().BoolVar(&listUnique, "unique", false, "Deduplicate by port+PID")
}
