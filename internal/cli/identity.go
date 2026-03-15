package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newIdentityCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{Use: "identity", Short: "Manage identities"}
	cmd.AddCommand(newIdentityCreateCommand(app))
	cmd.AddCommand(newIdentityGetCommand(app))
	cmd.AddCommand(newIdentityUpdateCommand(app))
	cmd.AddCommand(newIdentityDeleteCommand(app))
	cmd.AddCommand(newIdentityCredentialsCommand(app))
	return cmd
}

func newIdentityCreateCommand(app *App) *cobra.Command {
	return newIdentityCreateCommandWithUse(app, "create", "Create an identity")
}

func newIdentityCreateCommandWithUse(app *App, use, short string) *cobra.Command {
	var bodyPath string
	var validateAsync bool
	var name, source, metadataJSON, appName, appDescription string
	var username, password, authenticatorSecret, authenticatorOTP string
	var customFields []string

	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}

			payload, err := parseBodyAsMap(bodyPath)
			if err != nil {
				return err
			}
			if payload == nil {
				payload = map[string]any{}
			}

			if cmd.Flags().Changed("name") {
				payload["name"] = name
			}
			if cmd.Flags().Changed("source") {
				payload["source"] = source
			}
			if cmd.Flags().Changed("application-name") {
				payload["applicationName"] = appName
			}
			if cmd.Flags().Changed("application-description") {
				payload["applicationDescription"] = appDescription
			}
			if cmd.Flags().Changed("metadata") {
				meta, err := parseJSONObjectFlag(metadataJSON)
				if err != nil {
					return fmt.Errorf("parse --metadata: %w", err)
				}
				payload["metadata"] = meta
			}

			credentials, err := buildCredentialsFromFlags(cmd, username, password, authenticatorSecret, authenticatorOTP, customFields)
			if err != nil {
				return err
			}
			if len(credentials) > 0 {
				payload["credentials"] = credentials
			}

			if _, ok := payload["source"]; !ok {
				return fmt.Errorf("identity source is required (set --source or include it in --body)")
			}
			if _, ok := payload["credentials"]; !ok {
				return fmt.Errorf("identity credentials are required (set credential flags or include them in --body)")
			}

			result, err := app.newAPIClient().IdentityCreate(cmd.Context(), resolved.Value, validateAsync, payload)
			return app.printDryRunOrValue(result, err)
		},
	}

	cmd.Flags().StringVar(&bodyPath, "body", "", "Path to JSON/YAML body file, '-' for stdin, or inline JSON")
	cmd.Flags().BoolVar(&validateAsync, "validate-async", true, "Validate identity asynchronously")
	cmd.Flags().StringVar(&name, "name", "", "Identity name")
	cmd.Flags().StringVar(&source, "source", "", "Source login URL")
	cmd.Flags().StringVar(&metadataJSON, "metadata", "", "Metadata JSON object")
	cmd.Flags().StringVar(&appName, "application-name", "", "Application name")
	cmd.Flags().StringVar(&appDescription, "application-description", "", "Application description")
	cmd.Flags().StringVar(&username, "username", "", "Username for username_password credential")
	cmd.Flags().StringVar(&password, "password", "", "Password for username_password credential")
	cmd.Flags().StringVar(&authenticatorSecret, "authenticator-secret", "", "TOTP secret for authenticator credential")
	cmd.Flags().StringVar(&authenticatorOTP, "authenticator-otp", "", "One-time code for authenticator credential")
	cmd.Flags().StringSliceVar(&customFields, "custom-field", nil, "Custom credential field key=value (repeatable)")

	return cmd
}

func newIdentityGetCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "get <identity-id>",
		Short: "Get identity details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			result, err := app.newAPIClient().IdentityGet(cmd.Context(), resolved.Value, args[0])
			return app.printDryRunOrValue(result, err)
		},
	}
}

