package cli

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/anchorbrowser/cli/internal/api"
)

type wizardStep int

const (
	wizardStepAuthRequired wizardStep = iota
	wizardStepApplicationURL
	wizardStepResolvingApplication
	wizardStepLoadingIdentities
	wizardStepIdentitySelect
	wizardStepLoadingAuthFlows
	wizardStepAuthFlowSelect
	wizardStepCredentialInput
	wizardStepCreatingIdentity
	wizardStepManualDisplayName
	wizardStepManualToken
	wizardStepManualWaitingForBrowser
	wizardStepManualLookupRecent
	wizardStepManualPasteIdentityID
	wizardStepAntiBot
	wizardStepDone
)

type wizardMode int

const (
	wizardModeSelect wizardMode = iota
	wizardModeText
	wizardModeWait
	wizardModeLoading
)

type wizardCredentialField struct {
	Key        string
	Label      string
	Required   bool
	Secret     bool
	CustomName string
}

type wizardResolveApplicationMsg struct {
	ApplicationID  string
	ApplicationObj map[string]any
	SourceURL      string
	Err            error
}

type wizardListIdentitiesMsg struct {
	Rows []map[string]any
	Err  error
}

type wizardListFlowsMsg struct {
	Flows []interactiveAuthFlow
	Err   error
}

type wizardCreateIdentityMsg struct {
	IdentityID string
	Err        error
}

type wizardManualTokenMsg struct {
	ManualURL string
	OpenErr   error
	Err       error
}

type wizardRecentIdentityMsg struct {
	IdentityID string
	Err        error
}

type sessionCreateWizardModel struct {
	ctx    context.Context
	client *api.Client
	apiKey string

	step      wizardStep
	mode      wizardMode
	title     string
	hint      string
	notice    string
	errMsg    string
	completed []string
	payload   map[string]any

	options    []tuiOption
	searchable bool
	query      textinput.Model
	filtered   []int
	cursor     int

	input     textinput.Model
	required  bool
	validator func(string) string

	authenticated  bool
	applicationID  string
	applicationObj map[string]any
	applicationURL string
	identityID     string

	flowByID         map[string]interactiveAuthFlow
	selectedFlow     interactiveAuthFlow
	credentialFields []wizardCredentialField
	credentialValues map[string]string
	credentialIndex  int

	manualDisplayName string
	manualURL         string
	manualOpenErr     error

	done     bool
	canceled bool
	fatalErr error
}

func runInteractiveSessionCreateTUI(cmd *cobra.Command, app *App, apiKey string) (map[string]any, error) {
	model := newSessionCreateWizardModel(cmd.Context(), app.newAPIClient(), apiKey)
	p := tea.NewProgram(model, tea.WithInput(app.Stdin), tea.WithOutput(app.Stderr))
	result, err := p.Run()
	if err != nil {
		return nil, err
	}
	finalModel, ok := result.(*sessionCreateWizardModel)
	if !ok {
		return nil, fmt.Errorf("unexpected interactive wizard model result")
	}
	if finalModel.fatalErr != nil {
		return nil, finalModel.fatalErr
	}
	if finalModel.canceled {
		return nil, fmt.Errorf("interactive selection canceled")
	}
	if !finalModel.done {
		return nil, fmt.Errorf("interactive wizard ended before payload was finalized")
	}
	return finalModel.payload, nil
}

func newSessionCreateWizardModel(ctx context.Context, client *api.Client, apiKey string) *sessionCreateWizardModel {
	m := &sessionCreateWizardModel{
		ctx:      ctx,
		client:   client,
		apiKey:   apiKey,
		payload:  map[string]any{},
		flowByID: map[string]interactiveAuthFlow{},
	}
	m.setupSelectStep(
		wizardStepAuthRequired,
		"Do you need this session to be authenticated?",
		[]tuiOption{{Label: "Yes", Value: "yes"}, {Label: "No", Value: "no"}},
		false,
		"yes",
		"(↑/↓ to navigate, Enter to select, Esc to cancel)",
	)
	return m
}

func (m *sessionCreateWizardModel) Init() tea.Cmd {
	return nil
}

