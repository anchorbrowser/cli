package cli

import (
	"os"

	"github.com/spf13/cobra"
)

func newInternalCompletionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "__completion",
		Short:  "Generate shell completion scripts",
		Hidden: true,
	}

	cmd.AddCommand(&cobra.Command{
		Use:    "bash",
		Short:  "Generate bash completion script",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenBashCompletionV2(os.Stdout, true)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:    "zsh",
		Short:  "Generate zsh completion script",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenZshCompletion(os.Stdout)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:    "fish",
		Short:  "Generate fish completion script",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:    "powershell",
		Short:  "Generate powershell completion script",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		},
	})

	return cmd
}
