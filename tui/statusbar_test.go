package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestStatusBar_activate(t *testing.T) {
	m := NewStatusBar().Activate()
	if !m.IsActive() {
		t.Fatal("status bar should be active after Activate")
	}
	// With just "/" typed, all commands match.
	if len(m.matchingCommands()) != len(allCommands) {
		t.Errorf("matching = %d, want all %d", len(m.matchingCommands()), len(allCommands))
	}
}

func TestStatusBar_fuzzyMatch(t *testing.T) {
	m := NewStatusBar().Activate()
	m, _ = m.Update(key("new"))
	matches := m.matchingCommands()
	if len(matches) == 0 || matches[0] != "/new" {
		t.Errorf("matches = %v, want /new first", matches)
	}
}

func TestStatusBar_enterResolvesSingleMatch(t *testing.T) {
	m := NewStatusBar().Activate()
	// "ren" fuzzy-matches only /rename.
	m, _ = m.Update(key("ren"))
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter should emit a command")
	}
	msg, ok := cmd().(CommandMsg)
	if !ok {
		t.Fatalf("got %T, want CommandMsg", cmd())
	}
	if msg.Name != "/rename" {
		t.Errorf("command = %q, want /rename", msg.Name)
	}
}

func TestStatusBar_enterExactMatch(t *testing.T) {
	m := NewStatusBar().Activate() // query already seeded with "/"
	for _, r := range "pull" {
		m, _ = m.Update(key(string(r)))
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter should emit a command for exact match")
	}
	if msg := cmd().(CommandMsg); msg.Name != "/pull" {
		t.Errorf("command = %q, want /pull", msg.Name)
	}
}

func TestStatusBar_tabCycles(t *testing.T) {
	m := NewStatusBar().Activate() // query "/" => all commands
	n := len(m.matchingCommands())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.selectedIdx != 0 {
		t.Errorf("after first Tab selectedIdx = %d, want 0", m.selectedIdx)
	}
	// Cycle past the end wraps to 0.
	for i := 0; i < n; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	if m.selectedIdx != 0 {
		t.Errorf("after wrapping selectedIdx = %d, want 0", m.selectedIdx)
	}

	// Shift+Tab from 0 wraps to last.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.selectedIdx != n-1 {
		t.Errorf("after Shift+Tab selectedIdx = %d, want %d", m.selectedIdx, n-1)
	}
}

func TestStatusBar_escapeCancels(t *testing.T) {
	m := NewStatusBar().Activate()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.IsActive() {
		t.Error("ESC should deactivate the status bar")
	}
}

func TestStatusBar_backspaceToEmptyDeactivates(t *testing.T) {
	m := NewStatusBar().Activate() // query == "/"
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.IsActive() {
		t.Error("backspacing the leading / should deactivate")
	}
}

func TestStatusBar_inactiveIgnoresKeys(t *testing.T) {
	m := NewStatusBar()
	_, cmd := m.Update(key("x"))
	if cmd != nil {
		t.Error("inactive status bar should ignore keys")
	}
}