func (m *sessionCreateWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			m.fatalErr = fmt.Errorf("interactive selection canceled")
			return m, tea.Quit
		}
	}

	switch data := msg.(type) {
	case wizardResolveApplicationMsg:
		if data.Err != nil {
			m.fail(data.Err)
			return m, tea.Quit
		}
		m.applicationID = data.ApplicationID
		m.applicationObj = data.ApplicationObj
		m.applicationURL = data.SourceURL
		m.completed = append(m.completed, fmt.Sprintf("Application URL: %s", data.SourceURL))
		m.setupLoadingStep(wizardStepLoadingIdentities, "Loading identities", "Fetching identities for this application...")
		return m, m.cmdListIdentities(data.ApplicationID)

	case wizardListIdentitiesMsg:
		if data.Err != nil {
			m.fail(data.Err)
			return m, tea.Quit
		}
		options := make([]tuiOption, 0, len(data.Rows)+1)
		for _, row := range data.Rows {
			id := firstString(row["id"])
			if id == "" {
				continue
			}
			name := firstString(row["name"], id)
			options = append(options, tuiOption{Label: name, Value: id})
		}
		options = append(options, tuiOption{Label: "Create new identity", Value: "__create_identity__"})
		m.setupSelectStep(
			wizardStepIdentitySelect,
			"Select identity",
			options,
			true,
			"",
			"(↑/↓ to navigate, type to search, Enter to select, Esc to cancel)",
		)
		return m, nil

	case wizardListFlowsMsg:
		if data.Err != nil {
			m.fail(data.Err)
			return m, tea.Quit
		}
		if len(data.Flows) == 0 {
			m.setupManualDisplayNameStep()
			m.notice = "No auth flows found for this application. Falling back to manual identity creation."
			return m, nil
		}
		m.flowByID = map[string]interactiveAuthFlow{}
		options := make([]tuiOption, 0, len(data.Flows))
		for _, flow := range data.Flows {
			label := flow.Name
			if flow.Recommended {
				label += " (recommended)"
			}
			m.flowByID[flow.ID] = flow
			options = append(options, tuiOption{Label: label, Value: flow.ID})
		}
		m.setupSelectStep(
			wizardStepAuthFlowSelect,
			"Select authentication flow",
			options,
			true,
			"",
			"(↑/↓ to navigate, type to search, Enter to select, Esc to cancel)",
		)
		return m, nil

	case wizardCreateIdentityMsg:
		if data.Err != nil {
			m.fail(data.Err)
			return m, tea.Quit
		}
		m.attachIdentity(data.IdentityID)
		m.setupAntiBotStep()
		return m, nil

	case wizardManualTokenMsg:
		if data.Err != nil {
			m.fail(data.Err)
			return m, tea.Quit
		}
		m.manualURL = data.ManualURL
		m.manualOpenErr = data.OpenErr
		m.setupManualWaitingStep()
		return m, nil

	case wizardRecentIdentityMsg:
		if data.IdentityID != "" {
			m.attachIdentity(data.IdentityID)
			m.setupAntiBotStep()
			return m, nil
		}
		m.setupManualPasteIDStep()
		m.notice = "Could not auto-find a new identity by name. Please paste the identity ID."
		return m, nil
	}

	switch m.mode {
	case wizardModeSelect:
		selected, cmd := m.updateSelect(msg)
		if selected == "" {
			return m, cmd
		}
		return m, m.handleSelectedValue(selected)

	case wizardModeText:
		submitted, value, cmd := m.updateText(msg)
		if !submitted {
			return m, cmd
		}
		return m, m.handleTextValue(value)

	case wizardModeWait:
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			return m, m.handleWaitContinue()
		}
	}

	return m, nil
}

func (m *sessionCreateWizardModel) View() string {
	if m.done {
		return ""
	}
	body := m.renderBody()
	return renderInteractiveFrame(m.title, m.completed, body, m.hint)
}

