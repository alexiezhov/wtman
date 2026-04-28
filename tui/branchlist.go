package tui

import (
	"strings"

	"github.com/hibobio/wtman/core"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SortMode int

const (
	SortByDate SortMode = iota
	SortByName
)

type BranchListModel struct {
	branches     []core.FeatureBranch
	cursor       int
	selectedName string
	width        int
	height       int
	sortMode     SortMode
}

func NewBranchList() BranchListModel {
	return BranchListModel{sortMode: SortByDate}
}

func (m BranchListModel) SetBranches(branches []core.FeatureBranch) BranchListModel {
	m.branches = branches
	m.sortBranches()
	if len(m.branches) == 0 {
		m.cursor = 0
		m.selectedName = ""
		return m
	}
	if idx := indexOfBranchName(m.branches, m.selectedName); idx >= 0 {
		m.cursor = idx
		return m
	}
	if m.cursor >= len(m.branches) {
		m.cursor = max(0, len(m.branches)-1)
	}
	m.selectedName = m.branches[m.cursor].Name
	return m
}

func (m BranchListModel) SetSize(w, h int) BranchListModel {
	m.width = w
	m.height = h
	return m
}

func (m BranchListModel) SetSortMode(mode SortMode) BranchListModel {
	name := m.selectedName
	if name == "" && len(m.branches) > 0 && m.cursor >= 0 && m.cursor < len(m.branches) {
		name = m.branches[m.cursor].Name
	}
	m.sortMode = mode
	m.sortBranches()
	if len(m.branches) == 0 {
		m.cursor = 0
		m.selectedName = ""
		return m
	}
	if idx := indexOfBranchName(m.branches, name); idx >= 0 {
		m.cursor = idx
		m.selectedName = name
		return m
	}
	if m.cursor >= len(m.branches) {
		m.cursor = max(0, len(m.branches)-1)
	}
	m.selectedName = m.branches[m.cursor].Name
	return m
}

func (m BranchListModel) SelectedBranch() (core.FeatureBranch, bool) {
	if len(m.branches) == 0 {
		return core.FeatureBranch{}, false
	}
	return m.branches[m.cursor], true
}

func (m BranchListModel) Update(msg tea.Msg) (BranchListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
				m.selectedName = m.branches[m.cursor].Name
			}
		case tea.KeyDown:
			if m.cursor < len(m.branches)-1 {
				m.cursor++
				m.selectedName = m.branches[m.cursor].Name
			}
		case tea.KeyEnter:
			if b, ok := m.SelectedBranch(); ok {
				return m, func() tea.Msg { return BranchSelectedMsg{Branch: b} }
			}
		case tea.KeyBackspace, tea.KeyDelete:
			return m, func() tea.Msg { return CommandMsg{Name: "/delete"} }
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "d":
				return m, func() tea.Msg { return CommandMsg{Name: "/delete"} }
			case "o":
				return m, func() tea.Msg { return CommandMsg{Name: "/open"} }
			case "?":
				return m, func() tea.Msg { return CommandMsg{Name: "/help"} }
			case "k":
				if m.cursor > 0 {
					m.cursor--
					m.selectedName = m.branches[m.cursor].Name
				}
			case "j":
				if m.cursor < len(m.branches)-1 {
					m.cursor++
					m.selectedName = m.branches[m.cursor].Name
				}
			}
		}
	}
	return m, nil
}

