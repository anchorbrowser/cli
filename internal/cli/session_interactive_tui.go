package cli

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/anchorbrowser/cli/internal/api"
)

func runInteractiveSessionCreateTUI(cmd *cobra.Command, app *App, apiKey string) (map[string]any, error) {
	client := app.newAPIClient()
	payload := map[string]any{}

	authenticated, err := tuiSelectYesNo(app, "Do you need this session to be authenticated?", false)
	if err != nil {
		return nil, err
	}
	if authenticated {
		applicationURL, err := tuiPromptText(app, "Application URL", true, false)
		if err != nil {
			return nil, err
		}
		appID, appObj, err := resolveApplicationForInteractive(cmd.Context(), client, apiKey, applicationURL)
		if err != nil {
			return nil, err
		}
		identityID, err := interactiveSelectOrCreateIdentityTUI(cmd.Context(), app, client, apiKey, appID, appObj, applicationURL)
		if err != nil {
			return nil, err
		}
		payload["identities"] = []map[string]any{{"id": identityID}}
	}

	useRecommended, err := tuiSelectYesNo(app, "Use recommended anti-bot settings (stealth + captcha solver + proxy)?", false)
	if err != nil {
		return nil, err
	}
	if useRecommended {
		applyRecommendedAntiBotPayload(payload)
	}
	if authenticated {
		_, _ = fmt.Fprintln(app.Stderr, "Creating an authenticated session...")
	} else {
		_, _ = fmt.Fprintln(app.Stderr, "Creating a session...")
	}
	return payload, nil
}

func interactiveSelectOrCreateIdentityTUI(
	ctx context.Context,
	app *App,
	client *api.Client,
	apiKey, applicationID string,
	application map[string]any,
	applicationURL string,
) (string, error) {
	listResult, err := client.ApplicationListIdentities(ctx, apiKey, applicationID, url.Values{})
	if err != nil {
		return "", err
	}
	identities := extractIdentityRows(listResult)

	options := make([]tuiOption, 0, len(identities)+1)
	for _, ident := range identities {
		id := firstString(ident["id"])
		name := firstString(ident["name"], id)
		if id == "" {
			continue
		}
		options = append(options, tuiOption{Label: name, Value: id})
	}
	options = append(options, tuiOption{Label: "Create new identity", Value: "__create_identity__"})

	selected, err := tuiSelectWithSearch(app, "Select identity", options, true)
	if err != nil {
		return "", err
	}
	if selected == "__create_identity__" {
		return interactiveCreateIdentityTUI(ctx, app, client, apiKey, applicationID, application, applicationURL)
	}
	return selected, nil
}

func interactiveCreateIdentityTUI(
	ctx context.Context,
	app *App,
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
		_, _ = fmt.Fprintln(app.Stderr, "No auth flows found for this application; falling back to manual identity creation.")
		return interactiveCreateIdentityManualTUI(ctx, app, client, apiKey, applicationID)
	}

	options := make([]tuiOption, 0, len(flows))
	flowByID := map[string]interactiveAuthFlow{}
	for _, flow := range flows {
		label := flow.Name
		if flow.Recommended {
			label += " (recommended)"
		}
		options = append(options, tuiOption{Label: label, Value: flow.ID})
		flowByID[flow.ID] = flow
	}

	selectedID, err := tuiSelectWithSearch(app, "Select authentication flow", options, true)
	if err != nil {
		return "", err
	}
	selectedFlow, ok := flowByID[selectedID]
	if !ok {
		return "", fmt.Errorf("selected auth flow not found")
	}

	if requiresManualIdentityFlow(selectedFlow.Methods) {
		_, _ = fmt.Fprintln(app.Stderr, "Selected flow requires manual browser completion.")
		return interactiveCreateIdentityManualTUI(ctx, app, client, apiKey, applicationID)
	}

	credentials, identityName, err := promptCredentialsForFlowTUI(app, selectedFlow)
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

func interactiveCreateIdentityManualTUI(ctx context.Context, app *App, client *api.Client, apiKey, applicationID string) (string, error) {
	tokenResult, err := client.ApplicationCreateToken(ctx, apiKey, applicationID, map[string]any{})
	if err != nil {
		return "", err
	}
	token := firstString(mapGet(tokenResult, "token"), mapGet(tokenResult, "data", "token"))
	if token == "" {
		return "", fmt.Errorf("token response missing token")
	}

	userName, err := tuiPromptText(app, "Display name for manual identity (optional)", false, false)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Set("token", token)
	if strings.TrimSpace(userName) != "" {
		params.Set("userName", strings.TrimSpace(userName))
	}
	manualURL := "https://app.anchorbrowser.io/identity/create?" + params.Encode()
	_, _ = fmt.Fprintln(app.Stderr, "Open this URL to complete manual identity setup:")
	_, _ = fmt.Fprintln(app.Stderr, manualURL)
	if openErr := openBrowserURL(manualURL); openErr != nil {
		_, _ = fmt.Fprintf(app.Stderr, "Could not open browser automatically: %v\n", openErr)
	}
	_, _ = tuiPromptText(app, "Press Enter after finishing manual identity creation", false, false)

	if strings.TrimSpace(userName) != "" {
		if identityID, lookupErr := findRecentIdentityByName(ctx, client, apiKey, applicationID, userName, 45*time.Minute); lookupErr == nil && identityID != "" {
			_, _ = fmt.Fprintf(app.Stderr, "Found newly created identity by name: %s\n", identityID)
			return identityID, nil
		}
	}
	for {
		identityID, inputErr := tuiPromptText(app, "Paste created identity ID", true, false)
		if inputErr != nil {
			return "", inputErr
		}
		if uuidPattern.MatchString(identityID) {
			return identityID, nil
		}
		_, _ = fmt.Fprintln(app.Stderr, "Identity ID must be a valid UUID.")
	}
}