func (m *sessionCreateWizardModel) renderBody() string {
	var b strings.Builder
	if strings.TrimSpace(m.notice) != "" {
		b.WriteString(m.notice)
		b.WriteString("\n\n")
	}
	switch m.mode {
	case wizardModeSelect:
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
				b.WriteString(prefix)
				b.WriteString(m.options[idx].Label)
				b.WriteString("\n")
			}
		}

	case wizardModeText:
		b.WriteString(m.input.View())
		if strings.TrimSpace(m.errMsg) != "" {
			b.WriteString("\n")
			b.WriteString(m.errMsg)
		}

	case wizardModeWait:
		if strings.TrimSpace(m.manualURL) != "" {
			b.WriteString("Open this URL to complete manual identity setup:\n")
			b.WriteString(m.manualURL)
			b.WriteString("\n")
		}
		if m.manualOpenErr != nil {
			b.WriteString("\nCould not open browser automatically: ")
			b.WriteString(m.manualOpenErr.Error())
			b.WriteString("\n")
		}
		b.WriteString("\nPress Enter after you finish manual identity creation in the browser.")

	case wizardModeLoading:
		if strings.TrimSpace(m.notice) == "" {
			b.WriteString("Working...")
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

func (m *sessionCreateWizardModel) setupSelectStep(step wizardStep, title string, options []tuiOption, searchable bool, defaultValue, hint string) {
	m.step = step
	m.mode = wizardModeSelect
	m.title = title
	m.hint = hint
	m.notice = ""
	m.errMsg = ""
	m.options = options
	m.searchable = searchable
	query := textinput.New()
	query.Prompt = "Search: "
	if searchable {
		query.Focus()
	} else {
		query.Blur()
	}
	m.query = query
	m.recomputeFilteredOptions()
	if defaultValue != "" {
		for i, idx := range m.filtered {
			if m.options[idx].Value == defaultValue {
				m.cursor = i
				break
			}
		}
	}
}

func (m *sessionCreateWizardModel) setupTextStep(step wizardStep, title string, required, secret bool, initialValue, hint string, validator func(string) string) {
	m.step = step
	m.mode = wizardModeText
	m.title = title
	m.hint = hint
	m.notice = ""
	m.required = required
	m.validator = validator
	m.errMsg = ""
	in := textinput.New()
	in.Focus()
	in.Prompt = "> "
	if secret {
		in.EchoMode = textinput.EchoPassword
		in.EchoCharacter = '•'
	}
	if strings.TrimSpace(initialValue) != "" {
		in.SetValue(strings.TrimSpace(initialValue))
	}
	m.input = in
}

func (m *sessionCreateWizardModel) setupLoadingStep(step wizardStep, title, message string) {
	m.step = step
	m.mode = wizardModeLoading
	m.title = title
	m.hint = ""
	m.errMsg = ""
	m.notice = message
}

func (m *sessionCreateWizardModel) setupAntiBotStep() {
	m.notice = ""
	m.setupSelectStep(
		wizardStepAntiBot,
		"Use recommended anti-bot settings (stealth + captcha solver + proxy)?",
		[]tuiOption{{Label: "Yes", Value: "yes"}, {Label: "No", Value: "no"}},
		false,
		"yes",
		"(↑/↓ to navigate, Enter to select, Esc to cancel)",
	)
}

func (m *sessionCreateWizardModel) setupManualDisplayNameStep() {
	m.manualDisplayName = ""
	m.setupTextStep(
		wizardStepManualDisplayName,
		"Display name for manual identity (optional)",
		false,
		false,
		"",
		"(Enter to confirm, Esc to cancel)",
		nil,
	)
}

func (m *sessionCreateWizardModel) setupManualWaitingStep() {
	m.step = wizardStepManualWaitingForBrowser
	m.mode = wizardModeWait
	m.title = "Manual identity browser completion"
	m.hint = "(Enter to continue, Esc to cancel)"
	m.errMsg = ""
}

func (m *sessionCreateWizardModel) setupManualPasteIDStep() {
	m.setupTextStep(
		wizardStepManualPasteIdentityID,
		"Paste created identity ID",
		true,
		false,
		"",
		"(Enter to confirm, Esc to cancel)",
		func(value string) string {
			if !uuidPattern.MatchString(value) {
				return "Identity ID must be a valid UUID."
			}
			return ""
		},
	)
}

func (m *sessionCreateWizardModel) setupCredentialInputStep(flow interactiveAuthFlow) {
	m.selectedFlow = flow
	m.credentialFields = buildCredentialFields(flow)
	m.credentialValues = map[string]string{}
	m.credentialIndex = 0
	m.notice = ""
	if len(m.credentialFields) == 0 {
		m.fail(fmt.Errorf("selected auth flow has no supported methods"))
		return
	}
	m.setupCurrentCredentialField()
}

func (m *sessionCreateWizardModel) setupCurrentCredentialField() {
	field := m.credentialFields[m.credentialIndex]
	title := field.Label
	if !field.Required {
		title += " (optional)"
	}
	m.setupTextStep(
		wizardStepCredentialInput,
		title,
		field.Required,
		field.Secret,
		"",
		"(Enter to confirm, Esc to cancel)",
		nil,
	)
}

func (m *sessionCreateWizardModel) updateSelect(msg tea.Msg) (string, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if len(m.filtered) > 0 && m.cursor > 0 {
				m.cursor--
			}
			return "", nil
		case "down", "j":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return "", nil
		case "enter":
			if len(m.filtered) == 0 {
				return "", nil
			}
			return m.options[m.filtered[m.cursor]].Value, nil
		}
	}

	if m.searchable {
		var cmd tea.Cmd
		m.query, cmd = m.query.Update(msg)
		m.recomputeFilteredOptions()
		return "", cmd
	}

	return "", nil
}

