package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"freeport/internal/scan"
	"github.com/spf13/cobra"
)

var checkWait time.Duration

var checkCmd = &cobra.Command{
	Use:   "check <port>",
	Short: "Check if a TCP port is free (exit 0 if free, 1 if in-use, 2 on error)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		port, err := strconv.Atoi(args[0])
		if err != nil || port < 1 || port > 65535 {
			fmt.Fprintf(os.Stderr, "invalid port: %q\n", args[0])
			os.Exit(2)
		}

		inUse, err := waitForPortFree(port, checkWait)
		if err != nil {
			fmt.Fprintf(os.Stderr, "check failed: %v\n", err)
			os.Exit(2)
		}

		status := "free"
		if inUse {
			status = "in-use"
		}

		if jsonOutput {
			_ = scan.WriteJSON(os.Stdout, map[string]any{
				"port":   port,
				"status": status,
				"in_use": inUse,
			})
		} else {
			fmt.Fprintf(os.Stdout, "port %d: %s\n", port, status)
		}

		if inUse {
			os.Exit(1)
		}
	},
}

func init() {
	checkCmd.Flags().DurationVar(&checkWait, "wait", 0, "Wait for port to become free (e.g., 2s)")
}

func waitForPortFree(port int, wait time.Duration) (bool, error) {
	if wait <= 0 {
		return scan.HasTCPListenerOnPort(context.Background(), port)
	}

	deadline := time.Now().Add(wait)
	for {
		inUse, err := scan.HasTCPListenerOnPort(context.Background(), port)
		if err != nil {
			return false, err
		}
		if !inUse {
			return false, nil
		}
		if time.Now().After(deadline) {
			return true, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
}