func newIdentityUpdateCommand(app *App) *cobra.Command {
	var bodyPath string
	var name, metadataJSON, username, password, authenticatorSecret, authenticatorOTP string
	var customFields []string

	cmd := &cobra.Command{
		Use:   "update <identity-id>",
		Short: "Update identity fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}

			payload, err := parseBodyAsMap(bodyPath)
			if err != nil {
				return err
			}
			if payload == nil {
				payload = map[string]any{}
			}

			if cmd.Flags().Changed("name") {
				payload["name"] = name
			}
			if cmd.Flags().Changed("metadata") {
				meta, err := parseJSONObjectFlag(metadataJSON)
				if err != nil {
					return fmt.Errorf("parse --metadata: %w", err)
				}
				payload["metadata"] = meta
			}
			credentials, err := buildCredentialsFromFlags(cmd, username, password, authenticatorSecret, authenticatorOTP, customFields)
			if err != nil {
				return err
			}
			if len(credentials) > 0 {
				payload["credentials"] = credentials
			}

			if len(payload) == 0 {
				return fmt.Errorf("no update fields provided")
			}

			result, err := app.newAPIClient().IdentityUpdate(cmd.Context(), resolved.Value, args[0], payload)
			return app.printDryRunOrValue(result, err)
		},
	}

	cmd.Flags().StringVar(&bodyPath, "body", "", "Path to JSON/YAML body file, '-' for stdin, or inline JSON")
	cmd.Flags().StringVar(&name, "name", "", "Identity name")
	cmd.Flags().StringVar(&metadataJSON, "metadata", "", "Metadata JSON object")
	cmd.Flags().StringVar(&username, "username", "", "Username for username_password credential")
	cmd.Flags().StringVar(&password, "password", "", "Password for username_password credential")
	cmd.Flags().StringVar(&authenticatorSecret, "authenticator-secret", "", "TOTP secret for authenticator credential")
	cmd.Flags().StringVar(&authenticatorOTP, "authenticator-otp", "", "One-time code for authenticator credential")
	cmd.Flags().StringSliceVar(&customFields, "custom-field", nil, "Custom credential field key=value (repeatable)")

	return cmd
}

func newIdentityDeleteCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <identity-id>",
		Short: "Delete identity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			result, err := app.newAPIClient().IdentityDelete(cmd.Context(), resolved.Value, args[0])
			return app.printDryRunOrValue(result, err)
		},
	}
}

func newIdentityCredentialsCommand(app *App) *cobra.Command {
	var reveal bool
	cmd := &cobra.Command{
		Use:   "credentials <identity-id>",
		Short: "Get identity credentials (redacted by default)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			result, err := app.newAPIClient().IdentityCredentials(cmd.Context(), resolved.Value, args[0])
			if err != nil {
				return app.printDryRunOrValue(result, err)
			}
			return app.printValue(redactSensitive(result, reveal))
		},
	}
	cmd.Flags().BoolVar(&reveal, "reveal-secrets", false, "Show sensitive credential values")
	return cmd
}

func buildCredentialsFromFlags(cmd *cobra.Command, username, password, authenticatorSecret, authenticatorOTP string, customFields []string) ([]any, error) {
	credentials := []any{}
	if cmd.Flags().Changed("username") || cmd.Flags().Changed("password") {
		if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
			return nil, fmt.Errorf("--username and --password must both be set")
		}
		credentials = append(credentials, map[string]any{
			"type":     "username_password",
			"username": username,
			"password": password,
		})
	}
	if cmd.Flags().Changed("authenticator-secret") || cmd.Flags().Changed("authenticator-otp") {
		if strings.TrimSpace(authenticatorSecret) == "" {
			return nil, fmt.Errorf("--authenticator-secret is required when setting authenticator credential")
		}
		auth := map[string]any{"type": "authenticator", "secret": authenticatorSecret}
		if strings.TrimSpace(authenticatorOTP) != "" {
			auth["otp"] = authenticatorOTP
		}
		credentials = append(credentials, auth)
	}
	if len(customFields) > 0 {
		parsed, err := parseKV(customFields)
		if err != nil {
			return nil, err
		}
		fields := make([]map[string]any, 0, len(parsed))
		for key, val := range parsed {
			fields = append(fields, map[string]any{"name": key, "value": val})
		}
		credentials = append(credentials, map[string]any{"type": "custom", "fields": fields})
	}
	return credentials, nil
}
