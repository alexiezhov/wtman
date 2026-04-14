package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var allCommands = []string{
	"/new",
	"/delete",
	"/rename",
	"/source-dir",
	"/target-dir",
	"/sort-by-name",
	"/sort-by-date",
}

type StatusBarModel struct {
	active      bool
	input       string
	selectedIdx int
	width       int
}

func NewStatusBar() StatusBarModel {
	return StatusBarModel{selectedIdx: -1}
}

func (m StatusBarModel) IsActive() bool {
	return m.active
}

func (m StatusBarModel) Update(msg tea.Msg) (StatusBarModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		matches := m.matchingCommands()
		switch msg.Type {
		case tea.KeyEscape:
			m.active = false
			m.input = ""
			m.selectedIdx = -1
			return m, nil

		case tea.KeyEnter:
			cmd := m.resolveCommand()
			m.active = false
			m.input = ""
			m.selectedIdx = -1
			if cmd != "" {
				return m, func() tea.Msg {
					name, arg, _ := strings.Cut(cmd, " ")
					return CommandMsg{Name: name, Arg: strings.TrimSpace(arg)}
				}
			}
			return m, nil

		case tea.KeyTab, tea.KeyDown:
			if len(matches) > 0 {
				m.selectedIdx++
				if m.selectedIdx >= len(matches) {
					m.selectedIdx = 0
				}
				m.input = matches[m.selectedIdx]
			}
			return m, nil

		case tea.KeyShiftTab, tea.KeyUp:
			if len(matches) > 0 {
				m.selectedIdx--
				if m.selectedIdx < 0 {
					m.selectedIdx = len(matches) - 1
				}
				m.input = matches[m.selectedIdx]
			}
			return m, nil

		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
				m.selectedIdx = -1
			}
			if m.input == "" {
				m.active = false
			}
			return m, nil

		case tea.KeyRunes:
			m.input += string(msg.Runes)
			m.selectedIdx = -1
			return m, nil
		}
	}
	return m, nil
}

func (m StatusBarModel) Activate() StatusBarModel {
	m.active = true
	m.input = "/"
	m.selectedIdx = -1
	return m
}

func (m StatusBarModel) SetWidth(w int) StatusBarModel {
	m.width = w
	return m
}

func (m StatusBarModel) View() string {
	if !m.active {
		return ""
	}

	cursor := styleFilter.Render("\u2588")
	line := "  " + m.input + cursor

	matches := m.matchingCommands()
	if len(matches) > 0 {
		var rendered []string
		for i, c := range matches {
			if i == m.selectedIdx {
				rendered = append(rendered, lipgloss.NewStyle().
					Bold(true).
					Foreground(colorCyan).
					Render(c))
			} else {
				rendered = append(rendered, styleAutocomplete.Render(c))
			}
		}
		hint := "   " + strings.Join(rendered, "    ")
		return line + "\n" + hint
	}
	return line
}

func (m StatusBarModel) matchingCommands() []string {
	if m.input == "/" {
		return allCommands
	}
	var matches []string
	for _, c := range allCommands {
		if strings.HasPrefix(c, m.input) {
			matches = append(matches, c)
		}
	}
	return matches
}

func (m StatusBarModel) resolveCommand() string {
	matches := m.matchingCommands()
	// Exact match
	for _, c := range allCommands {
		if c == m.input {
			return c
		}
	}
	// If user has selected one via Tab/arrows, use it
	if m.selectedIdx >= 0 && m.selectedIdx < len(matches) {
		return matches[m.selectedIdx]
	}
	// Single match = autocomplete
	if len(matches) == 1 {
		return matches[0]
	}
	return ""
}
