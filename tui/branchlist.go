package tui

import (
	"fmt"
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
	branches []core.FeatureBranch
	cursor   int
	width    int
	height   int
	sortMode SortMode
}

func NewBranchList() BranchListModel {
	return BranchListModel{sortMode: SortByDate}
}

func (m BranchListModel) SetBranches(branches []core.FeatureBranch) BranchListModel {
	m.branches = branches
	m.sortBranches()
	if m.cursor >= len(m.branches) {
		m.cursor = max(0, len(m.branches)-1)
	}
	return m
}

func (m BranchListModel) SetSize(w, h int) BranchListModel {
	m.width = w
	m.height = h
	return m
}

func (m BranchListModel) SetSortMode(mode SortMode) BranchListModel {
	m.sortMode = mode
	m.sortBranches()
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
			}
		case tea.KeyDown:
			if m.cursor < len(m.branches)-1 {
				m.cursor++
			}
		case tea.KeyEnter:
			if b, ok := m.SelectedBranch(); ok {
				return m, func() tea.Msg { return BranchSelectedMsg{Branch: b} }
			}
		}
	}
	return m, nil
}

func (m BranchListModel) View() string {
	if m.width == 0 {
		return ""
	}

	dateW := 12
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
		"  " + strings.Repeat("\u2500", dateW) + "\u253c" +
			strings.Repeat("\u2500", branchW+2) + "\u253c" +
			strings.Repeat("\u2500", reposW+2))
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
		date := br.CreatedAt.Format("2006-01-02")
		repos := truncate(strings.Join(br.Repos, ", "), reposW)

		dateCol := padRight(date, dateW)
		branchCol := padRight(br.Name, branchW)
		reposCol := padRight(repos, reposW)

		row := "  " + dateCol + sep + branchCol + sep + reposCol

		if i == m.cursor {
			row = styleSelectedRow.Width(m.width).Render(
				"  " + padRight(date, dateW) + " \u2502 " +
					padRight(br.Name, branchW) + " \u2502 " +
					padRight(repos, reposW))
		}

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
		return styleHint.Render("  / command")
	}
	return styleHint.Render(fmt.Sprintf("  up/down navigate  ENTER update  / command"))
}
