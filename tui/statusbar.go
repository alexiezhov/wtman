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
	"/pull",
	"/source-dir",
	"/target-dir",
	"/sort-by-name",
	"/sort-by-date",
}

type StatusBarModel struct {
	active      bool
	query       string // what the user typed (used for filtering)
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
			m.query = ""
			m.selectedIdx = -1
			return m, nil

		case tea.KeyEnter:
			cmd := m.resolveCommand()
			m.active = false
			m.query = ""
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
			}
			return m, nil

		case tea.KeyShiftTab, tea.KeyUp:
			if len(matches) > 0 {
				m.selectedIdx--
				if m.selectedIdx < 0 {
					m.selectedIdx = len(matches) - 1
				}
			}
			return m, nil

		case tea.KeyBackspace:
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.selectedIdx = -1
			}
			if m.query == "" {
				m.active = false
			}
			return m, nil

		case tea.KeyRunes:
			m.query += string(msg.Runes)
			m.selectedIdx = -1
			return m, nil
		}
	}
	return m, nil
}

func (m StatusBarModel) Activate() StatusBarModel {
	m.active = true
	m.query = "/"
	m.selectedIdx = -1
	return m
}

func (m StatusBarModel) SetWidth(w int) StatusBarModel {
	m.width = w
	return m
}

func (m StatusBarModel) displayText() string {
	matches := m.matchingCommands()
	if m.selectedIdx >= 0 && m.selectedIdx < len(matches) {
		return matches[m.selectedIdx]
	}
	return m.query
}

func (m StatusBarModel) View() string {
	if !m.active {
		return ""
	}

	cursor := styleFilter.Render("\u2588")
	line := "  " + m.displayText() + cursor

	matches := m.matchingCommands()
	if len(matches) > 0 {
		var rendered []string
		for i, c := range matches {
			if i == m.selectedIdx {
				rendered = append(rendered, lipgloss.NewStyle().
					Bold(true).
					Foreground(colorAccent).
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
	if m.query == "/" {
		return allCommands
	}
	raw := strings.TrimPrefix(m.query, "/")
	if raw == "" {
		return allCommands
	}
	lower := strings.ToLower(raw)

	type scored struct {
		cmd   string
		score int
	}
	var matches []scored
	for _, c := range allCommands {
		name := strings.TrimPrefix(c, "/")
		if s := fuzzyScore(strings.ToLower(name), lower); s > 0 {
			matches = append(matches, scored{cmd: c, score: s})
		}
	}
	for i := 1; i < len(matches); i++ {
		for j := i; j > 0 && matches[j].score > matches[j-1].score; j-- {
			matches[j], matches[j-1] = matches[j-1], matches[j]
		}
	}
	result := make([]string, len(matches))
	for i, s := range matches {
		result[i] = s.cmd
	}
	return result
}

func (m StatusBarModel) resolveCommand() string {
	matches := m.matchingCommands()
	if m.selectedIdx >= 0 && m.selectedIdx < len(matches) {
		return matches[m.selectedIdx]
	}
	for _, c := range allCommands {
		if c == m.query {
			return c
		}
	}
	if len(matches) == 1 {
		return matches[0]
	}
	return ""
}
