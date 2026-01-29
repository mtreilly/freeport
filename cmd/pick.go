package cmd

import (
	"fmt"
	"os"

	"fp/internal/ports"
	"fp/internal/scan"
	"github.com/spf13/cobra"
)

var (
	pickPrefer []int
	pickRange  string
)

var pickCmd = &cobra.Command{
	Use:   "pick",
	Short: "Pick a free TCP port (best-effort)",
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := ports.ParseRange(pickRange)
		if err != nil {
			return err
		}

		chosen, err := ports.PickTCPPort(pickPrefer, r)
		if err != nil {
			return err
		}

		if jsonOutput {
			return scan.WriteJSON(os.Stdout, map[string]int{"port": chosen})
		}

		fmt.Fprintf(os.Stdout, "%d\n", chosen)
		return nil
	},
}

func init() {
	pickCmd.Flags().IntSliceVar(&pickPrefer, "prefer", []int{3000}, "Preferred ports (tries in order; 0 means OS-assigned)")
	pickCmd.Flags().StringVar(&pickRange, "range", "3000-3999", "Port range to search (inclusive)")
}
