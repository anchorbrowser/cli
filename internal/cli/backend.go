package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/anchorbrowser/cli/internal/backend"
	"github.com/anchorbrowser/cli/internal/config"
)

func newBackendCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backend",
		Short: "Manage embedded agent-browser backend",
	}

	cmd.AddCommand(newBackendInstallCommand(app))
	cmd.AddCommand(newBackendStatusCommand(app))
	cmd.AddCommand(newBackendPathCommand(app))
	cmd.AddCommand(newBackendUninstallCommand(app))
	cmd.AddCommand(newBackendDoctorCommand(app))
	return cmd
}

func newBackendInstallCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install pinned agent-browser backend",
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager, err := backend.NewManager(config.DefaultAppName)
			if err != nil {
				return err
			}
			path, err := manager.Install(cmd.Context())
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(app.Stdout, "Installed backend %s at %s\n", backend.PinnedVersion, path)
			return nil
		},
	}
}

func newBackendStatusCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show backend installation status",
		RunE: func(_ *cobra.Command, _ []string) error {
			manager, err := backend.NewManager(config.DefaultAppName)
			if err != nil {
				return err
			}
			status, err := manager.Status()
			if err != nil {
				return err
			}
			return app.printValue(status)
		},
	}
}

func newBackendPathCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print backend executable path",
		RunE: func(_ *cobra.Command, _ []string) error {
			manager, err := backend.NewManager(config.DefaultAppName)
			if err != nil {
				return err
			}
			path, err := manager.BinaryPath()
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(app.Stdout, path)
			return nil
		},
	}
}

func newBackendUninstallCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove installed backend binaries",
		RunE: func(_ *cobra.Command, _ []string) error {
			manager, err := backend.NewManager(config.DefaultAppName)
			if err != nil {
				return err
			}
			if err := manager.Uninstall(); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(app.Stdout, "Removed installed backend binaries.")
			return nil
		},
	}
}

func newBackendDoctorCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run backend diagnostics",
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager, err := backend.NewManager(config.DefaultAppName)
			if err != nil {
				return err
			}
			report, err := manager.Doctor(cmd.Context())
			if err != nil {
				return err
			}
			return app.printValue(report)
		},
	}
}
