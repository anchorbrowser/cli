package cli

import (
	"context"
	"fmt"
	neturl "net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anchorbrowser/cli/internal/api"
)

func newIdentityCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{Use: "identity", Short: "Manage identities"}
	cmd.AddCommand(newIdentityCreateCommand(app))
	cmd.AddCommand(newIdentityListCommand(app))
	cmd.AddCommand(newIdentityGetCommand(app))
	cmd.AddCommand(newIdentityUpdateCommand(app))
	cmd.AddCommand(newIdentityDeleteCommand(app))
	cmd.AddCommand(newIdentityCredentialsCommand(app))
	return cmd
}

func newIdentityListCommand(app *App) *cobra.Command {
	var applicationURL string
	var search string
	var page, limit int

	cmd := &cobra.Command{
		Use:   "list [application-url]",
		Short: "List identities for an application URL",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			rawURL := strings.TrimSpace(applicationURL)
			if len(args) > 0 {
				if rawURL != "" && rawURL != args[0] {
					return fmt.Errorf("application URL conflict: positional %q vs --application-url %q", args[0], rawURL)
				}
				rawURL = strings.TrimSpace(args[0])
			}
			if rawURL == "" {
				return fmt.Errorf("application URL is required (pass as positional argument or --application-url)")
			}

			client := app.newAPIClient()
			appID, applicationObj, err := resolveApplicationByURL(cmd.Context(), client, resolved.Value, rawURL)
			if err != nil {
				return err
			}

			query := make(neturl.Values)
			if strings.TrimSpace(search) != "" {
				query.Set("search", search)
			}
			if page > 0 {
				query.Set("page", fmt.Sprintf("%d", page))
			}
			if limit > 0 {
				query.Set("limit", fmt.Sprintf("%d", limit))
			}

			result, err := client.ApplicationListIdentities(cmd.Context(), resolved.Value, appID, query)
			if err != nil {
				return app.printDryRunOrValue(result, err)
			}

			out := map[string]any{
				"application": applicationObj,
			}
			if m, ok := result.(map[string]any); ok {
				for k, v := range m {
					out[k] = v
				}
			} else {
				out["result"] = result
			}
			return app.printValue(out)
		},
	}

	cmd.Flags().StringVar(&applicationURL, "application-url", "", "Application URL used to resolve the application")
	cmd.Flags().StringVar(&search, "search", "", "Search identities by name")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 50, "Page size")

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

func resolveApplicationByURL(ctx context.Context, client *api.Client, apiKey, rawURL string) (string, map[string]any, error) {
	normalizedInput, inputHost := normalizeURLForMatch(rawURL)
	query := make(neturl.Values)
	if inputHost != "" {
		query.Set("search", inputHost)
	}

	raw, err := client.ApplicationList(ctx, apiKey, query)
	if err != nil {
		return "", nil, err
	}
	root, ok := raw.(map[string]any)
	if !ok {
		return "", nil, fmt.Errorf("unexpected application list response")
	}
	items, ok := root["applications"].([]any)
	if !ok || len(items) == 0 {
		return "", nil, fmt.Errorf("no applications found for %q", rawURL)
	}

	bestScore := -1
	var best map[string]any
	var bestID string
	tied := 0

	for _, item := range items {
		appObj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		appID := firstString(appObj["id"], appObj["applicationId"])
		if appID == "" {
			continue
		}
		source := firstString(appObj["source"], appObj["url"], appObj["applicationUrl"], appObj["loginUrl"])
		score := scoreApplicationURLMatch(normalizedInput, inputHost, source)
		if score > bestScore {
			bestScore = score
			best = appObj
			bestID = appID
			tied = 1
			continue
		}
		if score == bestScore && score >= 2 {
			tied++
		}
	}

	if bestID == "" || bestScore <= 0 {
		return "", nil, fmt.Errorf("could not resolve application for URL %q", rawURL)
	}
	if tied > 1 && bestScore >= 2 {
		return "", nil, fmt.Errorf("multiple applications matched %q; please refine URL", rawURL)
	}
	return bestID, best, nil
}

func scoreApplicationURLMatch(normalizedInput, inputHost, candidate string) int {
	if strings.TrimSpace(candidate) == "" {
		return 0
	}
	normalizedCandidate, candidateHost := normalizeURLForMatch(candidate)
	if normalizedInput != "" && normalizedCandidate == normalizedInput {
		return 3
	}
	if inputHost != "" && candidateHost != "" && equivalentHost(inputHost, candidateHost) {
		return 2
	}
	if normalizedInput != "" && strings.Contains(normalizedCandidate, normalizedInput) {
		return 1
	}
	return 0
}

func normalizeURLForMatch(raw string) (string, string) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", ""
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	u, err := neturl.Parse(value)
	if err != nil {
		return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(raw), "/")), ""
	}
	host := strings.ToLower(u.Hostname())
	path := strings.TrimSuffix(u.EscapedPath(), "/")
	if path == "" {
		path = "/"
	}
	return host + path, host
}

func equivalentHost(a, b string) bool {
	trim := func(s string) string {
		return strings.TrimPrefix(strings.ToLower(s), "www.")
	}
	return trim(a) == trim(b)
}

func firstString(vals ...any) string {
	for _, v := range vals {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
