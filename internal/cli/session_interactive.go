package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	neturl "net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/anchorbrowser/cli/internal/api"
)

var uuidPattern = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[1-5][a-fA-F0-9]{3}-[89abAB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`)

const interactiveSummaryPayloadKey = "__interactive_completed_steps"

func validateInteractiveSessionCreateCompatibility(cmd *cobra.Command) error {
	incompatible := []string{
		"body",
		"initial-url",
		"tag",
		"recording",
		"proxy-active",
		"proxy-type",
		"proxy-country-code",
		"proxy-region",
		"proxy-city",
		"max-duration",
		"idle-timeout",
		"headless",
		"viewport-width",
		"viewport-height",
		"profile-name",
		"profile-persist",
		"identity-id",
		"integration-id",
	}
	var changed []string
	for _, flagName := range incompatible {
		if cmd.Flags().Changed(flagName) {
			changed = append(changed, "--"+flagName)
		}
	}
	if len(changed) > 0 {
		return fmt.Errorf("--interactive cannot be combined with create payload flags: %s", strings.Join(changed, ", "))
	}
	return nil
}

func runInteractiveSessionCreate(cmd *cobra.Command, app *App, apiKey string) (map[string]any, error) {
	if shouldUseInteractiveTUI(app) {
		return runInteractiveSessionCreateTUI(cmd, app, apiKey)
	}
	if err := printBanner(app.Stderr); err != nil {
		return nil, err
	}
	return runInteractiveSessionCreatePlain(cmd, app, apiKey)
}

func runInteractiveSessionCreatePlain(cmd *cobra.Command, app *App, apiKey string) (map[string]any, error) {
	reader := bufio.NewReader(app.Stdin)
	client := app.newAPIClient()
	payload := map[string]any{}
	completed := []string{}

	authenticated, err := promptYesNo(reader, app.Stderr, "Do you need this session to be authenticated? [Y/n]: ", true)
	if err != nil {
		return nil, err
	}
	completed = append(completed, fmt.Sprintf("Authenticated session: %s", yesNoLabel(authenticated)))

	if authenticated {
		applicationURL, err := promptRequired(reader, app.Stderr, "Application URL: ")
		if err != nil {
			return nil, err
		}
		completed = append(completed, fmt.Sprintf("Application URL: %s", normalizeApplicationSourceURL(applicationURL)))
		appID, appObj, err := resolveApplicationForInteractive(cmd.Context(), client, apiKey, applicationURL)
		if err != nil {
			return nil, err
		}

		identityID, err := interactiveSelectOrCreateIdentity(cmd.Context(), reader, app.Stderr, client, apiKey, appID, appObj, applicationURL)
		if err != nil {
			return nil, err
		}
		payload["identities"] = []map[string]any{{"id": identityID}}
		completed = append(completed, fmt.Sprintf("Identity attached: %s", identityID))
	}

	useRecommended, err := promptYesNo(reader, app.Stderr, "Use recommended anti-bot settings (stealth + captcha solver + proxy)? [Y/n]: ", true)
	if err != nil {
		return nil, err
	}
	if useRecommended {
		applyRecommendedAntiBotPayload(payload)
	}
	completed = append(completed, fmt.Sprintf("Recommended anti-bot bundle: %s", yesNoLabel(useRecommended)))
	attachInteractiveSummary(payload, completed)

	return payload, nil
}

func attachInteractiveSummary(payload map[string]any, completed []string) {
	if len(completed) == 0 {
		return
	}
	snapshot := append([]string(nil), completed...)
	payload[interactiveSummaryPayloadKey] = snapshot
}

func shouldUseInteractiveTUI(app *App) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ANCHORBROWSER_INTERACTIVE_UI"))) {
	case "plain", "fallback":
		return false
	case "tui":
		return true
	}
	return isTerminalReader(app.Stdin) && isTerminalWriter(app.Stderr)
}

func isTerminalReader(r io.Reader) bool {
	f, ok := r.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

func isTerminalWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

func resolveApplicationForInteractive(ctx context.Context, client *api.Client, apiKey, rawURL string) (string, map[string]any, error) {
	appID, appObj, err := resolveApplicationByURL(ctx, client, apiKey, rawURL)
	if err == nil {
		return appID, appObj, nil
	}
	if !strings.Contains(strings.ToLower(err.Error()), "no applications found") {
		return "", nil, err
	}

	source := normalizeApplicationSourceURL(rawURL)
	createBody := map[string]any{
		"source": source,
		"name":   deriveApplicationName(source),
	}
	created, createErr := client.ApplicationCreate(ctx, apiKey, createBody)
	if createErr != nil {
		return "", nil, fmt.Errorf("resolve application failed (%v), and auto-create failed: %w", err, createErr)
	}

	root, ok := created.(map[string]any)
	if !ok {
		return "", nil, fmt.Errorf("unexpected application create response")
	}
	createdID := firstString(root["id"], mapGet(root, "data", "id"))
	if createdID == "" {
		return "", nil, fmt.Errorf("application created but id missing in response")
	}
	return createdID, root, nil
}

func interactiveSelectOrCreateIdentity(
	ctx context.Context,
	reader *bufio.Reader,
	out io.Writer,
	client *api.Client,
	apiKey, applicationID string,
	application map[string]any,
	applicationURL string,
) (string, error) {
	search := ""
	for {
		query := make(neturl.Values)
		if strings.TrimSpace(search) != "" {
			query.Set("search", search)
		}

		listResult, err := client.ApplicationListIdentities(ctx, apiKey, applicationID, query)
		if err != nil {
			return "", err
		}

		identities := extractIdentityRows(listResult)
		if len(identities) > 0 {
			_, _ = fmt.Fprintln(out, "Available identities:")
			for i, ident := range identities {
				_, _ = fmt.Fprintf(out, "  %d) %s\n", i+1, firstString(ident["name"], ident["id"]))
			}
			_, _ = fmt.Fprintln(out, "  c) Create new identity")
			_, _ = fmt.Fprintln(out, "  s) Search identities")
			choice, promptErr := promptText(reader, out, "Select identity: ")
			if promptErr != nil {
				return "", promptErr
			}
			choice = strings.TrimSpace(strings.ToLower(choice))
			switch choice {
			case "c":
				return interactiveCreateIdentity(ctx, reader, out, client, apiKey, applicationID, application, applicationURL)
			case "s":
				search, promptErr = promptText(reader, out, "Search query: ")
				if promptErr != nil {
					return "", promptErr
				}
				continue
			default:
				n, convErr := strconv.Atoi(choice)
				if convErr != nil || n < 1 || n > len(identities) {
					_, _ = fmt.Fprintln(out, "Invalid selection.")
					continue
				}
				selectedID := firstString(identities[n-1]["id"])
				if selectedID == "" {
					return "", fmt.Errorf("selected identity missing id")
				}
				return selectedID, nil
			}
		}

		_, _ = fmt.Fprintln(out, "No identities found.")
		_, _ = fmt.Fprintln(out, "  c) Create new identity")
		_, _ = fmt.Fprintln(out, "  s) Search again")
		choice, promptErr := promptText(reader, out, "Select option: ")
		if promptErr != nil {
			return "", promptErr
		}
		switch strings.TrimSpace(strings.ToLower(choice)) {
		case "c":
			return interactiveCreateIdentity(ctx, reader, out, client, apiKey, applicationID, application, applicationURL)
		case "s":
			search, promptErr = promptText(reader, out, "Search query: ")
			if promptErr != nil {
				return "", promptErr
			}
		default:
			_, _ = fmt.Fprintln(out, "Invalid selection.")
		}
	}
}

type interactiveAuthFlow struct {
	ID           string
	Name         string
	Methods      []string
	CustomFields []string
	Recommended  bool
}

func interactiveCreateIdentity(
	ctx context.Context,
	reader *bufio.Reader,
	out io.Writer,
	client *api.Client,
	apiKey, applicationID string,
	application map[string]any,
	applicationURL string,
) (string, error) {
	flowsResult, err := client.ApplicationListAuthFlows(ctx, apiKey, applicationID)
	if err != nil {
		return "", err
	}
	flows := extractAuthFlows(flowsResult)
	if len(flows) == 0 {
		_, _ = fmt.Fprintln(out, "No auth flows found for this application; falling back to manual identity creation.")
		return interactiveCreateIdentityManual(ctx, reader, out, client, apiKey, applicationID)
	}

	_, _ = fmt.Fprintln(out, "Authentication flows:")
	for i, flow := range flows {
		recommended := ""
		if flow.Recommended {
			recommended = " (recommended)"
		}
		_, _ = fmt.Fprintf(out, "  %d) %s%s\n", i+1, flow.Name, recommended)
	}

	var selected interactiveAuthFlow
	for {
		rawChoice, promptErr := promptText(reader, out, "Select auth flow: ")
		if promptErr != nil {
			return "", promptErr
		}
		n, convErr := strconv.Atoi(strings.TrimSpace(rawChoice))
		if convErr != nil || n < 1 || n > len(flows) {
			_, _ = fmt.Fprintln(out, "Invalid selection.")
			continue
		}
		selected = flows[n-1]
		break
	}

	if requiresManualIdentityFlow(selected.Methods) {
		_, _ = fmt.Fprintln(out, "Selected flow requires manual browser completion.")
		return interactiveCreateIdentityManual(ctx, reader, out, client, apiKey, applicationID)
	}

	credentials, identityName, err := promptCredentialsForFlow(reader, out, selected)
	if err != nil {
		return "", err
	}
	source := normalizeApplicationSourceURL(firstString(mapGet(application, "url"), applicationURL))
	createPayload := map[string]any{
		"source":      source,
		"credentials": credentials,
	}
	if identityName != "" {
		createPayload["name"] = identityName
	}
	if appName := firstString(mapGet(application, "name")); appName != "" {
		createPayload["applicationName"] = appName
	}

	createResult, err := client.IdentityCreate(ctx, apiKey, true, createPayload)
	if err != nil {
		return "", err
	}
	identityID := extractIdentityID(createResult)
	if identityID == "" {
		return "", fmt.Errorf("identity created but id was not returned")
	}
	return identityID, nil
}

func interactiveCreateIdentityManual(ctx context.Context, reader *bufio.Reader, out io.Writer, client *api.Client, apiKey, applicationID string) (string, error) {
	tokenResult, err := client.ApplicationCreateToken(ctx, apiKey, applicationID, map[string]any{})
	if err != nil {
		return "", err
	}
	token := firstString(mapGet(tokenResult, "token"), mapGet(tokenResult, "data", "token"))
	if token == "" {
		return "", fmt.Errorf("token response missing token")
	}

	userName, err := promptText(reader, out, "Display name for manual identity (optional): ")
	if err != nil {
		return "", err
	}

	params := neturl.Values{}
	params.Set("token", token)
	if strings.TrimSpace(userName) != "" {
		params.Set("userName", strings.TrimSpace(userName))
	}
	manualURL := "https://app.anchorbrowser.io/identity/create?" + params.Encode()
	_, _ = fmt.Fprintln(out, "Open this URL to complete manual identity setup:")
	_, _ = fmt.Fprintln(out, manualURL)
	if openErr := openBrowserURL(manualURL); openErr != nil {
		_, _ = fmt.Fprintf(out, "Could not open browser automatically: %v\n", openErr)
	}
	_, _ = promptText(reader, out, "Press Enter after you finish manual identity creation in the browser...")

	if strings.TrimSpace(userName) != "" {
		if identityID, lookupErr := findRecentIdentityByName(ctx, client, apiKey, applicationID, userName, 45*time.Minute); lookupErr == nil && identityID != "" {
			_, _ = fmt.Fprintf(out, "Found newly created identity by name: %s\n", identityID)
			return identityID, nil
		}
	}

	for {
		identityID, promptErr := promptRequired(reader, out, "Paste created identity ID: ")
		if promptErr != nil {
			return "", promptErr
		}
		if !uuidPattern.MatchString(identityID) {
			_, _ = fmt.Fprintln(out, "Identity ID must be a valid UUID.")
			continue
		}
		return identityID, nil
	}
}

func findRecentIdentityByName(ctx context.Context, client *api.Client, apiKey, applicationID, name string, window time.Duration) (string, error) {
	query := make(neturl.Values)
	query.Set("search", name)
	res, err := client.ApplicationListIdentities(ctx, apiKey, applicationID, query)
	if err != nil {
		return "", err
	}
	rows := extractIdentityRows(res)
	if len(rows) == 0 {
		return "", nil
	}

	needle := strings.ToLower(strings.TrimSpace(name))
	now := time.Now()
	bestID := ""
	var bestTime time.Time

	for _, row := range rows {
		rowName := strings.ToLower(strings.TrimSpace(firstString(row["name"])))
		if rowName == "" || !strings.Contains(rowName, needle) {
			continue
		}
		id := firstString(row["id"])
		if id == "" {
			continue
		}
		createdAtRaw := firstString(row["created_at"])
		createdAt, parseErr := time.Parse(time.RFC3339, createdAtRaw)
		if parseErr != nil {
			continue
		}
		if now.Sub(createdAt) > window {
			continue
		}
		if bestID == "" || createdAt.After(bestTime) {
			bestID = id
			bestTime = createdAt
		}
	}
	return bestID, nil
}

func promptCredentialsForFlow(reader *bufio.Reader, out io.Writer, flow interactiveAuthFlow) ([]any, string, error) {
	methods := slices.Clone(flow.Methods)
	slices.Sort(methods)

	credentials := make([]any, 0, len(methods))
	identityName := ""

	if slices.Contains(methods, "username_password") {
		username, err := promptRequired(reader, out, "Username: ")
		if err != nil {
			return nil, "", err
		}
		password, err := promptRequired(reader, out, "Password: ")
		if err != nil {
			return nil, "", err
		}
		credentials = append(credentials, map[string]any{
			"type":     "username_password",
			"username": username,
			"password": password,
		})
		identityName = username
	}
	if slices.Contains(methods, "authenticator") {
		secret, err := promptRequired(reader, out, "Authenticator secret: ")
		if err != nil {
			return nil, "", err
		}
		otp, err := promptText(reader, out, "Authenticator OTP (optional): ")
		if err != nil {
			return nil, "", err
		}
		auth := map[string]any{
			"type":   "authenticator",
			"secret": secret,
		}
		if strings.TrimSpace(otp) != "" {
			auth["otp"] = strings.TrimSpace(otp)
		}
		credentials = append(credentials, auth)
	}
	if slices.Contains(methods, "custom") {
		fields := make([]map[string]any, 0, len(flow.CustomFields))
		for _, fieldName := range flow.CustomFields {
			value, err := promptRequired(reader, out, fmt.Sprintf("%s: ", fieldName))
			if err != nil {
				return nil, "", err
			}
			fields = append(fields, map[string]any{
				"name":  fieldName,
				"value": value,
			})
		}
		credentials = append(credentials, map[string]any{
			"type":   "custom",
			"fields": fields,
		})
	}

	if len(credentials) == 0 {
		return nil, "", fmt.Errorf("selected auth flow has no supported methods")
	}
	return credentials, identityName, nil
}

func applyRecommendedAntiBotPayload(payload map[string]any) {
	session := ensureMap(payload, "session")
	proxy := ensureMap(session, "proxy")
	proxy["active"] = true
	proxy["type"] = "anchor_proxy"

	browser := ensureMap(payload, "browser")
	browser["extra_stealth"] = map[string]any{"active": true}
	browser["captcha_solver"] = map[string]any{"active": true}
}

func requiresManualIdentityFlow(methods []string) bool {
	if len(methods) == 0 {
		return true
	}
	supported := map[string]bool{
		"username_password": true,
		"authenticator":     true,
		"custom":            true,
	}
	for _, method := range methods {
		if method == "profile" {
			return true
		}
		if !supported[method] {
			return true
		}
	}
	return false
}

func extractIdentityRows(v any) []map[string]any {
	root, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	rawList, ok := root["identities"].([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(rawList))
	for _, item := range rawList {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func extractAuthFlows(v any) []interactiveAuthFlow {
	root, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	rawFlows, ok := root["auth_flows"].([]any)
	if !ok {
		return nil
	}
	out := make([]interactiveAuthFlow, 0, len(rawFlows))
	for _, item := range rawFlows {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := firstString(m["id"])
		name := firstString(m["name"])
		if id == "" || name == "" {
			continue
		}
		methods := []string{}
		if rawMethods, ok := m["methods"].([]any); ok {
			for _, rm := range rawMethods {
				if s, ok := rm.(string); ok && strings.TrimSpace(s) != "" {
					methods = append(methods, s)
				}
			}
		}
		customFields := []string{}
		if rawFields, ok := m["custom_fields"].([]any); ok {
			for _, rf := range rawFields {
				if fm, ok := rf.(map[string]any); ok {
					if fieldName := firstString(fm["name"]); fieldName != "" {
						customFields = append(customFields, fieldName)
					}
				}
			}
		}
		recommended, _ := m["is_recommended"].(bool)
		out = append(out, interactiveAuthFlow{
			ID:           id,
			Name:         name,
			Methods:      methods,
			CustomFields: customFields,
			Recommended:  recommended,
		})
	}
	return out
}

func extractIdentityID(v any) string {
	if root, ok := v.(map[string]any); ok {
		if id := firstString(root["id"]); id != "" {
			return id
		}
		if data, ok := root["data"].(map[string]any); ok {
			return firstString(data["id"])
		}
	}
	return ""
}

func mapGet(v any, path ...string) any {
	cur := v
	for _, p := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[p]
	}
	return cur
}

func normalizeApplicationSourceURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	u, err := neturl.Parse(value)
	if err != nil {
		return value
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	if u.Host == "" {
		return value
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	return u.String()
}

func deriveApplicationName(source string) string {
	u, err := neturl.Parse(source)
	if err != nil {
		return source
	}
	host := u.Hostname()
	if host == "" {
		return source
	}
	parts := strings.Split(strings.TrimPrefix(host, "www."), ".")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return host
	}
	base := parts[0]
	return strings.ToUpper(base[:1]) + base[1:]
}

func promptText(reader *bufio.Reader, out io.Writer, prompt string) (string, error) {
	_, _ = fmt.Fprint(out, prompt)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if errors.Is(err, io.EOF) && line == "" {
		return "", io.EOF
	}
	return strings.TrimSpace(line), nil
}

func promptRequired(reader *bufio.Reader, out io.Writer, prompt string) (string, error) {
	for {
		value, err := promptText(reader, out, prompt)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value), nil
		}
		_, _ = fmt.Fprintln(out, "Value is required.")
	}
}

func promptYesNo(reader *bufio.Reader, out io.Writer, prompt string, defaultYes bool) (bool, error) {
	for {
		value, err := promptText(reader, out, prompt)
		if err != nil {
			return false, err
		}
		value = strings.TrimSpace(strings.ToLower(value))
		if value == "" {
			return defaultYes, nil
		}
		switch value {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			_, _ = fmt.Fprintln(out, "Please answer yes or no.")
		}
	}
}

func openBrowserURL(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	return cmd.Start()
}
