package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"fp/internal/scan"
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for fp.

To load completions:

Bash:
  $ source <(fp completion bash)
  # Or add to ~/.bashrc:
  # eval "$(fp completion bash)"

Zsh:
  $ source <(fp completion zsh)
  # Or add to ~/.zshrc:
  # eval "$(fp completion zsh)"

Fish:
  $ fp completion fish | source
  # Or save to completions dir:
  # fp completion fish > ~/.config/fish/completions/fp.fish

PowerShell:
  PS> fp completion powershell | Out-String | Invoke-Expression
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletionV2(os.Stdout, true)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
	whoCmd.ValidArgsFunction = completeListeningPorts
	killCmd.ValidArgsFunction = completeListeningPorts
}

func completeListeningPorts(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	listeners, err := scan.ListTCPListeners(cmd.Context())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	descriptions := make(map[int]map[string]bool)
	for _, listener := range listeners {
		if !strings.HasPrefix(strconv.Itoa(listener.Port), toComplete) {
			continue
		}
		if descriptions[listener.Port] == nil {
			descriptions[listener.Port] = make(map[string]bool)
		}
		// Keep process names on one line for Cobra's tab-separated descriptions.
		description := strings.Join(strings.Fields(listener.Command), " ")
		if listener.PID > 0 {
			if description != "" {
				description += " "
			}
			description += fmt.Sprintf("(pid %d)", listener.PID)
		}
		if description != "" {
			descriptions[listener.Port][description] = true
		}
	}

	ports := make([]int, 0, len(descriptions))
	for port := range descriptions {
		ports = append(ports, port)
	}
	sort.Ints(ports)
	completions := make([]string, 0, len(ports))
	for _, port := range ports {
		var processDescriptions []string
		for description := range descriptions[port] {
			processDescriptions = append(processDescriptions, description)
		}
		sort.Strings(processDescriptions)
		completion := strconv.Itoa(port)
		if len(processDescriptions) > 0 {
			completion += "\t" + strings.Join(processDescriptions, "; ")
		}
		completions = append(completions, completion)
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}
