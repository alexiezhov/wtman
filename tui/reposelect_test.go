package tui

import (
	"testing"

	"github.com/alexiezhov/wtman/core"
	tea "github.com/charmbracelet/bubbletea"
)

func key(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func TestFuzzyScore(t *testing.T) {
	t.Run("empty pattern matches", func(t *testing.T) {
		if fuzzyScore("anything", "") != 1 {
			t.Error("empty pattern should score 1")
		}
	})
	t.Run("pattern longer than text", func(t *testing.T) {
		if fuzzyScore("ab", "abc") != 0 {
			t.Error("longer pattern should not match")
		}
	})
	t.Run("no match", func(t *testing.T) {
		if fuzzyScore("payment-gateway", "xyz") != 0 {
			t.Error("non-matching pattern should score 0")
		}
	})
	t.Run("subsequence matches", func(t *testing.T) {
		if fuzzyScore("payment-gateway", "pga") == 0 {
			t.Error("pga should fuzzy-match payment-gateway")
		}
	})
	t.Run("exact substring beats fuzzy", func(t *testing.T) {
		sub := fuzzyScore("payment-gateway", "gateway")
		fuzzy := fuzzyScore("payment-gateway", "pmt")
		if sub <= fuzzy {
			t.Errorf("substring score %d should beat fuzzy score %d", sub, fuzzy)
		}
	})
	t.Run("prefix beats non-prefix substring", func(t *testing.T) {
		prefix := fuzzyScore("payment", "pay")
		mid := fuzzyScore("repayment", "pay")
		if prefix <= mid {
			t.Errorf("prefix %d should beat mid-substring %d", prefix, mid)
		}
	})
}

func repos(names ...string) []core.RepoEntry {
	var r []core.RepoEntry
	for _, n := range names {
		r = append(r, core.RepoEntry{Name: n, Path: "/s/" + n})
	}
	return r
}

func TestRepoSelect_preSelect(t *testing.T) {
	m := NewRepoSelect().Activate(repos("auth", "billing", "payments"), []string{"billing"}, true, "update x")
	if m.SelectedCount() != 1 {
		t.Fatalf("SelectedCount = %d, want 1", m.SelectedCount())
	}
	sel := m.SelectedRepos()
	if len(sel) != 1 || sel[0].Name != "billing" {
		t.Errorf("SelectedRepos = %v", sel)
	}
}

func TestRepoSelect_toggleAndConfirm(t *testing.T) {
	m := NewRepoSelect().Activate(repos("auth", "billing"), nil, false, "new")
	// Space toggles the repo under the cursor (auth).
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if m.SelectedCount() != 1 {
		t.Fatalf("after toggle count = %d", m.SelectedCount())
	}

	// Enter with a selection emits ReposConfirmedMsg.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter with selection should emit a command")
	}
	msg := cmd()
	confirmed, ok := msg.(ReposConfirmedMsg)
	if !ok {
		t.Fatalf("got %T, want ReposConfirmedMsg", msg)
	}
	if len(confirmed.Repos) != 1 || confirmed.Repos[0].Name != "auth" {
		t.Errorf("confirmed repos = %v", confirmed.Repos)
	}
}

func TestRepoSelect_enterWithoutSelectionDoesNothing(t *testing.T) {
	m := NewRepoSelect().Activate(repos("auth"), nil, false, "new")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Enter with no selection should not emit a command")
	}
}

func TestRepoSelect_filterAndEscape(t *testing.T) {
	m := NewRepoSelect().Activate(repos("auth", "billing", "payment-gateway"), nil, false, "new")

	// Type a filter; only matching repos remain.
	m, _ = m.Update(key("pay"))
	if len(m.filtered) != 1 || m.filtered[0].Name != "payment-gateway" {
		t.Fatalf("filtered = %v", m.filtered)
	}

	// ESC clears the filter (does not cancel) when a filter is set.
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd != nil {
		t.Error("ESC with active filter should not cancel")
	}
	if m.filter != "" || len(m.filtered) != 3 {
		t.Errorf("filter not cleared: filter=%q filtered=%v", m.filter, m.filtered)
	}

	// ESC again (empty filter) cancels the mode.
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("ESC with empty filter should cancel")
	}
	if _, ok := cmd().(ReposCancelledMsg); !ok {
		t.Error("expected ReposCancelledMsg")
	}
}

func TestRepoSelect_filterRanksExactSubstringFirst(t *testing.T) {
	m := NewRepoSelect().Activate(repos("repayment", "payment"), nil, false, "new")
	m, _ = m.Update(key("payment"))
	if len(m.filtered) < 1 || m.filtered[0].Name != "payment" {
		t.Errorf("expected exact substring 'payment' ranked first, got %v", m.filtered)
	}
}

func TestRepoSelect_selectedReposInAllReposOrder(t *testing.T) {
	m := NewRepoSelect().Activate(repos("auth", "billing", "payments"), []string{"payments", "auth"}, true, "x")
	sel := m.SelectedRepos()
	// SelectedRepos iterates allRepos, so order follows the discovery order.
	if len(sel) != 2 || sel[0].Name != "auth" || sel[1].Name != "payments" {
		t.Errorf("SelectedRepos order = %v, want [auth payments]", sel)
	}
}