func (m *sessionCreateWizardModel) updateText(msg tea.Msg) (bool, string, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "enter" {
			value := strings.TrimSpace(m.input.Value())
			if m.required && value == "" {
				m.errMsg = "Value is required."
				return false, "", nil
			}
			if m.validator != nil {
				if validationErr := strings.TrimSpace(m.validator(value)); validationErr != "" {
					m.errMsg = validationErr
					return false, "", nil
				}
			}
			m.errMsg = ""
			return true, value, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.errMsg != "" {
		m.errMsg = ""
	}
	return false, "", cmd
}

func (m *sessionCreateWizardModel) handleSelectedValue(selected string) tea.Cmd {
	m.notice = ""
	switch m.step {
	case wizardStepAuthRequired:
		m.authenticated = selected == "yes"
		m.completed = append(m.completed, fmt.Sprintf("Authenticated session: %s", yesNoLabel(m.authenticated)))
		if m.authenticated {
			m.setupTextStep(
				wizardStepApplicationURL,
				"Application URL",
				true,
				false,
				"",
				"(Enter to confirm, Esc to cancel)",
				nil,
			)
			return nil
		}
		m.setupAntiBotStep()
		return nil

	case wizardStepIdentitySelect:
		if selected == "__create_identity__" {
			m.setupLoadingStep(wizardStepLoadingAuthFlows, "Loading authentication flows", "Fetching available authentication flows...")
			return m.cmdListAuthFlows(m.applicationID)
		}
		m.attachIdentity(selected)
		m.setupAntiBotStep()
		return nil

	case wizardStepAuthFlowSelect:
		flow, ok := m.flowByID[selected]
		if !ok {
			m.fail(fmt.Errorf("selected auth flow not found"))
			return tea.Quit
		}
		m.completed = append(m.completed, fmt.Sprintf("Auth flow: %s", flow.Name))
		if requiresManualIdentityFlow(flow.Methods) {
			m.setupManualDisplayNameStep()
			m.notice = "Selected flow requires manual browser completion."
			return nil
		}
		m.setupCredentialInputStep(flow)
		if m.fatalErr != nil {
			return tea.Quit
		}
		return nil

	case wizardStepAntiBot:
		useRecommended := selected == "yes"
		if useRecommended {
			applyRecommendedAntiBotPayload(m.payload)
		}
		m.completed = append(m.completed, fmt.Sprintf("Recommended anti-bot bundle: %s", yesNoLabel(useRecommended)))
		attachInteractiveSummary(m.payload, m.completed)
		m.step = wizardStepDone
		m.done = true
		return tea.Quit
	}

	return nil
}

func (m *sessionCreateWizardModel) handleTextValue(value string) tea.Cmd {
	m.notice = ""
	switch m.step {
	case wizardStepApplicationURL:
		m.setupLoadingStep(wizardStepResolvingApplication, "Resolving application", "Resolving application by URL...")
		return m.cmdResolveApplication(value)

	case wizardStepCredentialInput:
		field := m.credentialFields[m.credentialIndex]
		m.credentialValues[field.Key] = value
		m.credentialIndex++
		if m.credentialIndex < len(m.credentialFields) {
			m.setupCurrentCredentialField()
			return nil
		}
		m.setupLoadingStep(wizardStepCreatingIdentity, "Creating identity", "Creating identity from provided credentials...")
		return m.cmdCreateIdentity()

	case wizardStepManualDisplayName:
		m.manualDisplayName = strings.TrimSpace(value)
		if m.manualDisplayName != "" {
			m.completed = append(m.completed, fmt.Sprintf("Manual identity name: %s", m.manualDisplayName))
		}
		m.setupLoadingStep(wizardStepManualToken, "Preparing manual identity URL", "Generating identity token and browser link...")
		return m.cmdCreateManualToken()

	case wizardStepManualPasteIdentityID:
		m.attachIdentity(value)
		m.setupAntiBotStep()
		return nil
	}

	return nil
}

func (m *sessionCreateWizardModel) handleWaitContinue() tea.Cmd {
	if m.step != wizardStepManualWaitingForBrowser {
		return nil
	}
	if strings.TrimSpace(m.manualDisplayName) != "" {
		m.setupLoadingStep(wizardStepManualLookupRecent, "Finding recent identity", "Looking for newly created identity by name...")
		return m.cmdFindRecentIdentity()
	}
	m.setupManualPasteIDStep()
	return nil
}

func (m *sessionCreateWizardModel) recomputeFilteredOptions() {
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

func (m *sessionCreateWizardModel) attachIdentity(identityID string) {
	m.identityID = identityID
	m.payload["identities"] = []map[string]any{{"id": identityID}}
	m.completed = append(m.completed, fmt.Sprintf("Identity attached: %s", identityID))
}

func (m *sessionCreateWizardModel) fail(err error) {
	m.fatalErr = err
	m.done = false
}

func (m *sessionCreateWizardModel) cmdResolveApplication(rawURL string) tea.Cmd {
	return func() tea.Msg {
		appID, appObj, err := resolveApplicationForInteractive(m.ctx, m.client, m.apiKey, rawURL)
		return wizardResolveApplicationMsg{
			ApplicationID:  appID,
			ApplicationObj: appObj,
			SourceURL:      normalizeApplicationSourceURL(rawURL),
			Err:            err,
		}
	}
}

func (m *sessionCreateWizardModel) cmdListIdentities(applicationID string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.ApplicationListIdentities(m.ctx, m.apiKey, applicationID, url.Values{})
		if err != nil {
			return wizardListIdentitiesMsg{Err: err}
		}
		return wizardListIdentitiesMsg{Rows: extractIdentityRows(result)}
	}
}

func (m *sessionCreateWizardModel) cmdListAuthFlows(applicationID string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.ApplicationListAuthFlows(m.ctx, m.apiKey, applicationID)
		if err != nil {
			return wizardListFlowsMsg{Err: err}
		}
		return wizardListFlowsMsg{Flows: extractAuthFlows(result)}
	}
}

