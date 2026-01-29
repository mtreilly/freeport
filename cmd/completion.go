package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for freeport.

To load completions:

Bash:
  $ source <(freeport completion bash)
  # Or add to ~/.bashrc:
  # eval "$(freeport completion bash)"

Zsh:
  $ source <(freeport completion zsh)
  # Or add to ~/.zshrc:
  # eval "$(freeport completion zsh)"

Fish:
  $ freeport completion fish | source
  # Or save to completions dir:
  # freeport completion fish > ~/.config/fish/completions/freeport.fish

PowerShell:
  PS> freeport completion powershell | Out-String | Invoke-Expression
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
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
}
