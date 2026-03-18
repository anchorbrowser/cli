package cli

import "github.com/spf13/cobra"

func newProxyCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:                "proxy [agent-browser args...]",
		Short:              "Run agent-browser commands through Anchor proxy",
		DisableFlagParsing: true,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				args = []string{"--help"}
			}
			parsed, err := parseParityArgs(args)
			if err != nil {
				return err
			}
			applyParityGlobals(app, parsed.Global)
			return runParityCommandFn(app, parsed)
		},
	}
}
