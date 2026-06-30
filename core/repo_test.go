package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverRepos_depth1(t *testing.T) {
	requireGit(t)
	source := tempDir(t)
	initRepo(t, filepath.Join(source, "zeta"))
	initRepo(t, filepath.Join(source, "alpha"))
	// hidden dir and a non-repo dir must be ignored
	initRepo(t, filepath.Join(source, ".hidden"))
	if err := os.MkdirAll(filepath.Join(source, "plain"), 0o755); err != nil {
		t.Fatal(err)
	}

	repos, err := DiscoverRepos(source, 1)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, r := range repos {
		names = append(names, r.Name)
	}
	if strings.Join(names, ",") != "alpha,zeta" {
		t.Errorf("DiscoverRepos = %v, want [alpha zeta]", names)
	}
}

func TestDiscoverRepos_nestedDepth(t *testing.T) {
	requireGit(t)
	source := tempDir(t)
	initRepo(t, filepath.Join(source, "group", "svc"))

	// Depth 1 should not find the nested repo.
	repos, err := DiscoverRepos(source, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 0 {
		t.Errorf("depth 1 found %v, want none", repos)
	}

	// Depth 2 should find it, named with a forward slash.
	repos, err = DiscoverRepos(source, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].Name != "group/svc" {
		t.Errorf("depth 2 = %v, want [group/svc]", repos)
	}
}

func TestDiscoverRepos_missingDir(t *testing.T) {
	if _, err := DiscoverRepos(filepath.Join(tempDir(t), "nope"), 1); err == nil {
		t.Fatal("expected error for missing source dir")
	}
}

func TestDiscoverRepos_clampsDepth(t *testing.T) {
	requireGit(t)
	source := tempDir(t)
	initRepo(t, filepath.Join(source, "a"))
	repos, err := DiscoverRepos(source, 0) // clamped to 1
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Errorf("got %d repos, want 1", len(repos))
	}
}

func TestIsGitRepo(t *testing.T) {
	requireGit(t)
	dir := tempDir(t)
	repo := filepath.Join(dir, "repo")
	initRepo(t, repo)
	if !IsGitRepo(repo) {
		t.Error("initialized repo not detected")
	}
	plain := filepath.Join(dir, "plain")
	if err := os.MkdirAll(plain, 0o755); err != nil {
		t.Fatal(err)
	}
	if IsGitRepo(plain) {
		t.Error("plain dir reported as git repo")
	}
}

func TestAnnotateNonMaster(t *testing.T) {
	requireGit(t)
	source := tempDir(t)
	repos := sourceWith(t, source, "onmain", "offmain")
	// Move the second repo off main.
	git(t, repos[1].Path, "checkout", "-q", "-b", "feature")

	annotated := AnnotateNonMaster(repos)
	byName := map[string]bool{}
	for _, r := range annotated {
		byName[r.Name] = r.NonMaster
	}
	if byName["onmain"] {
		t.Error("onmain should not be flagged NonMaster")
	}
	if !byName["offmain"] {
		t.Error("offmain should be flagged NonMaster")
	}
}