func (m BranchListModel) View() string {
	if m.width == 0 {
		return ""
	}

	dateW := 16 // width for core.BranchCreatedAtLayout
	branchW := 28
	sepW := 3 // " | "
	reposW := m.width - dateW - branchW - sepW*2 - 4
	if reposW < 10 {
		reposW = 10
	}

	sep := styleSeparator.Render(" \u2502 ")

	var b strings.Builder

	// Header
	hDate := styleHeader.Render(padRight("Date", dateW))
	hBranch := styleHeader.Render(padRight("Branch", branchW))
	hRepos := styleHeader.Render(padRight("Repos", reposW))
	b.WriteString("  " + hDate + sep + hBranch + sep + hRepos + "\n")

	// Separator line
	line := styleSeparator.Render(
		"  " + strings.Repeat("\u2500", dateW) + "\u2500\u253c\u2500" +
			strings.Repeat("\u2500", branchW) + "\u2500\u253c\u2500" +
			strings.Repeat("\u2500", reposW))
	b.WriteString(line + "\n")

	if len(m.branches) == 0 {
		b.WriteString(styleHint.Render("  (no feature branches)") + "\n")
		return b.String()
	}

	// Rows
	maxRows := m.height - 4
	if maxRows < 1 {
		maxRows = len(m.branches)
	}

	start, end := scrollWindow(m.cursor, len(m.branches), maxRows)
	for i := start; i < end; i++ {
		br := m.branches[i]
		date := br.CreatedAt.Format(core.BranchCreatedAtLayout)

		reposStyled := truncate(strings.Join(br.Repos, ", "), reposW)

		dateCol := padRight(date, dateW)

		if i == m.cursor {
			selPlain := styleSelectedRow.Render

			reposSelStr := selPlain(truncate(strings.Join(br.Repos, ", "), reposW))

			selSep := styleSelectedRow.Copy().Foreground(lipgloss.Color("240")).Render(" \u2502 ")
			// Pad the repos column to fill remaining width so the background band is full-width.
			usedW := 2 + dateW + 3 + branchW + 3 + reposW
			trailingW := max(0, m.width-usedW)
			reposColSel := padRightStyledWidth(reposSelStr, reposW+trailingW, selPlain)
			row := selPlain("  "+padRight(date, dateW)) + selSep + selPlain(padRight(br.Name, branchW)) + selSep + reposColSel
			b.WriteString(row + "\n")
			continue
		}

		reposCol := padRightWidth(reposStyled, reposW)
		row := "  " + dateCol + sep + padRight(br.Name, branchW) + sep + reposCol
		b.WriteString(row + "\n")
	}

	return b.String()
}

func (m *BranchListModel) sortBranches() {
	switch m.sortMode {
	case SortByName:
		sortSlice(m.branches, func(a, b core.FeatureBranch) bool {
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		})
	case SortByDate:
		sortSlice(m.branches, func(a, b core.FeatureBranch) bool {
			return a.CreatedAt.After(b.CreatedAt)
		})
	}
}

func padRight(s string, w int) string {
	if len(s) >= w {
		return s[:w]
	}
	return s + strings.Repeat(" ", w-len(s))
}

func truncate(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	if w <= 3 {
		return s[:w]
	}
	for lipgloss.Width(s) > w-3 {
		s = s[:len(s)-1]
	}
	return s + "..."
}

func scrollWindow(cursor, total, maxVisible int) (int, int) {
	if total <= maxVisible {
		return 0, total
	}
	start := cursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > total {
		end = total
		start = end - maxVisible
	}
	return start, end
}

func sortSlice[T any](s []T, less func(a, b T) bool) {
	n := len(s)
	for i := 1; i < n; i++ {
		for j := i; j > 0 && less(s[j], s[j-1]); j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m BranchListModel) HintView() string {
	if len(m.branches) == 0 {
		return styleHint.Render("  / command  ? help")
	}
	return styleHint.Render("  j/k navigate  ENTER update  o open  d delete  / command  ? help")
}

func indexOfBranchName(branches []core.FeatureBranch, name string) int {
	if name == "" {
		return -1
	}
	for i, b := range branches {
		if b.Name == name {
			return i
		}
	}
	return -1
}

func padRightWidth(s string, w int) string {
	n := lipgloss.Width(s)
	if n >= w {
		return s
	}
	return s + strings.Repeat(" ", w-n)
}

// padRightStyledWidth pads s to visual width w by appending spaces rendered
// through the provided style function, so the background color extends fully.
func padRightStyledWidth(s string, w int, styleFn func(...string) string) string {
	n := lipgloss.Width(s)
	if n >= w {
		return s
	}
	return s + styleFn(strings.Repeat(" ", w-n))
}
