package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for the specified shell.

Examples:
  # Bash
  conoha completion bash > /etc/bash_completion.d/conoha

  # Zsh
  conoha completion zsh > "${fpath[1]}/_conoha"

  # Fish
  conoha completion fish > ~/.config/fish/completions/conoha.fish`,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cmdutil.ExactArgs(1),
	DisableFlagsInUseLine: true,
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
		default:
			return nil
		}
	},
}
