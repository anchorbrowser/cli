package cli

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTUISelectModelFiltersByQuery(t *testing.T) {
	model := newTUISelectModel("Pick", []tuiOption{
		{Label: "Alpha", Value: "a"},
		{Label: "Beta", Value: "b"},
		{Label: "Gamma", Value: "g"},
	}, true)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	m, ok := updated.(tuiSelectModel)
	if !ok {
		t.Fatalf("unexpected model type")
	}
	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 filtered option, got %d", len(m.filtered))
	}
	if m.options[m.filtered[0]].Value != "g" {
		t.Fatalf("expected gamma to remain filtered")
	}
}

func TestTUITextModelRequiresValue(t *testing.T) {
	model := newTUITextModel("Input", true, false)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m, ok := updated.(tuiTextModel)
	if !ok {
		t.Fatalf("unexpected model type")
	}
	if m.errMsg == "" {
		t.Fatalf("expected required value error message")
	}
}
