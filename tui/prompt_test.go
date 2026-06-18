package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestKebabCase(t *testing.T) {
	tests := []struct {
		name         string
		in           string
		trimTrailing bool
		want         string
	}{
		{"lowercase", "MyFeature", true, "myfeature"},
		{"space to dash", "My Feature", true, "my-feature"},
		{"collapse and trim on paste", "  My  Feature  ", true, "my-feature"},
		{"preserve slash", "FIX/Bug 123", true, "fix/bug-123"},
		{"collapse existing dashes", "a--b", true, "a-b"},
		{"keep trailing dash when typing", "my ", false, "my-"},
		{"trim trailing dash on submit", "my ", true, "my"},
		{"trim leading dash", "  feature", false, "feature"},
		{"tabs become dash", "a\tb", true, "a-b"},
		{"empty", "", true, ""},
		{"only whitespace trimmed", "   ", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := kebabCase(tt.in, tt.trimTrailing)
			if got != tt.want {
				t.Errorf("kebabCase(%q, %v) = %q, want %q", tt.in, tt.trimTrailing, got, tt.want)
			}
		})
	}
}

// typeRunes simulates per-keystroke input. A space is delivered as tea.KeySpace
// (as bubbletea does for a manual spacebar press), not as tea.KeyRunes.
func typeRunes(m PromptModel, s string) PromptModel {
	for _, r := range s {
		if r == ' ' {
			m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
			continue
		}
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

func TestPromptKebabTyping(t *testing.T) {
	m := NewPrompt().ActivateKebab("Branch name:")
	m = typeRunes(m, "My New Feature")
	if m.value != "my-new-feature" {
		t.Errorf("typed value = %q, want %q", m.value, "my-new-feature")
	}
}

func TestPromptKebabTypingKeepsTrailingDash(t *testing.T) {
	m := NewPrompt().ActivateKebab("Branch name:")
	m = typeRunes(m, "my ")
	if m.value != "my-" {
		t.Errorf("value after trailing space = %q, want %q", m.value, "my-")
	}
	// Next char must attach via the kept dash, not collapse into "myf".
	m = typeRunes(m, "f")
	if m.value != "my-f" {
		t.Errorf("value after next char = %q, want %q", m.value, "my-f")
	}
}

func TestPromptKebabPasteTrimsTrailing(t *testing.T) {
	m := NewPrompt().ActivateKebab("Branch name:")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("  Hello World  "), Paste: true})
	if m.value != "hello-world" {
		t.Errorf("pasted value = %q, want %q", m.value, "hello-world")
	}
}

func TestPromptKebabEnterTrimsTrailing(t *testing.T) {
	m := NewPrompt().ActivateKebab("Branch name:")
	m = typeRunes(m, "my feature ")
	if m.value != "my-feature-" {
		t.Fatalf("live value = %q, want %q", m.value, "my-feature-")
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from Enter")
	}
	msg := cmd()
	res, ok := msg.(PromptResultMsg)
	if !ok {
		t.Fatalf("expected PromptResultMsg, got %T", msg)
	}
	if res.Value != "my-feature" {
		t.Errorf("submitted value = %q, want %q", res.Value, "my-feature")
	}
}

func TestPromptTextNotKebab(t *testing.T) {
	m := NewPrompt().ActivateText("Base branch (default main/master):")
	m = typeRunes(m, "My Feature")
	if m.value != "My Feature" {
		t.Errorf("plain text value = %q, want %q (no normalization)", m.value, "My Feature")
	}
}
