package tui

import (
	"fmt"
	"strings"

	"github.com/hibobio/wtman/core"
	tea "github.com/charmbracelet/bubbletea"
)

type RepoSelectModel struct {
	allRepos []core.RepoEntry
	selected map[string]bool // keyed by repo name
	cursor   int
	filter   string
	filtered []core.RepoEntry
	width    int
	height   int
	isUpdate bool
	title    string
}

func NewRepoSelect() RepoSelectModel {
	return RepoSelectModel{
		selected: make(map[string]bool),
	}
}

func (m RepoSelectModel) Activate(repos []core.RepoEntry, preSelected []string, isUpdate bool, title string) RepoSelectModel {
	m.allRepos = repos
	m.selected = make(map[string]bool)
	for _, name := range preSelected {
		m.selected[name] = true
	}
	m.cursor = 0
	m.filter = ""
	m.isUpdate = isUpdate
	m.title = title
	m.applyFilter()
	return m
}

func (m RepoSelectModel) SetSize(w, h int) RepoSelectModel {
	m.width = w
	m.height = h
	return m
}

func (m RepoSelectModel) SetRepos(repos []core.RepoEntry) RepoSelectModel {
	m.allRepos = repos
	m.applyFilter()
	return m
}

func (m RepoSelectModel) SelectedRepos() []core.RepoEntry {
	var result []core.RepoEntry
	for _, r := range m.allRepos {
		if m.selected[r.Name] {
			result = append(result, r)
		}
	}
	return result
}

func (m RepoSelectModel) SelectedCount() int {
	count := 0
	for _, v := range m.selected {
		if v {
			count++
		}
	}
	return count
}

func (m RepoSelectModel) Update(msg tea.Msg) (RepoSelectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case tea.KeySpace:
			if len(m.filtered) > 0 {
				name := m.filtered[m.cursor].Name
				m.selected[name] = !m.selected[name]
			}
			return m, nil
		case tea.KeyEscape:
			if m.filter != "" {
				m.filter = ""
				m.applyFilter()
				return m, nil
			}
			return m, func() tea.Msg { return ReposCancelledMsg{} }
		case tea.KeyEnter:
			if m.SelectedCount() > 0 {
				repos := m.SelectedRepos()
				return m, func() tea.Msg { return ReposConfirmedMsg{Repos: repos} }
			}
			return m, nil
		case tea.KeyBackspace:
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.applyFilter()
			}
			return m, nil
		case tea.KeyRunes:
			m.filter += string(msg.Runes)
			m.applyFilter()
			return m, nil
		}
	}
	return m, nil
}

func (m *RepoSelectModel) applyFilter() {
	if m.filter == "" {
		m.filtered = m.allRepos
	} else {
		lower := strings.ToLower(m.filter)
		type scored struct {
			repo  core.RepoEntry
			score int
		}
		var matches []scored
		for _, r := range m.allRepos {
			if s := fuzzyScore(strings.ToLower(r.Name), lower); s > 0 {
				matches = append(matches, scored{repo: r, score: s})
			}
		}
		// Higher score first, then alphabetical
		for i := 1; i < len(matches); i++ {
			for j := i; j > 0; j-- {
				if matches[j].score > matches[j-1].score ||
					(matches[j].score == matches[j-1].score &&
						strings.ToLower(matches[j].repo.Name) < strings.ToLower(matches[j-1].repo.Name)) {
					matches[j], matches[j-1] = matches[j-1], matches[j]
				} else {
					break
				}
			}
		}
		m.filtered = make([]core.RepoEntry, len(matches))
		for i, s := range matches {
			m.filtered[i] = s.repo
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// fuzzyScore returns a positive score if pattern fuzzy-matches text, 0 if no match.
// Consecutive matches, prefix matches, and matches after separators score higher.
func fuzzyScore(text, pattern string) int {
	if len(pattern) == 0 {
		return 1
	}
	if len(pattern) > len(text) {
		return 0
	}

	// Exact substring gets a high bonus
	if strings.Contains(text, pattern) {
		score := 100
		if strings.HasPrefix(text, pattern) {
			score += 50
		}
		return score
	}

	// Fuzzy: each pattern char must appear in order in text
	score := 0
	pi := 0
	prevMatch := -2
	for ti := 0; ti < len(text) && pi < len(pattern); ti++ {
		if text[ti] == pattern[pi] {
			score += 10
			if ti == 0 {
				score += 20 // prefix bonus
			}
			if ti > 0 && (text[ti-1] == '-' || text[ti-1] == '_' || text[ti-1] == '/') {
				score += 15 // word boundary bonus
			}
			if ti == prevMatch+1 {
				score += 5 // consecutive bonus
			}
			prevMatch = ti
			pi++
		}
	}

	if pi < len(pattern) {
		return 0
	}
	return score
}

func (m RepoSelectModel) View() string {
	var b strings.Builder

	if len(m.filtered) == 0 {
		b.WriteString(styleHint.Render("  (no repos match filter)") + "\n")
	} else {
		maxRows := m.height - 6
		if maxRows < 1 {
			maxRows = len(m.filtered)
		}
		start, end := scrollWindow(m.cursor, len(m.filtered), maxRows)
		for i := start; i < end; i++ {
			r := m.filtered[i]
			if i == m.cursor {
				check := "[ ]"
				if m.selected[r.Name] {
					check = "[x]"
				}
				name := r.Name
				if r.NonMaster {
					name = styleError.Render("!") + name
				}
				line := fmt.Sprintf("  %s %s", check, name)
				b.WriteString(styleSelectedRow.Width(m.width).Render(line) + "\n")
			} else {
				check := styleCheckOff.Render("[ ]")
				if m.selected[r.Name] {
					check = styleCheckOn.Render("[x]")
				}
				name := r.Name
				if r.NonMaster {
					name = styleError.Render("!") + name
				}
				line := fmt.Sprintf("  %s %s", check, name)
				b.WriteString(line + "\n")
			}
		}
	}

	return b.String()
}

func (m RepoSelectModel) FilterView() string {
	if m.filter == "" {
		return ""
	}
	cursor := styleFilter.Render("\u2588")
	return "  " + styleFilter.Render("Filter: "+m.filter) + cursor
}

func (m RepoSelectModel) HintView() string {
	countStr := ""
	if c := m.SelectedCount(); c > 0 {
		countStr = fmt.Sprintf(" (%d)", c)
	}
	if m.filter != "" {
		return styleHint.Render(fmt.Sprintf("  up/down navigate  SPACE toggle  ESC clear filter  ENTER confirm%s", countStr))
	}
	return styleHint.Render(fmt.Sprintf("  up/down navigate  SPACE toggle  type to filter  ESC cancel  ENTER confirm%s", countStr))
}

func (m RepoSelectModel) Title() string {
	return m.title
}
