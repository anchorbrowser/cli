package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/spf13/cobra"

	"github.com/anchorbrowser/cli/internal/api"
	"github.com/anchorbrowser/cli/internal/auth"
)

func newAuthCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage API key authentication",
	}
	cmd.AddCommand(newAuthLoginCommand(app))
	cmd.AddCommand(newAuthKeysCommand(app))
	cmd.AddCommand(newAuthCurrentCommand(app))
	return cmd
}

func newAuthLoginCommand(app *App) *cobra.Command {
	var name string
	var apiKey string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store an API key securely in your OS keychain",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := app.ensureAuthStore(); err != nil {
				return err
			}
			if strings.TrimSpace(apiKey) == "" {
				if err := printBanner(app.Stderr); err != nil {
					return err
				}
				secret, err := promptSecret("Enter AnchorBrowser API key: ")
				if err != nil {
					return err
				}
				apiKey = secret
			}
			if err := app.validateAPIKey(cmd.Context(), apiKey); err != nil {
				return fmt.Errorf("api key validation failed: %w", err)
			}
			if err := app.Auth.Login(name, apiKey); err != nil {
				return err
			}
			_, err := fmt.Fprintln(app.Stdout, "Logged in successfully.")
			return err
		},
	}
	cmd.Flags().StringVar(&name, "name", "default", "Name to assign this API key")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key value (if omitted, prompts securely)")
	return cmd
}

func newAuthKeysCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage stored named API keys",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List stored API key names",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := app.ensureAuthStore(); err != nil {
				return err
			}
			names, active, err := app.Auth.List()
			if err != nil {
				return err
			}
			return app.printValue(map[string]any{"keys": names, "active_key": active})
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "use <name>",
		Short: "Set active API key name",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := app.ensureAuthStore(); err != nil {
				return err
			}
			if err := app.Auth.Use(args[0]); err != nil {
				return err
			}
			return app.printValue(map[string]any{"status": "ok", "active_key": normalizeAuthName(args[0])})
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a named API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := app.ensureAuthStore(); err != nil {
				return err
			}
			if err := app.Auth.Remove(args[0]); err != nil {
				return err
			}
			return app.printValue(map[string]any{"status": "ok", "removed": normalizeAuthName(args[0])})
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "rename <old-name> <new-name>",
		Short: "Rename a named API key",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := app.ensureAuthStore(); err != nil {
				return err
			}
			if err := app.Auth.Rename(args[0], args[1]); err != nil {
				return err
			}
			return app.printValue(map[string]any{"status": "ok", "old_name": normalizeAuthName(args[0]), "new_name": normalizeAuthName(args[1])})
		},
	})

	return cmd
}

func newAuthCurrentCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show currently active key name",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := app.ensureAuthStore(); err != nil {
				return err
			}
			active, err := app.Auth.Current()
			if err != nil {
				return err
			}
			return app.printValue(map[string]any{
				"active_key":   active,
				"env_var_name": auth.EnvVarName,
				"env_var_set":  strings.TrimSpace(os.Getenv(auth.EnvVarName)) != "",
			})
		},
	}
}

func promptSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		bytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("read api key from terminal: %w", err)
		}
		value := strings.TrimSpace(string(bytes))
		if value == "" {
			return "", fmt.Errorf("api key cannot be empty")
		}
		return value, nil
	}
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", fmt.Errorf("failed reading api key from stdin")
	}
	value := strings.TrimSpace(scanner.Text())
	if value == "" {
		return "", fmt.Errorf("api key cannot be empty")
	}
	return value, nil
}

func normalizeAuthName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "default"
	}
	return name
}

func (a *App) validateAPIKey(ctx context.Context, apiKey string) error {
	client := a.newAPIClient()

	// Primary validation probe.
	if _, err := client.SessionStatusAll(ctx, apiKey, url.Values{}); err == nil {
		return nil
	} else if isAuthDenied(err) {
		return err
	}

	// Fallback probe for accounts where a single endpoint may be unstable.
	query := url.Values{}
	query.Set("page", "1")
	query.Set("limit", "1")
	_, err := client.SessionHistory(ctx, apiKey, query)
	if err == nil {
		return nil
	}
	return err
}

func isAuthDenied(err error) bool {
	var reqErr *api.RequestError
	if !errors.As(err, &reqErr) {
		return false
	}
	return reqErr.StatusCode == 401 || reqErr.StatusCode == 403
}
