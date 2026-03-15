package cli

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newSessionCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage browser sessions",
		Aliases: []string{
			"ses",
		},
	}
	cmd.PersistentFlags().String("session-id", "", "Session ID to use when omitted from command args")
	cmd.PersistentFlags().Bool("no-cache", false, "Do not use cached latest session ID when session ID is omitted")

	cmd.AddCommand(newSessionCreateCommand(app))
	cmd.AddCommand(newSessionListCommand(app))
	cmd.AddCommand(newSessionGetCommand(app))
	cmd.AddCommand(newSessionEndCommand(app))
	cmd.AddCommand(newSessionEndAllCommand(app))
	cmd.AddCommand(newSessionPagesCommand(app))
	cmd.AddCommand(newSessionHistoryCommand(app))
	cmd.AddCommand(newSessionStatusAllCommand(app))
	cmd.AddCommand(newSessionDownloadsCommand(app))
	cmd.AddCommand(newSessionRecordingsCommand(app))
	cmd.AddCommand(newSessionRecordingFetchPrimaryCommand(app))
	cmd.AddCommand(newSessionScreenshotCommand(app))
	cmd.AddCommand(newSessionClickCommand(app))
	cmd.AddCommand(newSessionDoubleClickCommand(app))
	cmd.AddCommand(newSessionMouseDownCommand(app))
	cmd.AddCommand(newSessionMouseUpCommand(app))
	cmd.AddCommand(newSessionMoveCommand(app))
	cmd.AddCommand(newSessionDragDropCommand(app))
	cmd.AddCommand(newSessionScrollCommand(app))
	cmd.AddCommand(newSessionTypeCommand(app))
	cmd.AddCommand(newSessionShortcutCommand(app))
	cmd.AddCommand(newSessionClipboardCommand(app))
	cmd.AddCommand(newSessionCopyCommand(app))
	cmd.AddCommand(newSessionPasteCommand(app))
	cmd.AddCommand(newSessionGotoCommand(app))
	cmd.AddCommand(newSessionUploadCommand(app))
	cmd.AddCommand(newAgentRunCommand(app))

	return cmd
}

func newSessionCreateCommand(app *App) *cobra.Command {
	return newSessionCreateCommandWithUse(app, "create", "Create a browser session")
}