func promptCredentialsForFlowTUI(app *App, flow interactiveAuthFlow) ([]any, string, error) {
	methods := append([]string(nil), flow.Methods...)
	credentials := make([]any, 0, len(methods))
	identityName := ""

	if slicesContains(methods, "username_password") {
		username, err := tuiPromptText(app, "Username", true, false)
		if err != nil {
			return nil, "", err
		}
		password, err := tuiPromptText(app, "Password", true, true)
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
	if slicesContains(methods, "authenticator") {
		secret, err := tuiPromptText(app, "Authenticator secret", true, false)
		if err != nil {
			return nil, "", err
		}
		otp, err := tuiPromptText(app, "Authenticator OTP (optional)", false, false)
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
	if slicesContains(methods, "custom") {
		fields := make([]map[string]any, 0, len(flow.CustomFields))
		for _, fieldName := range flow.CustomFields {
			value, err := tuiPromptText(app, fieldName, true, false)
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

func slicesContains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

type tuiOption struct {
	Label string
	Value string
}

func tuiSelectYesNo(app *App, title string, defaultYes bool) (bool, error) {
	options := []tuiOption{
		{Label: "Yes", Value: "yes"},
		{Label: "No", Value: "no"},
	}
	if !defaultYes {
		options[0], options[1] = options[1], options[0]
	}
	choice, err := tuiSelectWithSearch(app, title, options, false)
	if err != nil {
		return false, err
	}
	return choice == "yes", nil
}

func tuiSelectWithSearch(app *App, title string, options []tuiOption, searchable bool) (string, error) {
	model := newTUISelectModel(title, options, searchable)
	p := tea.NewProgram(model, tea.WithInput(app.Stdin), tea.WithOutput(app.Stderr), tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return "", err
	}
	finalModel, ok := result.(tuiSelectModel)
	if !ok {
		return "", fmt.Errorf("unexpected select model result")
	}
	if finalModel.canceled {
		return "", fmt.Errorf("interactive selection canceled")
	}
	return finalModel.selectedValue, nil
}

func tuiPromptText(app *App, title string, required, secret bool) (string, error) {
	model := newTUITextModel(title, required, secret)
	p := tea.NewProgram(model, tea.WithInput(app.Stdin), tea.WithOutput(app.Stderr), tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return "", err
	}
	finalModel, ok := result.(tuiTextModel)
	if !ok {
		return "", fmt.Errorf("unexpected text model result")
	}
	if finalModel.canceled {
		return "", fmt.Errorf("interactive input canceled")
	}
	return strings.TrimSpace(finalModel.value), nil
}

type tuiTextModel struct {
	title    string
	input    textinput.Model
	required bool
	canceled bool
	value    string
	errMsg   string
}

func newTUITextModel(title string, required, secret bool) tuiTextModel {
	in := textinput.New()
	in.Focus()
	in.Prompt = "> "
	if secret {
		in.EchoMode = textinput.EchoPassword
		in.EchoCharacter = '•'
	}
	return tuiTextModel{
		title:    title,
		input:    in,
		required: required,
	}
}

func (m tuiTextModel) Init() tea.Cmd { return textinput.Blink }

func (m tuiTextModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			return m, tea.Quit
		case "enter":
			value := strings.TrimSpace(m.input.Value())
			if m.required && value == "" {
				m.errMsg = "Value is required."
				return m, nil
			}
			m.value = value
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.errMsg != "" {
		m.errMsg = ""
	}
	return m, cmd
}

func (m tuiTextModel) View() string {
	s := m.title + "\n\n" + m.input.View()
	if m.errMsg != "" {
		s += "\n" + m.errMsg
	}
	s += "\n\n(Enter to confirm, Esc to cancel)"
	return s
}

type tuiSelectModel struct {
	title         string
	options       []tuiOption
	searchable    bool
	query         textinput.Model
	filtered      []int
	cursor        int
	selectedValue string
	canceled      bool
}

func newTUISelectModel(title string, options []tuiOption, searchable bool) tuiSelectModel {
	q := textinput.New()
	q.Prompt = "Search: "
	if searchable {
		q.Focus()
	}
	m := tuiSelectModel{
		title:      title,
		options:    options,
		searchable: searchable,
		query:      q,
	}
	m.recompute()
	return m
}

func (m tuiSelectModel) Init() tea.Cmd { return textinput.Blink }

func (m tuiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			return m, tea.Quit
		case "up", "k":
			if len(m.filtered) > 0 && m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case "enter":
			if len(m.filtered) == 0 {
				return m, nil
			}
			m.selectedValue = m.options[m.filtered[m.cursor]].Value
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	if m.searchable {
		m.query, cmd = m.query.Update(msg)
		m.recompute()
	}
	return m, cmd
}

func (m *tuiSelectModel) recompute() {
	m.filtered = m.filtered[:0]
	query := strings.ToLower(strings.TrimSpace(m.query.Value()))
	for i, opt := range m.options {
		if query == "" || strings.Contains(strings.ToLower(opt.Label), query) {
			m.filtered = append(m.filtered, i)
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m tuiSelectModel) View() string {
	var b strings.Builder
	b.WriteString(m.title)
	b.WriteString("\n\n")
	if m.searchable {
		b.WriteString(m.query.View())
		b.WriteString("\n\n")
	}
	if len(m.filtered) == 0 {
		b.WriteString("No matches.")
	} else {
		for i, idx := range m.filtered {
			prefix := "  "
			if i == m.cursor {
				prefix = "> "
			}
			b.WriteString(prefix + m.options[idx].Label + "\n")
		}
	}
	b.WriteString("\n(↑/↓ to navigate, Enter to select, Esc to cancel)")
	return b.String()
}