func (m *sessionCreateWizardModel) cmdCreateIdentity() tea.Cmd {
	flow := m.selectedFlow
	applicationObj := m.applicationObj
	applicationURL := m.applicationURL
	values := make(map[string]string, len(m.credentialValues))
	for k, v := range m.credentialValues {
		values[k] = v
	}

	return func() tea.Msg {
		credentials, identityName, err := buildCredentialsFromValues(flow, values)
		if err != nil {
			return wizardCreateIdentityMsg{Err: err}
		}
		source := normalizeApplicationSourceURL(firstString(mapGet(applicationObj, "url"), applicationURL))
		createPayload := map[string]any{
			"source":      source,
			"credentials": credentials,
		}
		if identityName != "" {
			createPayload["name"] = identityName
		}
		if appName := firstString(mapGet(applicationObj, "name")); appName != "" {
			createPayload["applicationName"] = appName
		}
		createResult, createErr := m.client.IdentityCreate(m.ctx, m.apiKey, true, createPayload)
		if createErr != nil {
			return wizardCreateIdentityMsg{Err: createErr}
		}
		identityID := extractIdentityID(createResult)
		if identityID == "" {
			return wizardCreateIdentityMsg{Err: fmt.Errorf("identity created but id was not returned")}
		}
		return wizardCreateIdentityMsg{IdentityID: identityID}
	}
}

func (m *sessionCreateWizardModel) cmdCreateManualToken() tea.Cmd {
	applicationID := m.applicationID
	userName := m.manualDisplayName
	return func() tea.Msg {
		tokenResult, err := m.client.ApplicationCreateToken(m.ctx, m.apiKey, applicationID, map[string]any{})
		if err != nil {
			return wizardManualTokenMsg{Err: err}
		}
		token := firstString(mapGet(tokenResult, "token"), mapGet(tokenResult, "data", "token"))
		if token == "" {
			return wizardManualTokenMsg{Err: fmt.Errorf("token response missing token")}
		}
		params := url.Values{}
		params.Set("token", token)
		if strings.TrimSpace(userName) != "" {
			params.Set("userName", strings.TrimSpace(userName))
		}
		manualURL := "https://app.anchorbrowser.io/identity/create?" + params.Encode()
		openErr := openBrowserURL(manualURL)
		return wizardManualTokenMsg{ManualURL: manualURL, OpenErr: openErr}
	}
}