func newSessionCreateCommandWithUse(app *App, use, short string) *cobra.Command {
	var bodyPath string
	var initialURL string
	var tags []string
	var recording bool
	var proxyActive bool
	var proxyType string
	var proxyCountry string
	var proxyRegion string
	var proxyCity string
	var maxDuration int
	var idleTimeout int
	var headless bool
	var viewportWidth int
	var viewportHeight int
	var profileName string
	var profilePersist bool
	var identityIDs []string
	var integrationIDs []string
	var interactive bool

	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}

			var payload map[string]any
			if interactive {
				if app.Global.DryRun {
					return fmt.Errorf("--interactive is not supported with --dry-run")
				}
				if err := validateInteractiveSessionCreateCompatibility(cmd); err != nil {
					return err
				}
				payload, err = runInteractiveSessionCreate(cmd, app, resolved.Value)
				if err != nil {
					return err
				}
			} else {
				payload, err = parseBodyAsMap(bodyPath)
				if err != nil {
					return err
				}
				if payload == nil {
					payload = map[string]any{}
				}

				if cmd.Flags().Changed("initial-url") || cmd.Flags().Changed("tag") || cmd.Flags().Changed("recording") ||
					cmd.Flags().Changed("max-duration") || cmd.Flags().Changed("idle-timeout") ||
					cmd.Flags().Changed("proxy-active") || cmd.Flags().Changed("proxy-type") ||
					cmd.Flags().Changed("proxy-country-code") || cmd.Flags().Changed("proxy-region") || cmd.Flags().Changed("proxy-city") {
					session := ensureMap(payload, "session")
					if cmd.Flags().Changed("initial-url") {
						session["initial_url"] = initialURL
					}
					if cmd.Flags().Changed("tag") {
						session["tags"] = tags
					}
					if cmd.Flags().Changed("recording") {
						session["recording"] = map[string]any{"active": recording}
					}
					if cmd.Flags().Changed("max-duration") || cmd.Flags().Changed("idle-timeout") {
						timeout := ensureMap(session, "timeout")
						if cmd.Flags().Changed("max-duration") {
							timeout["max_duration"] = maxDuration
						}
						if cmd.Flags().Changed("idle-timeout") {
							timeout["idle_timeout"] = idleTimeout
						}
					}
					if cmd.Flags().Changed("proxy-active") || cmd.Flags().Changed("proxy-type") || cmd.Flags().Changed("proxy-country-code") || cmd.Flags().Changed("proxy-region") || cmd.Flags().Changed("proxy-city") {
						proxy := map[string]any{}
						if cmd.Flags().Changed("proxy-active") {
							proxy["active"] = proxyActive
						}
						if cmd.Flags().Changed("proxy-type") {
							proxy["type"] = proxyType
						}
						if cmd.Flags().Changed("proxy-country-code") {
							proxy["country_code"] = proxyCountry
						}
						if cmd.Flags().Changed("proxy-region") {
							proxy["region"] = proxyRegion
						}
						if cmd.Flags().Changed("proxy-city") {
							proxy["city"] = proxyCity
						}
						session["proxy"] = proxy
					}
				}

				if cmd.Flags().Changed("headless") || cmd.Flags().Changed("viewport-width") || cmd.Flags().Changed("viewport-height") || cmd.Flags().Changed("profile-name") || cmd.Flags().Changed("profile-persist") {
					browser := ensureMap(payload, "browser")
					if cmd.Flags().Changed("headless") {
						browser["headless"] = map[string]any{"active": headless}
					}
					if cmd.Flags().Changed("viewport-width") || cmd.Flags().Changed("viewport-height") {
						viewport := map[string]any{}
						if cmd.Flags().Changed("viewport-width") {
							viewport["width"] = viewportWidth
						}
						if cmd.Flags().Changed("viewport-height") {
							viewport["height"] = viewportHeight
						}
						browser["viewport"] = viewport
					}
					if cmd.Flags().Changed("profile-name") || cmd.Flags().Changed("profile-persist") {
						profile := map[string]any{}
						if cmd.Flags().Changed("profile-name") {
							profile["name"] = profileName
						}
						if cmd.Flags().Changed("profile-persist") {
							profile["persist"] = profilePersist
						}
						browser["profile"] = profile
					}
				}

				if cmd.Flags().Changed("identity-id") {
					identities := make([]map[string]any, 0, len(identityIDs))
					for _, id := range identityIDs {
						identities = append(identities, map[string]any{"id": id})
					}
					payload["identities"] = identities
				}
				if cmd.Flags().Changed("integration-id") {
					integrations := make([]map[string]any, 0, len(integrationIDs))
					for _, id := range integrationIDs {
						integrations = append(integrations, map[string]any{"id": id})
					}
					payload["integrations"] = integrations
				}
			}

			client := app.newAPIClient()
			result, err := createSessionWithProgress(cmd.Context(), app.Stderr, client, resolved.Value, payload, interactive)
			if err != nil {
				return app.printDryRunOrValue(result, err)
			}
			noCache, _ := cmd.Flags().GetBool("no-cache")
			if !noCache {
				cachedSessionID := extractSessionIDFromResponse(result)
				if cacheErr := app.cacheSessionID(cachedSessionID); cacheErr != nil {
					return cacheErr
				}
				if cachedSessionID != "" {
					_, _ = fmt.Fprintf(app.Stderr, "Session %s created and cached as latest session.\n", cachedSessionID)
				}
			}
			return app.printValue(result)
		},
	}

	cmd.Flags().StringVar(&bodyPath, "body", "", "Path to JSON/YAML body file, '-' for stdin, or inline JSON")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "Run interactive wizard to build session create payload")
	cmd.Flags().StringVar(&initialURL, "initial-url", "", "Session initial URL")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "Session tags (repeatable)")
	cmd.Flags().BoolVar(&recording, "recording", true, "Enable session recording")
	cmd.Flags().BoolVar(&proxyActive, "proxy-active", false, "Enable proxy")
	cmd.Flags().StringVar(&proxyType, "proxy-type", "", "Proxy type (anchor_proxy|custom)")
	cmd.Flags().StringVar(&proxyCountry, "proxy-country-code", "", "Proxy country code")
	cmd.Flags().StringVar(&proxyRegion, "proxy-region", "", "Proxy region")
	cmd.Flags().StringVar(&proxyCity, "proxy-city", "", "Proxy city")
	cmd.Flags().IntVar(&maxDuration, "max-duration", 0, "Max session duration in minutes")
	cmd.Flags().IntVar(&idleTimeout, "idle-timeout", 0, "Idle timeout in minutes")
	cmd.Flags().BoolVar(&headless, "headless", false, "Run browser in headless mode")
	cmd.Flags().IntVar(&viewportWidth, "viewport-width", 0, "Browser viewport width")
	cmd.Flags().IntVar(&viewportHeight, "viewport-height", 0, "Browser viewport height")
	cmd.Flags().StringVar(&profileName, "profile-name", "", "Profile name")
	cmd.Flags().BoolVar(&profilePersist, "profile-persist", false, "Persist profile at session end")
	cmd.Flags().StringSliceVar(&identityIDs, "identity-id", nil, "Identity IDs to attach (repeatable)")
	cmd.Flags().StringSliceVar(&integrationIDs, "integration-id", nil, "Integration IDs to attach (repeatable)")
	return cmd
}

