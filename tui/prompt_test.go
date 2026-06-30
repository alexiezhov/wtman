package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPromptText_typeAndConfirm(t *testing.T) {
	m := NewPrompt().ActivateText("Branch name:")
	if !m.IsActive() {
		t.Fatal("prompt should be active")
	}
	for _, r := range "feat-x" {
		m, _ = m.Update(key(string(r)))
	}
	// Backspace removes the last char.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter should emit a result")
	}
	res, ok := cmd().(PromptResultMsg)
	if !ok {
		t.Fatalf("got %T, want PromptResultMsg", cmd())
	}
	if res.Cancelled {
		t.Error("Enter result should not be cancelled")
	}
	if res.Value != "feat-" {
		t.Errorf("value = %q, want feat-", res.Value)
	}
}

func TestPromptText_escapeCancels(t *testing.T) {
	m := NewPrompt().ActivateText("Branch name:")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("ESC should emit a result")
	}
	res := cmd().(PromptResultMsg)
	if !res.Cancelled {
		t.Error("ESC should set Cancelled")
	}
}

func TestPromptConfirm_enterAndY(t *testing.T) {
	for _, tc := range []struct {
		name string
		msg  tea.KeyMsg
		want bool
	}{
		{"enter confirms", tea.KeyMsg{Type: tea.KeyEnter}, true},
		{"y confirms", key("y"), true},
		{"Y confirms", key("Y"), true},
		{"n denies", key("n"), false},
		{"escape denies", tea.KeyMsg{Type: tea.KeyEscape}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := NewPrompt().ActivateConfirm("Delete?")
			_, cmd := m.Update(tc.msg)
			if cmd == nil {
				t.Fatal("confirm should emit a result")
			}
			res, ok := cmd().(ConfirmResultMsg)
			if !ok {
				t.Fatalf("got %T, want ConfirmResultMsg", cmd())
			}
			if res.Confirmed != tc.want {
				t.Errorf("Confirmed = %v, want %v", res.Confirmed, tc.want)
			}
		})
	}
}

func TestPromptConfirm_ignoresOtherKeys(t *testing.T) {
	m := NewPrompt().ActivateConfirm("Delete?")
	_, cmd := m.Update(key("x"))
	if cmd != nil {
		t.Error("confirm should ignore unrelated keys")
	}
	if !m.IsActive() {
		t.Error("confirm should stay active on unrelated keys")
	}
}

func TestPrompt_inactiveIgnoresKeys(t *testing.T) {
	m := NewPrompt()
	_, cmd := m.Update(key("x"))
	if cmd != nil {
		t.Error("inactive prompt should ignore keys")
	}
}
