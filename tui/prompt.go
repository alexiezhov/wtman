package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type PromptKind int

const (
	PromptText PromptKind = iota
	PromptConfirm
)

type PromptModel struct {
	kind    PromptKind
	label   string
	value   string
	active  bool
	width   int
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
	return m
}

func (m PromptModel) ActivateConfirm(label string) PromptModel {
	m.kind = PromptConfirm
	m.label = label
	m.value = ""
	m.active = true
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
		return m, func() tea.Msg { return PromptResultMsg{Value: m.value} }
	case tea.KeyBackspace:
		if len(m.value) > 0 {
			m.value = m.value[:len(m.value)-1]
		}
		return m, nil
	case tea.KeyRunes:
		m.value += string(msg.Runes)
		return m, nil
	}
	return m, nil
}

func (m PromptModel) updateConfirm(msg tea.KeyMsg) (PromptModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.active = false
		return m, func() tea.Msg { return ConfirmResultMsg{Confirmed: false} }
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
		hint := styleHint.Render("  y confirm  n/ESC cancel")
		return line + "\n" + hint
	}
	return ""
}
