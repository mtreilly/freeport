package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"freeport/internal/scan"
	"freeport/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [filter]",
	Short: "List listening TCP ports (best-effort)",
	Long: `List listening TCP ports (best-effort).

Optional filter argument matches against command name, executable path,
and command line (case-insensitive).

Examples:
  freeport list           # all ports
  freeport list node      # ports used by node processes
  freeport list python    # ports used by python processes
  freeport list redis     # ports used by redis`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		listeners, err := scan.ListTCPListeners(context.Background())
		if err != nil {
			return err
		}

		var filter string
		if len(args) > 0 {
			filter = strings.ToLower(args[0])
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

		if filter != "" {
			// Enrich for better filtering if not already verbose
			if !listVerbose {
				scan.EnrichListenersWithProcessInfo(context.Background(), listeners)
			}
			filtered := listeners[:0]
			for _, l := range listeners {
				if matchesFilter(l, filter) {
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

		if listVerbose {
			scan.EnrichListenersWithProcessInfo(context.Background(), listeners)
		}

		if jsonOutput {
			return scan.WriteJSON(os.Stdout, listeners)
		}

		if listVerbose {
			fmt.Fprintf(ui.Stdout(), "%s\n", ui.Header(ui.Stdout(), "PORT\tPID\tUSER\tEXE"))
			for _, l := range listeners {
				port := ui.Emphasis(ui.Stdout(), fmt.Sprintf("%d", l.Port))
				exe := truncatePath(l.CommandLine, 60)
				if exe == "" {
					exe = l.Command
				}
				fmt.Fprintf(ui.Stdout(), "%s\t%d\t%s\t%s\n", port, l.PID, l.User, exe)
			}
		} else {
			fmt.Fprintf(ui.Stdout(), "%s\n", ui.Header(ui.Stdout(), "PORT\tPID\tUSER\tCOMMAND\tADDR"))
			for _, l := range listeners {
				port := ui.Emphasis(ui.Stdout(), fmt.Sprintf("%d", l.Port))
				command := ui.Emphasis(ui.Stdout(), l.Command)
				fmt.Fprintf(ui.Stdout(), "%s\t%d\t%s\t%s\t%s\n", port, l.PID, l.User, command, l.Address)
			}
		}
		return nil
	},
}

var (
	listPort    int
	listUnique  bool
	listVerbose bool
)

func init() {
	listCmd.Flags().IntVar(&listPort, "port", 0, "Filter by port")
	listCmd.Flags().BoolVar(&listUnique, "unique", false, "Deduplicate by port+PID")
	listCmd.Flags().BoolVarP(&listVerbose, "verbose", "v", false, "Show executable path")
}

func truncatePath(cmdLine string, maxLen int) string {
	if cmdLine == "" {
		return ""
	}

	// Find where arguments likely start (first " -" pattern)
	exe := cmdLine
	if idx := strings.Index(cmdLine, " -"); idx > 0 {
		exe = cmdLine[:idx]
	}

	if maxLen > 0 && len(exe) > maxLen {
		return "..." + exe[len(exe)-maxLen+3:]
	}
	return exe
}

func matchesFilter(l scan.Listener, filter string) bool {
	// Match against command name, executable, or command line
	if strings.Contains(strings.ToLower(l.Command), filter) {
		return true
	}
	if strings.Contains(strings.ToLower(l.Executable), filter) {
		return true
	}
	if strings.Contains(strings.ToLower(l.CommandLine), filter) {
		return true
	}
	return false
}
