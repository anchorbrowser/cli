package cli

import "github.com/spf13/cobra"

func newAnchorCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "anchor",
		Short: "Anchor API commands",
	}

	cmd.AddCommand(newSessionCommand(app))
	cmd.AddCommand(newIdentityCommand(app))
	cmd.AddCommand(newTaskCommand(app))

	return cmd
}
