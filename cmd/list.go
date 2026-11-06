package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fp/internal/scan"
	"fp/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [filter]",
	Short: "List listening TCP ports (best-effort)",
	Long: `List listening TCP ports (best-effort).

Optional filter argument matches against command name, executable path,
and command line (case-insensitive).

Examples:
  fp list           # all ports
  fp list node      # ports used by node processes
  fp list python    # ports used by python processes
  fp list redis     # ports used by redis`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var cwd string
		if cmd.Flags().Changed("cwd") {
			abs, err := filepath.Abs(listCWD)
			if err != nil {
				return err
			}
			cwd, err = filepath.EvalSymlinks(abs)
			if err != nil {
				return err
			}
			info, err := os.Stat(cwd)
			if err != nil {
				return err
			}
			if !info.IsDir() {
				return fmt.Errorf("not a directory: %q", listCWD)
			}
		}

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

		if filter != "" || cwd != "" || listVerbose {
			scan.EnrichListenersWithProcessInfo(context.Background(), listeners)
		}

		if cwd != "" {
			filtered := listeners[:0]
			for _, l := range listeners {
				if l.CWD == "" {
					continue
				}
				processCWD, err := filepath.EvalSymlinks(l.CWD)
				if err != nil {
					continue
				}
				rel, err := filepath.Rel(cwd, processCWD)
				if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
					filtered = append(filtered, l)
				}
			}
			listeners = filtered
		}

		if filter != "" {
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
	listCWD     string
)

func init() {
	listCmd.Flags().IntVar(&listPort, "port", 0, "Filter by port")
	listCmd.Flags().StringVar(&listCWD, "cwd", "", "Filter by process working directory (including subdirectories)")
	listCmd.MarkFlagDirname("cwd")
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