func createSessionWithProgress(
	ctx context.Context,
	out io.Writer,
	client interface {
		SessionCreate(ctx context.Context, apiKey string, body any) (any, error)
	},
	apiKey string,
	payload map[string]any,
	interactive bool,
) (any, error) {
	if !interactive {
		return client.SessionCreate(ctx, apiKey, payload)
	}

	message := "Creating a session"
	if payloadHasIdentity(payload) {
		message = "Creating an authenticated session"
	}
	if !isTerminalWriter(out) {
		_, _ = fmt.Fprintf(out, "%s...\n", message)
		return client.SessionCreate(ctx, apiKey, payload)
	}

	type createResult struct {
		value any
		err   error
	}

	results := make(chan createResult, 1)
	go func() {
		value, err := client.SessionCreate(ctx, apiKey, payload)
		results <- createResult{value: value, err: err}
	}()

	dotCount := 0
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	render := func() {
		_, _ = fmt.Fprintf(out, "\r%s%s", message, strings.Repeat(".", dotCount))
	}
	render()

	for {
		select {
		case result := <-results:
			if result.err == nil {
				_, _ = fmt.Fprintf(out, "\r%s... done\n", message)
			} else {
				_, _ = fmt.Fprintf(out, "\r%s...\n", message)
			}
			return result.value, result.err
		case <-ticker.C:
			dotCount = (dotCount + 1) % 4
			render()
		case <-ctx.Done():
			_, _ = fmt.Fprintln(out)
			return nil, ctx.Err()
		}
	}
}

func payloadHasIdentity(payload map[string]any) bool {
	raw := payload["identities"]
	switch values := raw.(type) {
	case []map[string]any:
		return len(values) > 0
	case []any:
		return len(values) > 0
	default:
		return false
	}
}

