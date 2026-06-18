package tui

import (
	"fmt"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

type PromptKind int

const (
	PromptText PromptKind = iota
	PromptConfirm
)

type PromptModel struct {
	kind   PromptKind
	label  string
	value  string
	active bool
	kebab  bool
	width  int
}

func NewPrompt() PromptModel {
	return PromptModel{}
}

func (m PromptModel) IsActive() bool {
	return m.active
}

func (m PromptModel) ActivateText(label string) PromptModel {
	m.kind = PromptText
	m.label = label
	m.value = ""
	m.active = true
	m.kebab = false
	return m
}

// ActivateKebab activates a text prompt whose input is normalized to kebab-case
// as the user types or pastes (lowercase, whitespace to dash, collapsed/trimmed).
func (m PromptModel) ActivateKebab(label string) PromptModel {
	m.kind = PromptText
	m.label = label
	m.value = ""
	m.active = true
	m.kebab = true
	return m
}

func (m PromptModel) ActivateConfirm(label string) PromptModel {
	m.kind = PromptConfirm
	m.label = label
	m.value = ""
	m.active = true
	m.kebab = false
	return m
}

func (m PromptModel) SetWidth(w int) PromptModel {
	m.width = w
	return m
}

func (m PromptModel) Update(msg tea.Msg) (PromptModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.kind == PromptConfirm {
			return m.updateConfirm(msg)
		}
		return m.updateText(msg)
	}
	return m, nil
}

func (m PromptModel) updateText(msg tea.KeyMsg) (PromptModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.active = false
		return m, func() tea.Msg { return PromptResultMsg{Cancelled: true} }
	case tea.KeyEnter:
		m.active = false
		val := m.value
		if m.kebab {
			val = kebabCase(val, true)
		}
		return m, func() tea.Msg { return PromptResultMsg{Value: val} }
	case tea.KeyBackspace:
		if len(m.value) > 0 {
			m.value = m.value[:len(m.value)-1]
		}
		return m, nil
	case tea.KeyRunes, tea.KeySpace:
		if msg.Type == tea.KeySpace {
			m.value += " "
		} else {
			m.value += string(msg.Runes)
		}
		if m.kebab {
			m.value = kebabCase(m.value, msg.Paste)
		}
		return m, nil
	}
	return m, nil
}

// kebabCase normalizes s to kebab-case: lowercases, converts whitespace and dash
// runs into a single dash, and trims the leading dash. The trailing dash is only
// trimmed when trimTrailing is true (on paste and on submit) so that incremental
// typing of a separator followed by more text is preserved.
func kebabCase(s string, trimTrailing bool) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if unicode.IsSpace(r) || r == '-' {
			if !prevDash {
				b.WriteRune('-')
				prevDash = true
			}
			continue
		}
		b.WriteRune(r)
		prevDash = false
	}
	out := strings.TrimLeft(b.String(), "-")
	if trimTrailing {
		out = strings.TrimRight(out, "-")
	}
	return out
}

func (m PromptModel) updateConfirm(msg tea.KeyMsg) (PromptModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.active = false
		return m, func() tea.Msg { return ConfirmResultMsg{Confirmed: false} }
	case tea.KeyEnter:
		m.active = false
		return m, func() tea.Msg { return ConfirmResultMsg{Confirmed: true} }
	case tea.KeyRunes:
		ch := string(msg.Runes)
		if ch == "y" || ch == "Y" {
			m.active = false
			return m, func() tea.Msg { return ConfirmResultMsg{Confirmed: true} }
		}
		if ch == "n" || ch == "N" {
			m.active = false
			return m, func() tea.Msg { return ConfirmResultMsg{Confirmed: false} }
		}
	}
	return m, nil
}

func (m PromptModel) View() string {
	if !m.active {
		return ""
	}
	cursor := styleFilter.Render("\u2588")
	switch m.kind {
	case PromptText:
		line := fmt.Sprintf("  %s %s%s", m.label, m.value, cursor)
		hint := styleHint.Render("  ENTER confirm  ESC cancel")
		return line + "\n" + hint
	case PromptConfirm:
		line := fmt.Sprintf("  %s", m.label)
		hint := styleHint.Render("  ENTER/y confirm  ESC/n cancel")
		return line + "\n" + hint
	}
	return ""
}
