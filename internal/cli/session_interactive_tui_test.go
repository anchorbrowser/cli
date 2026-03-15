package cli

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWizardDefaultsToYesOnAuthQuestion(t *testing.T) {
	model := newSessionCreateWizardModel(context.Background(), nil, "")
	if model.step != wizardStepAuthRequired {
		t.Fatalf("expected auth-required step, got %v", model.step)
	}
	if len(model.filtered) != 2 {
		t.Fatalf("expected 2 options, got %d", len(model.filtered))
	}
	selectedIdx := model.filtered[model.cursor]
	if model.options[selectedIdx].Value != "yes" {
		t.Fatalf("expected default selected option to be yes, got %s", model.options[selectedIdx].Value)
	}
}

func TestWizardRenderShowsBannerAndCompletedSection(t *testing.T) {
	model := newSessionCreateWizardModel(context.Background(), nil, "")
	view := model.View()

	if !strings.Contains(view, "_______             ______") {
		t.Fatalf("expected banner in wizard view")
	}
	if !strings.Contains(view, "Completed") {
		t.Fatalf("expected completed section in wizard view")
	}
	if !strings.Contains(view, "Do you need this session to be authenticated?") {
		t.Fatalf("expected current question title in wizard view")
	}
}

func TestWizardSelectFilteringBySearch(t *testing.T) {
	model := newSessionCreateWizardModel(context.Background(), nil, "")
	model.setupSelectStep(
		wizardStepIdentitySelect,
		"Select identity",
		[]tuiOption{
			{Label: "Alpha", Value: "a"},
			{Label: "Beta", Value: "b"},
			{Label: "Gamma", Value: "g"},
		},
		true,
		"",
		"",
	)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	m, ok := updated.(*sessionCreateWizardModel)
	if !ok {
		t.Fatalf("unexpected model type")
	}
	if len(m.filtered) != 1 {
		t.Fatalf("expected one filtered option, got %d", len(m.filtered))
	}
	if m.options[m.filtered[0]].Value != "g" {
		t.Fatalf("expected gamma option to remain")
	}
}

func TestWizardNoAuthFlowBuildsRecommendedPayload(t *testing.T) {
	model := newSessionCreateWizardModel(context.Background(), nil, "")

	// choose No for auth
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m, ok := updated.(*sessionCreateWizardModel)
	if !ok {
		t.Fatalf("unexpected model type")
	}
	if m.step != wizardStepAntiBot {
		t.Fatalf("expected anti-bot step, got %v", m.step)
	}

	// anti-bot defaults to Yes
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m, ok = updated.(*sessionCreateWizardModel)
	if !ok {
		t.Fatalf("unexpected model type")
	}
	if !m.done {
		t.Fatalf("expected wizard to be done")
	}

	session, ok := m.payload["session"].(map[string]any)
	if !ok {
		t.Fatalf("expected session payload to exist")
	}
	proxy, ok := session["proxy"].(map[string]any)
	if !ok {
		t.Fatalf("expected proxy payload")
	}
	if proxy["active"] != true || proxy["type"] != "anchor_proxy" {
		t.Fatalf("unexpected proxy payload: %#v", proxy)
	}

	summary, ok := m.payload[interactiveSummaryPayloadKey].([]string)
	if !ok {
		t.Fatalf("expected interactive summary to be attached")
	}
	if len(summary) < 2 {
		t.Fatalf("expected summary to include at least auth and anti-bot, got %#v", summary)
	}
}

func TestWizardAuthFlowTransitionsToIdentitySelection(t *testing.T) {
	model := newSessionCreateWizardModel(context.Background(), nil, "")

	// accept default Yes for auth
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if model.step != wizardStepApplicationURL {
		t.Fatalf("expected application URL step, got %v", model.step)
	}

	model.input.SetValue("https://netsweet.co")
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if model.step != wizardStepResolvingApplication {
		t.Fatalf("expected resolving application step, got %v", model.step)
	}

	updated, _ := model.Update(wizardResolveApplicationMsg{
		ApplicationID:  "app-123",
		ApplicationObj: map[string]any{"name": "NetSweet", "url": "https://netsweet.co"},
		SourceURL:      "https://netsweet.co",
	})
	m, ok := updated.(*sessionCreateWizardModel)
	if !ok {
		t.Fatalf("unexpected model type")
	}
	if m.step != wizardStepLoadingIdentities {
		t.Fatalf("expected loading identities step, got %v", m.step)
	}

	updated, _ = model.Update(wizardListIdentitiesMsg{Rows: []map[string]any{{"id": "id-1", "name": "Identity One"}}})
	m, ok = updated.(*sessionCreateWizardModel)
	if !ok {
		t.Fatalf("unexpected model type")
	}
	if m.step != wizardStepIdentitySelect {
		t.Fatalf("expected identity select step, got %v", m.step)
	}

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if model.step != wizardStepAntiBot {
		t.Fatalf("expected anti-bot step after selecting identity, got %v", model.step)
	}
	if got := model.payload["identities"]; got == nil {
		t.Fatalf("expected identity payload after identity selection")
	}
}