func newSessionListCommand(app *App) *cobra.Command {
	var page, limit int
	var sortBy, sortOrder, search, status, tags, domains, createdFrom, createdTo, batchID, profileName string
	var taskInitiated, playground, proxy, extraStealth bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sessions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			query := url.Values{}
			if cmd.Flags().Changed("page") {
				query.Set("page", fmt.Sprintf("%d", page))
			}
			if cmd.Flags().Changed("limit") {
				query.Set("limit", fmt.Sprintf("%d", limit))
			}
			if cmd.Flags().Changed("sort-by") {
				query.Set("sort_by", sortBy)
			}
			if cmd.Flags().Changed("sort-order") {
				query.Set("sort_order", sortOrder)
			}
			if cmd.Flags().Changed("search") {
				query.Set("search", search)
			}
			if cmd.Flags().Changed("status") {
				query.Set("status", status)
			}
			if cmd.Flags().Changed("tags") {
				query.Set("tags", tags)
			}
			if cmd.Flags().Changed("domains") {
				query.Set("domains", domains)
			}
			if cmd.Flags().Changed("created-from") {
				query.Set("created_from", createdFrom)
			}
			if cmd.Flags().Changed("created-to") {
				query.Set("created_to", createdTo)
			}
			if cmd.Flags().Changed("batch-id") {
				query.Set("batch_id", batchID)
			}
			if cmd.Flags().Changed("task-initiated") {
				query.Set("task_initiated", fmt.Sprintf("%t", taskInitiated))
			}
			if cmd.Flags().Changed("playground") {
				query.Set("playground", fmt.Sprintf("%t", playground))
			}
			if cmd.Flags().Changed("proxy") {
				query.Set("proxy", fmt.Sprintf("%t", proxy))
			}
			if cmd.Flags().Changed("extra-stealth") {
				query.Set("extra_stealth", fmt.Sprintf("%t", extraStealth))
			}
			if cmd.Flags().Changed("profile-name") {
				query.Set("profile_name", profileName)
			}

			client := app.newAPIClient()
			result, err := client.SessionList(cmd.Context(), resolved.Value, query)
			return app.printDryRunOrValue(result, err)
		},
	}

	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 10, "Page size (10,20,50)")
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Sort field")
	cmd.Flags().StringVar(&sortOrder, "sort-order", "", "Sort direction (asc|desc)")
	cmd.Flags().StringVar(&search, "search", "", "Tag search terms")
	cmd.Flags().StringVar(&status, "status", "", "Session status filter")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tag list")
	cmd.Flags().StringVar(&domains, "domains", "", "Comma-separated domain list")
	cmd.Flags().StringVar(&createdFrom, "created-from", "", "Created from timestamp (ISO-8601)")
	cmd.Flags().StringVar(&createdTo, "created-to", "", "Created to timestamp (ISO-8601)")
	cmd.Flags().StringVar(&batchID, "batch-id", "", "Batch ID")
	cmd.Flags().BoolVar(&taskInitiated, "task-initiated", false, "Filter task-initiated sessions")
	cmd.Flags().BoolVar(&playground, "playground", false, "Filter playground sessions")
	cmd.Flags().BoolVar(&proxy, "proxy", false, "Filter proxied sessions")
	cmd.Flags().BoolVar(&extraStealth, "extra-stealth", false, "Filter extra stealth sessions")
	cmd.Flags().StringVar(&profileName, "profile-name", "", "Filter by profile name")

	return cmd
}

func newSessionGetCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Get a session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			client := app.newAPIClient()
			result, err := client.SessionGet(cmd.Context(), resolved.Value, sessionID)
			return app.printDryRunOrValue(result, err)
		},
	}
}

func newSessionEndCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "end",
		Short: "End a session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			client := app.newAPIClient()
			result, err := client.SessionEnd(cmd.Context(), resolved.Value, sessionID)
			return app.printDryRunOrValue(result, err)
		},
	}
}

func newSessionEndAllCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "end-all",
		Short: "End all active sessions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			client := app.newAPIClient()
			result, err := client.SessionEndAll(cmd.Context(), resolved.Value)
			return app.printDryRunOrValue(result, err)
		},
	}
}

func newSessionPagesCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "pages",
		Short: "List open pages for a session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			client := app.newAPIClient()
			result, err := client.SessionPages(cmd.Context(), resolved.Value, sessionID)
			return app.printDryRunOrValue(result, err)
		},
	}
}

func newSessionHistoryCommand(app *App) *cobra.Command {
	var fromDate, toDate string
	var metrics []string
	var page, limit int

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Get session history metrics",
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			query := url.Values{}
			if cmd.Flags().Changed("from-date") {
				query.Set("from_date", fromDate)
			}
			if cmd.Flags().Changed("to-date") {
				query.Set("to_date", toDate)
			}
			if cmd.Flags().Changed("metrics") {
				for _, m := range metrics {
					query.Add("metrics", m)
				}
			}
			if cmd.Flags().Changed("page") {
				query.Set("page", fmt.Sprintf("%d", page))
			}
			if cmd.Flags().Changed("limit") {
				query.Set("limit", fmt.Sprintf("%d", limit))
			}

			client := app.newAPIClient()
			result, err := client.SessionHistory(cmd.Context(), resolved.Value, query)
			return app.printDryRunOrValue(result, err)
		},
	}
	cmd.Flags().StringVar(&fromDate, "from-date", "", "Filter from date (ISO-8601)")
	cmd.Flags().StringVar(&toDate, "to-date", "", "Filter to date (ISO-8601)")
	cmd.Flags().StringSliceVar(&metrics, "metrics", nil, "Metrics to include")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 100, "Records per page")
	return cmd
}

