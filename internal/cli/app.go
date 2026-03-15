package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/anchorbrowser/cli/internal/api"
	"github.com/anchorbrowser/cli/internal/auth"
	"github.com/anchorbrowser/cli/internal/config"
	"github.com/anchorbrowser/cli/internal/output"
)

// GlobalOptions are shared command flags.
type GlobalOptions struct {
	APIKey  string
	KeyName string
	BaseURL string
	Timeout time.Duration
	Output  string
	Compact bool
	DryRun  bool
	Verbose bool
}

// App holds long-lived dependencies for command handlers.
type App struct {
	Version string
	Global  *GlobalOptions
	Config  *config.Manager
	Auth    *auth.Store
	Stdout  io.Writer
	Stderr  io.Writer
}

func NewRootCommand(version string) (*cobra.Command, error) {
	cfgManager, err := config.NewManager(config.DefaultAppName)
	if err != nil {
		return nil, err
	}

	global := &GlobalOptions{}
	var showVersion bool
	app := &App{
		Version: version,
		Global:  global,
		Config:  cfgManager,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}

	cmd := &cobra.Command{
		Use:           "anchorbrowser",
		Short:         "AnchorBrowser CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if err := output.ValidateFormat(global.Output); err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if showVersion {
				return app.printVersionInfo(cmd.Context())
			}
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().BoolVar(&showVersion, "version", false, "Print version information")
	cmd.PersistentFlags().StringVar(&global.APIKey, "api-key", "", "API key value (highest precedence)")
	cmd.PersistentFlags().StringVar(&global.KeyName, "key", "", "Named API key profile to use")
	cmd.PersistentFlags().StringVar(&global.BaseURL, "base-url", "https://api.anchorbrowser.io", "API base URL")
	cmd.PersistentFlags().DurationVar(&global.Timeout, "timeout", 30*time.Second, "HTTP request timeout")
	cmd.PersistentFlags().StringVar(&global.Output, "output", "json", "Output format: json|yaml")
	cmd.PersistentFlags().BoolVar(&global.Compact, "compact", false, "Compact output (json only)")
	cmd.PersistentFlags().BoolVar(&global.DryRun, "dry-run", false, "Print request payloads without sending API calls")
	cmd.PersistentFlags().BoolVar(&global.Verbose, "verbose", false, "Verbose request logging")

	cmd.AddCommand(newAuthCommand(app))
	cmd.AddCommand(newSessionCommand(app))
	cmd.AddCommand(newTaskCommand(app))
	cmd.AddCommand(newIdentityCommand(app))
	cmd.AddCommand(newCreateAliasCommand(app))
	cmd.AddCommand(newVersionCommand(app))

	return cmd, nil
}

func (a *App) ensureAuthStore() error {
	if a.Auth != nil {
		return nil
	}
	store, err := auth.NewStore(a.Config)
	if err != nil {
		return err
	}
	a.Auth = store
	return nil
}

func (a *App) resolveAPIKey() (*auth.ResolvedKey, error) {
	if strings.TrimSpace(a.Global.APIKey) != "" {
		return &auth.ResolvedKey{Value: a.Global.APIKey, Source: "flag:api-key"}, nil
	}

	if strings.TrimSpace(a.Global.KeyName) == "" {
		envValue := strings.TrimSpace(os.Getenv(auth.EnvVarName))
		if envValue != "" {
			return &auth.ResolvedKey{Value: envValue, Source: "env:" + auth.EnvVarName}, nil
		}
	}

	if err := a.ensureAuthStore(); err != nil {
		return nil, err
	}
	resolved, err := a.Auth.Resolve("", a.Global.KeyName, "")
	if err != nil {
		if errors.Is(err, auth.ErrNoAPIKeyConfigured) {
			return nil, fmt.Errorf("no API key configured. run `anchorbrowser auth login`, set --api-key, --key, or %s", auth.EnvVarName)
		}
		return nil, err
	}
	return resolved, nil
}

func (a *App) newAPIClient() *api.Client {
	return api.New(api.Options{
		BaseURL: strings.TrimSpace(a.Global.BaseURL),
		Timeout: a.Global.Timeout,
		DryRun:  a.Global.DryRun,
		Verbose: a.Global.Verbose,
		Out:     a.Stdout,
	})
}

func (a *App) printValue(v any) error {
	printer := output.Printer{Format: a.Global.Output, Compact: a.Global.Compact, Writer: a.Stdout}
	return printer.Print(v)
}

func (a *App) printDryRunOrValue(v any, err error) error {
	if err != nil {
		if errors.Is(err, api.ErrDryRun) {
			return nil
		}
		return err
	}
	return a.printValue(v)
}