func (m *sessionCreateWizardModel) cmdFindRecentIdentity() tea.Cmd {
	applicationID := m.applicationID
	userName := m.manualDisplayName
	return func() tea.Msg {
		identityID, err := findRecentIdentityByName(m.ctx, m.client, m.apiKey, applicationID, userName, 45*time.Minute)
		if err != nil {
			return wizardRecentIdentityMsg{Err: err}
		}
		return wizardRecentIdentityMsg{IdentityID: identityID}
	}
}

func buildCredentialFields(flow interactiveAuthFlow) []wizardCredentialField {
	methods := append([]string(nil), flow.Methods...)
	sort.Strings(methods)

	fields := make([]wizardCredentialField, 0, len(methods)+len(flow.CustomFields))
	if slicesContains(methods, "username_password") {
		fields = append(fields,
			wizardCredentialField{Key: "username", Label: "Username", Required: true},
			wizardCredentialField{Key: "password", Label: "Password", Required: true, Secret: true},
		)
	}
	if slicesContains(methods, "authenticator") {
		fields = append(fields,
			wizardCredentialField{Key: "auth_secret", Label: "Authenticator secret", Required: true},
			wizardCredentialField{Key: "auth_otp", Label: "Authenticator OTP", Required: false},
		)
	}
	if slicesContains(methods, "custom") {
		for i, name := range flow.CustomFields {
			trimmed := strings.TrimSpace(name)
			if trimmed == "" {
				continue
			}
			fields = append(fields, wizardCredentialField{
				Key:        fmt.Sprintf("custom_%d", i),
				Label:      trimmed,
				Required:   true,
				CustomName: trimmed,
			})
		}
	}
	return fields
}

func buildCredentialsFromValues(flow interactiveAuthFlow, values map[string]string) ([]any, string, error) {
	methods := append([]string(nil), flow.Methods...)
	sort.Strings(methods)

	credentials := make([]any, 0, len(methods))
	identityName := ""

	if slicesContains(methods, "username_password") {
		username := strings.TrimSpace(values["username"])
		password := strings.TrimSpace(values["password"])
		if username == "" || password == "" {
			return nil, "", fmt.Errorf("username and password are required")
		}
		credentials = append(credentials, map[string]any{
			"type":     "username_password",
			"username": username,
			"password": password,
		})
		identityName = username
	}

	if slicesContains(methods, "authenticator") {
		secret := strings.TrimSpace(values["auth_secret"])
		if secret == "" {
			return nil, "", fmt.Errorf("authenticator secret is required")
		}
		auth := map[string]any{"type": "authenticator", "secret": secret}
		if otp := strings.TrimSpace(values["auth_otp"]); otp != "" {
			auth["otp"] = otp
		}
		credentials = append(credentials, auth)
	}

	if slicesContains(methods, "custom") {
		fields := make([]map[string]any, 0, len(flow.CustomFields))
		for i, name := range flow.CustomFields {
			trimmedName := strings.TrimSpace(name)
			if trimmedName == "" {
				continue
			}
			value := strings.TrimSpace(values[fmt.Sprintf("custom_%d", i)])
			if value == "" {
				return nil, "", fmt.Errorf("%s is required", trimmedName)
			}
			fields = append(fields, map[string]any{"name": trimmedName, "value": value})
		}
		if len(fields) > 0 {
			credentials = append(credentials, map[string]any{"type": "custom", "fields": fields})
		}
	}

	if len(credentials) == 0 {
		return nil, "", fmt.Errorf("selected auth flow has no supported methods")
	}
	return credentials, identityName, nil
}

type tuiOption struct {
	Label string
	Value string
}

func slicesContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func renderInteractiveFrame(title string, completed []string, body, footer string) string {
	var b strings.Builder
	b.WriteString(strings.TrimPrefix(anchorBanner, "\n"))
	b.WriteString("\n\nCompleted\n")

	if len(completed) == 0 {
		b.WriteString("  [ ] (none yet)\n")
	} else {
		for _, item := range completed {
			b.WriteString("  [x] ")
			b.WriteString(item)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(body)
	if footer != "" {
		b.WriteString("\n\n")
		b.WriteString(footer)
	}
	return b.String()
}

func yesNoLabel(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}