func newSessionStatusAllCommand(app *App) *cobra.Command {
	var tags, domains, createdFrom, createdTo, batchID, profileName string
	var taskInitiated, playground, proxy, extraStealth bool

	cmd := &cobra.Command{
		Use:   "status-all",
		Short: "Get statuses for all sessions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			query := url.Values{}
			if cmd.Flags().Changed("tags") {
				query.Set("tags", tags)
			}
			if cmd.Flags().Changed("domains") {
				query.Set("domains", domains)
			}
			if cmd.Flags().Changed("created-from") {
				query.Set("created_from", createdFrom)
			}
			if cmd.Flags().Changed("created-to") {
				query.Set("created_to", createdTo)
			}
			if cmd.Flags().Changed("batch-id") {
				query.Set("batch_id", batchID)
			}
			if cmd.Flags().Changed("task-initiated") {
				query.Set("task_initiated", fmt.Sprintf("%t", taskInitiated))
			}
			if cmd.Flags().Changed("playground") {
				query.Set("playground", fmt.Sprintf("%t", playground))
			}
			if cmd.Flags().Changed("proxy") {
				query.Set("proxy", fmt.Sprintf("%t", proxy))
			}
			if cmd.Flags().Changed("extra-stealth") {
				query.Set("extra_stealth", fmt.Sprintf("%t", extraStealth))
			}
			if cmd.Flags().Changed("profile-name") {
				query.Set("profile_name", profileName)
			}

			client := app.newAPIClient()
			result, err := client.SessionStatusAll(cmd.Context(), resolved.Value, query)
			return app.printDryRunOrValue(result, err)
		},
	}
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tag list")
	cmd.Flags().StringVar(&domains, "domains", "", "Comma-separated domain list")
	cmd.Flags().StringVar(&createdFrom, "created-from", "", "Created from timestamp (ISO-8601)")
	cmd.Flags().StringVar(&createdTo, "created-to", "", "Created to timestamp (ISO-8601)")
	cmd.Flags().StringVar(&batchID, "batch-id", "", "Batch ID")
	cmd.Flags().BoolVar(&taskInitiated, "task-initiated", false, "Filter task-initiated sessions")
	cmd.Flags().BoolVar(&playground, "playground", false, "Filter playground sessions")
	cmd.Flags().BoolVar(&proxy, "proxy", false, "Filter proxied sessions")
	cmd.Flags().BoolVar(&extraStealth, "extra-stealth", false, "Filter extra stealth sessions")
	cmd.Flags().StringVar(&profileName, "profile-name", "", "Filter by profile name")
	return cmd
}

func newSessionDownloadsCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "downloads",
		Short: "List files downloaded during a session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			client := app.newAPIClient()
			result, err := client.SessionDownloads(cmd.Context(), resolved.Value, sessionID)
			return app.printDryRunOrValue(result, err)
		},
	}
}

func newSessionRecordingsCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "recordings",
		Short: "List session recordings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			client := app.newAPIClient()
			result, err := client.SessionRecordings(cmd.Context(), resolved.Value, sessionID)
			return app.printDryRunOrValue(result, err)
		},
	}
}

func newSessionRecordingFetchPrimaryCommand(app *App) *cobra.Command {
	var outPath string
	cmd := &cobra.Command{
		Use:   "recording",
		Short: "Recording operations",
	}

	fetchCmd := &cobra.Command{
		Use:   "fetch-primary",
		Short: "Download primary recording file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			client := app.newAPIClient()
			data, err := client.SessionRecordingFetchPrimary(cmd.Context(), resolved.Value, sessionID)
			if err != nil {
				if app.Global.DryRun {
					return nil
				}
				return err
			}
			return writeBinary(outPath, data)
		},
	}
	fetchCmd.Flags().StringVar(&outPath, "out", "-", "Output file path, '-' for stdout")
	cmd.AddCommand(fetchCmd)
	return cmd
}

