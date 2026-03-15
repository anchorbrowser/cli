package cli

import "github.com/spf13/cobra"

func newCreateAliasCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create resources",
	}
	cmd.AddCommand(newSessionCreateAliasCommand(app))
	cmd.AddCommand(newIdentityCreateAliasCommand(app))
	return cmd
}
