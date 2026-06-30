package tui

import (
	"testing"

	"github.com/alexiezhov/wtman/core"
)

func TestRepoNames(t *testing.T) {
	got := repoNames([]core.RepoEntry{{Name: "auth"}, {Name: "billing"}})
	if len(got) != 2 || got[0] != "auth" || got[1] != "billing" {
		t.Errorf("repoNames = %v", got)
	}
}

func TestJoinErrors(t *testing.T) {
	if err := joinErrors(nil); err != nil {
		t.Errorf("joinErrors(nil) = %v, want nil", err)
	}
	if err := joinErrors([]string{}); err != nil {
		t.Errorf("joinErrors(empty) = %v, want nil", err)
	}
	err := joinErrors([]string{"a failed", "b failed"})
	if err == nil || err.Error() != "a failed; b failed" {
		t.Errorf("joinErrors = %v, want 'a failed; b failed'", err)
	}
}