func newSessionScreenshotCommand(app *App) *cobra.Command {
	var outPath string
	cmd := &cobra.Command{
		Use:   "screenshot",
		Short: "Capture session screenshot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			client := app.newAPIClient()
			data, err := client.SessionScreenshot(cmd.Context(), resolved.Value, sessionID)
			if err != nil {
				if app.Global.DryRun {
					return nil
				}
				return err
			}
			return writeBinary(outPath, data)
		},
	}
	cmd.Flags().StringVar(&outPath, "out", "-", "Output file path, '-' for stdout")
	return cmd
}

func newSessionClickCommand(app *App) *cobra.Command {
	var x, y float64
	var selector, button string
	var timeout, index int
	cmd := &cobra.Command{
		Use:   "click",
		Short: "Click in session (coordinates or selector)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			selectorSet := cmd.Flags().Changed("selector")
			xySet := cmd.Flags().Changed("x") || cmd.Flags().Changed("y")
			if selectorSet == xySet {
				return fmt.Errorf("provide either --selector or --x/--y")
			}
			if xySet && !(cmd.Flags().Changed("x") && cmd.Flags().Changed("y")) {
				return fmt.Errorf("both --x and --y are required")
			}
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			body := map[string]any{}
			if xySet {
				body["x"] = x
				body["y"] = y
			}
			if selectorSet {
				body["selector"] = selector
			}
			if cmd.Flags().Changed("button") {
				body["button"] = button
			}
			if cmd.Flags().Changed("timeout") {
				body["timeout"] = timeout
			}
			if cmd.Flags().Changed("index") {
				body["index"] = index
			}
			client := app.newAPIClient()
			result, err := client.SessionClick(cmd.Context(), resolved.Value, sessionID, body)
			return app.printDryRunOrValue(result, err)
		},
	}
	cmd.Flags().Float64Var(&x, "x", 0, "X coordinate")
	cmd.Flags().Float64Var(&y, "y", 0, "Y coordinate")
	cmd.Flags().StringVar(&selector, "selector", "", "CSS selector")
	cmd.Flags().StringVar(&button, "button", "left", "Mouse button: left|middle|right")
	cmd.Flags().IntVar(&timeout, "timeout", 5000, "Selector wait timeout in ms")
	cmd.Flags().IntVar(&index, "index", 0, "Element index when selector matches multiple elements")
	return cmd
}

func newSessionDoubleClickCommand(app *App) *cobra.Command {
	return newPointWithButtonCommand(app, "double-click", "Double-click at coordinates", func(cmd *cobra.Command, key, sessionID string, body map[string]any) (any, error) {
		return app.newAPIClient().SessionDoubleClick(cmd.Context(), key, sessionID, body)
	})
}

func newSessionMouseDownCommand(app *App) *cobra.Command {
	return newPointWithButtonCommand(app, "mouse-down", "Mouse down at coordinates", func(cmd *cobra.Command, key, sessionID string, body map[string]any) (any, error) {
		return app.newAPIClient().SessionMouseDown(cmd.Context(), key, sessionID, body)
	})
}

func newSessionMouseUpCommand(app *App) *cobra.Command {
	return newPointWithButtonCommand(app, "mouse-up", "Mouse up at coordinates", func(cmd *cobra.Command, key, sessionID string, body map[string]any) (any, error) {
		return app.newAPIClient().SessionMouseUp(cmd.Context(), key, sessionID, body)
	})
}

func newPointWithButtonCommand(app *App, use, short string, call func(cmd *cobra.Command, key, sessionID string, body map[string]any) (any, error)) *cobra.Command {
	var x, y int
	var button string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !cmd.Flags().Changed("x") || !cmd.Flags().Changed("y") {
				return fmt.Errorf("both --x and --y are required")
			}
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			body := map[string]any{"x": x, "y": y}
			if cmd.Flags().Changed("button") {
				body["button"] = button
			}
			result, err := call(cmd, resolved.Value, sessionID, body)
			return app.printDryRunOrValue(result, err)
		},
	}
	cmd.Flags().IntVar(&x, "x", 0, "X coordinate")
	cmd.Flags().IntVar(&y, "y", 0, "Y coordinate")
	cmd.Flags().StringVar(&button, "button", "left", "Mouse button")
	return cmd
}

