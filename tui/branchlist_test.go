package tui

import (
	"testing"
	"time"

	"github.com/alexiezhov/wtman/core"
	tea "github.com/charmbracelet/bubbletea"
)

func fb(name string, day int) core.FeatureBranch {
	return core.FeatureBranch{
		Name:      name,
		CreatedAt: time.Date(2026, 1, day, 0, 0, 0, 0, time.UTC),
	}
}

func TestBranchList_sortByDateDescending(t *testing.T) {
	m := NewBranchList() // defaults to SortByDate
	m = m.SetBranches([]core.FeatureBranch{fb("old", 1), fb("new", 10), fb("mid", 5)})
	if got := m.branches[0].Name; got != "new" {
		t.Errorf("first by date = %q, want new (most recent)", got)
	}
	if got := m.branches[2].Name; got != "old" {
		t.Errorf("last by date = %q, want old", got)
	}
}

func TestBranchList_sortByName(t *testing.T) {
	m := NewBranchList().SetSortMode(SortByName)
	m = m.SetBranches([]core.FeatureBranch{fb("zebra", 1), fb("apple", 2), fb("mango", 3)})
	want := []string{"apple", "mango", "zebra"}
	for i, w := range want {
		if m.branches[i].Name != w {
			t.Errorf("branches[%d] = %q, want %q", i, m.branches[i].Name, w)
		}
	}
}

func TestBranchList_selectionStabilityByName(t *testing.T) {
	m := NewBranchList()
	m = m.SetBranches([]core.FeatureBranch{fb("a", 1), fb("b", 5), fb("c", 10)})
	// Move cursor to "b".
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	sel, _ := m.SelectedBranch()
	if sel.Name != "b" {
		t.Fatalf("selected = %q, want b", sel.Name)
	}
	// A refresh that reorders branches keeps "b" selected.
	m = m.SetBranches([]core.FeatureBranch{fb("c", 10), fb("b", 5), fb("a", 1), fb("d", 12)})
	sel, _ = m.SelectedBranch()
	if sel.Name != "b" {
		t.Errorf("after refresh selected = %q, want b", sel.Name)
	}
}

func TestBranchList_setSortModePreservesSelection(t *testing.T) {
	m := NewBranchList()
	m = m.SetBranches([]core.FeatureBranch{fb("a", 1), fb("b", 5), fb("c", 10)})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown}) // select b
	m = m.SetSortMode(SortByName)
	sel, _ := m.SelectedBranch()
	if sel.Name != "b" {
		t.Errorf("selection not preserved across sort change: %q", sel.Name)
	}
}

func TestBranchList_navigationBounds(t *testing.T) {
	m := NewBranchList()
	m = m.SetBranches([]core.FeatureBranch{fb("a", 1), fb("b", 2)})
	// Up at the top stays at index 0.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	// j moves down, then j again clamps at last.
	m, _ = m.Update(key("j"))
	m, _ = m.Update(key("j"))
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want clamped to 1", m.cursor)
	}
}

func TestBranchList_keyEmitsCommands(t *testing.T) {
	m := NewBranchList()
	m = m.SetBranches([]core.FeatureBranch{fb("a", 1)})

	cases := []struct {
		msg  tea.KeyMsg
		want string
	}{
		{key("d"), "/delete"},
		{tea.KeyMsg{Type: tea.KeyBackspace}, "/delete"},
		{tea.KeyMsg{Type: tea.KeyDelete}, "/delete"},
		{key("o"), "/open"},
		{key("?"), "/help"},
	}
	for _, tc := range cases {
		_, cmd := m.Update(tc.msg)
		if cmd == nil {
			t.Errorf("%v: no command emitted", tc.msg)
			continue
		}
		msg, ok := cmd().(CommandMsg)
		if !ok || msg.Name != tc.want {
			t.Errorf("%v: got %v, want CommandMsg %q", tc.msg, cmd(), tc.want)
		}
	}
}

func TestBranchList_enterEmitsBranchSelected(t *testing.T) {
	m := NewBranchList()
	m = m.SetBranches([]core.FeatureBranch{fb("a", 1)})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter should emit a command")
	}
	msg, ok := cmd().(BranchSelectedMsg)
	if !ok {
		t.Fatalf("got %T, want BranchSelectedMsg", cmd())
	}
	if msg.Branch.Name != "a" {
		t.Errorf("selected branch = %q, want a", msg.Branch.Name)
	}
}

func TestBranchList_emptyHasNoSelection(t *testing.T) {
	m := NewBranchList()
	if _, ok := m.SelectedBranch(); ok {
		t.Error("empty list should have no selection")
	}
}

// --- pure helper functions ---

func TestPadRight(t *testing.T) {
	if got := padRight("ab", 5); got != "ab   " {
		t.Errorf("padRight short = %q", got)
	}
	if got := padRight("abcdef", 3); got != "abc" {
		t.Errorf("padRight truncates = %q", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate noop = %q", got)
	}
	if got := truncate("a very long string", 8); got != "a ver..." {
		t.Errorf("truncate = %q, want a ver...", got)
	}
}

func TestScrollWindow(t *testing.T) {
	// Everything fits.
	if s, e := scrollWindow(0, 3, 10); s != 0 || e != 3 {
		t.Errorf("fits = %d,%d want 0,3", s, e)
	}
	// Cursor centered within a smaller window.
	s, e := scrollWindow(5, 20, 6)
	if e-s != 6 {
		t.Errorf("window size = %d, want 6", e-s)
	}
	if s > 5 || e <= 5 {
		t.Errorf("cursor 5 not within window %d,%d", s, e)
	}
	// Cursor at the end clamps the window to the tail.
	s, e = scrollWindow(19, 20, 6)
	if e != 20 || s != 14 {
		t.Errorf("end window = %d,%d want 14,20", s, e)
	}
}

func TestIndexOfBranchName(t *testing.T) {
	bs := []core.FeatureBranch{fb("a", 1), fb("b", 2)}
	if i := indexOfBranchName(bs, "b"); i != 1 {
		t.Errorf("index = %d, want 1", i)
	}
	if i := indexOfBranchName(bs, "missing"); i != -1 {
		t.Errorf("missing index = %d, want -1", i)
	}
	if i := indexOfBranchName(bs, ""); i != -1 {
		t.Errorf("empty name index = %d, want -1", i)
	}
}

func TestSortSliceAndMax(t *testing.T) {
	s := []int{3, 1, 2}
	sortSlice(s, func(a, b int) bool { return a < b })
	if s[0] != 1 || s[1] != 2 || s[2] != 3 {
		t.Errorf("sortSlice = %v", s)
	}
	if max(2, 5) != 5 || max(7, 3) != 7 {
		t.Error("max wrong")
	}
}
