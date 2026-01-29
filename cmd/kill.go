package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"fp/internal/scan"
	"fp/internal/ui"
	"github.com/spf13/cobra"
)

var (
	killForce   bool
	killSignal  string
	killTimeout time.Duration
	killJSON    bool
	killDryRun  bool
)

var killCmd = &cobra.Command{
	Use:   "kill <port>",
	Short: "Send a signal to processes listening on a port",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		port, err := strconv.Atoi(args[0])
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid port: %q", args[0])
		}

		sig, err := parseSignal(killSignal)
		if err != nil {
			return err
		}

		listeners, err := scan.ListTCPListeners(context.Background())
		if err != nil {
			return err
		}

		var targets []scan.Listener
		seen := make(map[int]bool)
		for _, l := range listeners {
			if l.Port != port {
				continue
			}
			if l.PID <= 0 || seen[l.PID] {
				continue
			}
			seen[l.PID] = true
			targets = append(targets, l)
		}

		if len(targets) == 0 {
			if jsonOutput || killJSON {
				return scan.WriteJSON(os.Stdout, map[string]any{
					"port":     port,
					"status":   "idle",
					"signaled": 0,
				})
			}
			fmt.Fprintf(ui.Stdout(), "%s port %d: nothing to kill\n", ui.LabelWarn(ui.Stdout()), port)
			return nil
		}

		current, _ := user.Current()
		for _, t := range targets {
			if !killForce && current != nil && t.User != "" && t.User != current.Username {
				return fmt.Errorf("refusing to kill pid %d owned by %q (use --force to override)", t.PID, t.User)
			}
		}

		if killDryRun {
			if jsonOutput || killJSON {
				return scan.WriteJSON(os.Stdout, map[string]any{
					"port":    port,
					"status":  "dry-run",
					"targets": targets,
				})
			}
			for _, t := range targets {
				fmt.Fprintf(ui.Stdout(), "%s would signal pid %d (%s)\n", ui.LabelInfo(ui.Stdout()), t.PID, t.Command)
			}
			return nil
		}

		signaled := 0
		for _, t := range targets {
			fmt.Fprintf(ui.Stdout(), "%s sending %s to pid %d (%s)\n", ui.LabelInfo(ui.Stdout()), sig.String(), t.PID, t.Command)
			if err := syscall.Kill(t.PID, sig); err != nil {
				if errors.Is(err, syscall.ESRCH) {
					continue
				}
				return err
			}
			signaled++
		}

		if killTimeout > 0 && sig != syscall.SIGKILL {
			deadline := time.Now().Add(killTimeout)
			for time.Now().Before(deadline) {
				time.Sleep(150 * time.Millisecond)
				stillListening, err := scan.HasTCPListenerOnPort(context.Background(), port)
				if err != nil {
					return err
				}
				if !stillListening {
					if jsonOutput || killJSON {
						return scan.WriteJSON(os.Stdout, map[string]any{
							"port":     port,
							"status":   "signaled",
							"signaled": signaled,
							"signal":   sig.String(),
						})
					}
					return nil
				}
			}

			fmt.Fprintf(ui.Stdout(), "%s port %d still busy after %s; sending SIGKILL\n", ui.LabelWarn(ui.Stdout()), port, killTimeout)
			for _, t := range targets {
				_ = syscall.Kill(t.PID, syscall.SIGKILL)
			}
		}

		if jsonOutput || killJSON {
			return scan.WriteJSON(os.Stdout, map[string]any{
				"port":     port,
				"status":   "signaled",
				"signaled": signaled,
				"signal":   sig.String(),
			})
		}

		return nil
	},
}

func init() {
	killCmd.Flags().BoolVar(&killForce, "force", false, "Allow killing processes not owned by your user")
	killCmd.Flags().StringVar(&killSignal, "signal", "TERM", "Signal to send (TERM, INT, KILL)")
	killCmd.Flags().DurationVar(&killTimeout, "timeout", 2*time.Second, "Wait before escalating to SIGKILL (0 to disable)")
	killCmd.Flags().BoolVar(&killJSON, "json", false, "Output JSON (alias for --json)")
	killCmd.Flags().BoolVar(&killDryRun, "dry-run", false, "Show targets without sending signals")
}

func parseSignal(s string) (syscall.Signal, error) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "TERM", "SIGTERM":
		return syscall.SIGTERM, nil
	case "INT", "SIGINT":
		return syscall.SIGINT, nil
	case "KILL", "SIGKILL":
		return syscall.SIGKILL, nil
	default:
		return 0, fmt.Errorf("unsupported signal: %q", s)
	}
}