func newSessionMoveCommand(app *App) *cobra.Command {
	var x, y int
	cmd := &cobra.Command{
		Use:   "move",
		Short: "Move mouse cursor",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !cmd.Flags().Changed("x") || !cmd.Flags().Changed("y") {
				return fmt.Errorf("both --x and --y are required")
			}
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			result, err := app.newAPIClient().SessionMove(cmd.Context(), resolved.Value, sessionID, map[string]any{"x": x, "y": y})
			return app.printDryRunOrValue(result, err)
		},
	}
	cmd.Flags().IntVar(&x, "x", 0, "X coordinate")
	cmd.Flags().IntVar(&y, "y", 0, "Y coordinate")
	return cmd
}

func newSessionDragDropCommand(app *App) *cobra.Command {
	var startX, startY, endX, endY int
	var button string
	cmd := &cobra.Command{
		Use:   "drag-drop",
		Short: "Drag and drop from start to end coordinates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !cmd.Flags().Changed("start-x") || !cmd.Flags().Changed("start-y") || !cmd.Flags().Changed("end-x") || !cmd.Flags().Changed("end-y") {
				return fmt.Errorf("--start-x, --start-y, --end-x, --end-y are required")
			}
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			body := map[string]any{"startX": startX, "startY": startY, "endX": endX, "endY": endY}
			if cmd.Flags().Changed("button") {
				body["button"] = button
			}
			result, err := app.newAPIClient().SessionDragDrop(cmd.Context(), resolved.Value, sessionID, body)
			return app.printDryRunOrValue(result, err)
		},
	}
	cmd.Flags().IntVar(&startX, "start-x", 0, "Start X coordinate")
	cmd.Flags().IntVar(&startY, "start-y", 0, "Start Y coordinate")
	cmd.Flags().IntVar(&endX, "end-x", 0, "End X coordinate")
	cmd.Flags().IntVar(&endY, "end-y", 0, "End Y coordinate")
	cmd.Flags().StringVar(&button, "button", "left", "Mouse button")
	return cmd
}

func newSessionScrollCommand(app *App) *cobra.Command {
	var x, y, deltaX, deltaY, steps int
	var useOS bool
	cmd := &cobra.Command{
		Use:   "scroll",
		Short: "Scroll in session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !cmd.Flags().Changed("x") || !cmd.Flags().Changed("y") || !cmd.Flags().Changed("delta-y") {
				return fmt.Errorf("--x, --y and --delta-y are required")
			}
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			body := map[string]any{"x": x, "y": y, "deltaY": deltaY}
			if cmd.Flags().Changed("delta-x") {
				body["deltaX"] = deltaX
			}
			if cmd.Flags().Changed("steps") {
				body["steps"] = steps
			}
			if cmd.Flags().Changed("use-os") {
				body["useOs"] = useOS
			}
			result, err := app.newAPIClient().SessionScroll(cmd.Context(), resolved.Value, sessionID, body)
			return app.printDryRunOrValue(result, err)
		},
	}
	cmd.Flags().IntVar(&x, "x", 0, "X coordinate")
	cmd.Flags().IntVar(&y, "y", 0, "Y coordinate")
	cmd.Flags().IntVar(&deltaX, "delta-x", 0, "Horizontal delta")
	cmd.Flags().IntVar(&deltaY, "delta-y", 0, "Vertical delta")
	cmd.Flags().IntVar(&steps, "steps", 0, "Scroll steps")
	cmd.Flags().BoolVar(&useOS, "use-os", false, "Use OS-level scrolling")
	return cmd
}

func newSessionTypeCommand(app *App) *cobra.Command {
	var text string
	var delay int
	cmd := &cobra.Command{
		Use:   "type",
		Short: "Type text",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !cmd.Flags().Changed("text") {
				return fmt.Errorf("--text is required")
			}
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			body := map[string]any{"text": text}
			if cmd.Flags().Changed("delay") {
				body["delay"] = delay
			}
			result, err := app.newAPIClient().SessionType(cmd.Context(), resolved.Value, sessionID, body)
			return app.printDryRunOrValue(result, err)
		},
	}
	cmd.Flags().StringVar(&text, "text", "", "Text to type")
	cmd.Flags().IntVar(&delay, "delay", 0, "Delay between keystrokes (ms)")
	return cmd
}

func newSessionShortcutCommand(app *App) *cobra.Command {
	var keys []string
	var holdTime int
	cmd := &cobra.Command{
		Use:   "shortcut",
		Short: "Press keyboard shortcut",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(keys) == 0 {
				return fmt.Errorf("--keys is required")
			}
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			body := map[string]any{"keys": keys}
			if cmd.Flags().Changed("hold-time") {
				body["holdTime"] = holdTime
			}
			result, err := app.newAPIClient().SessionShortcut(cmd.Context(), resolved.Value, sessionID, body)
			return app.printDryRunOrValue(result, err)
		},
	}
	cmd.Flags().StringSliceVar(&keys, "keys", nil, "Keys to press simultaneously (comma-separated or repeated)")
	cmd.Flags().IntVar(&holdTime, "hold-time", 0, "Hold time in ms")
	return cmd
}

func newSessionClipboardCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{Use: "clipboard", Short: "Clipboard operations"}

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get clipboard contents",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			result, err := app.newAPIClient().SessionClipboardGet(cmd.Context(), resolved.Value, sessionID)
			return app.printDryRunOrValue(result, err)
		},
	}

	var text string
	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Set clipboard text",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !cmd.Flags().Changed("text") {
				return fmt.Errorf("--text is required")
			}
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			result, err := app.newAPIClient().SessionClipboardSet(cmd.Context(), resolved.Value, sessionID, map[string]any{"text": text})
			return app.printDryRunOrValue(result, err)
		},
	}
	setCmd.Flags().StringVar(&text, "text", "", "Clipboard text")

	cmd.AddCommand(getCmd, setCmd)
	return cmd
}

func newSessionCopyCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "copy",
		Short: "Copy selected text to clipboard",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			result, err := app.newAPIClient().SessionCopy(cmd.Context(), resolved.Value, sessionID)
			return app.printDryRunOrValue(result, err)
		},
	}
}

func newSessionPasteCommand(app *App) *cobra.Command {
	var text string
	cmd := &cobra.Command{
		Use:   "paste",
		Short: "Paste text at cursor",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !cmd.Flags().Changed("text") {
				return fmt.Errorf("--text is required")
			}
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			result, err := app.newAPIClient().SessionPaste(cmd.Context(), resolved.Value, sessionID, map[string]any{"text": text})
			return app.printDryRunOrValue(result, err)
		},
	}
	cmd.Flags().StringVar(&text, "text", "", "Text to paste")
	return cmd
}

func newSessionGotoCommand(app *App) *cobra.Command {
	var targetURL string
	cmd := &cobra.Command{
		Use:   "goto [url]",
		Short: "Navigate to URL",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var finalURL string
			argURL := ""
			if len(args) == 1 {
				argURL = strings.TrimSpace(args[0])
			}
			flagURL := strings.TrimSpace(targetURL)
			if argURL != "" && flagURL != "" && argURL != flagURL {
				return fmt.Errorf("url specified twice with different values (arg=%q, --url=%q)", argURL, flagURL)
			}
			if argURL != "" {
				finalURL = argURL
			} else {
				finalURL = flagURL
			}
			if finalURL == "" {
				return fmt.Errorf("url is required (pass as `session goto <url>` or `--url`)")
			}
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			result, err := app.newAPIClient().SessionGoto(cmd.Context(), resolved.Value, sessionID, map[string]any{"url": finalURL})
			return app.printDryRunOrValue(result, err)
		},
	}
	cmd.Flags().StringVar(&targetURL, "url", "", "Destination URL")
	return cmd
}

func newSessionUploadCommand(app *App) *cobra.Command {
	var filePath string
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload a file to session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !cmd.Flags().Changed("file") {
				return fmt.Errorf("--file is required")
			}
			sessionID, err := app.resolveSessionID(cmd)
			if err != nil {
				return err
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			result, err := app.newAPIClient().SessionUpload(cmd.Context(), resolved.Value, sessionID, filePath)
			return app.printDryRunOrValue(result, err)
		},
	}
	cmd.Flags().StringVar(&filePath, "file", "", "File path to upload")
	return cmd
}
