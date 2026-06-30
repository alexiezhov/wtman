package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsWorktreeDirty(t *testing.T) {
	requireGit(t)
	repo := filepath.Join(tempDir(t), "repo")
	initRepo(t, repo)
	if IsWorktreeDirty(repo) {
		t.Error("fresh repo reported dirty")
	}
	writeFile(t, filepath.Join(repo, "README.md"), "changed\n")
	if !IsWorktreeDirty(repo) {
		t.Error("modified tracked file not reported dirty")
	}
}

func TestCurrentBranchAndIsOnMain(t *testing.T) {
	requireGit(t)
	repo := filepath.Join(tempDir(t), "repo")
	initRepo(t, repo)

	if br, err := CurrentBranch(repo); err != nil || br != "main" {
		t.Fatalf("CurrentBranch = %q, %v", br, err)
	}
	if !IsOnMainBranch(repo) {
		t.Error("expected IsOnMainBranch true on main")
	}

	git(t, repo, "checkout", "-q", "-b", "feature")
	if br, _ := CurrentBranch(repo); br != "feature" {
		t.Errorf("CurrentBranch = %q, want feature", br)
	}
	if IsOnMainBranch(repo) {
		t.Error("expected IsOnMainBranch false on feature")
	}
}

func TestBranchExistsLocally(t *testing.T) {
	requireGit(t)
	repo := filepath.Join(tempDir(t), "repo")
	initRepo(t, repo)
	if !branchExistsLocally(repo, "main") {
		t.Error("main should exist locally")
	}
	if branchExistsLocally(repo, "ghost") {
		t.Error("ghost branch should not exist")
	}
}

func TestDefaultStartPoint(t *testing.T) {
	requireGit(t)
	repo := filepath.Join(tempDir(t), "repo")
	initRepo(t, repo)
	sp, err := defaultStartPoint(repo)
	if err != nil {
		t.Fatal(err)
	}
	if sp != "main" {
		t.Errorf("defaultStartPoint = %q, want main", sp)
	}
}

func TestResolveStartPoint(t *testing.T) {
	requireGit(t)
	repo := filepath.Join(tempDir(t), "repo")
	initRepo(t, repo)
	git(t, repo, "branch", "develop")

	if sp, err := resolveStartPoint(repo, "develop"); err != nil || sp != "develop" {
		t.Errorf("resolveStartPoint(develop) = %q, %v", sp, err)
	}
	if _, err := resolveStartPoint(repo, "missing"); err == nil {
		t.Error("expected error for missing ref")
	}
}

func TestMainRepoFromWorktree(t *testing.T) {
	requireGit(t)
	source := tempDir(t)
	target := tempDir(t)
	repos := sourceWith(t, source, "auth")
	if err := CreateWorktrees(source, repos, "feat", target, ""); err != nil {
		t.Fatal(err)
	}

	wt := filepath.Join(target, "feat", "auth")
	main, err := mainRepoFromWorktree(wt)
	if err != nil {
		t.Fatalf("mainRepoFromWorktree: %v", err)
	}
	if main != repos[0].Path {
		t.Errorf("main repo = %q, want %q", main, repos[0].Path)
	}
}

func TestMainRepoFromWorktree_mainCheckout(t *testing.T) {
	requireGit(t)
	repo := filepath.Join(tempDir(t), "repo")
	initRepo(t, repo)
	// A primary checkout has a .git directory; it resolves to itself.
	main, err := mainRepoFromWorktree(repo)
	if err != nil {
		t.Fatal(err)
	}
	if main != repo {
		t.Errorf("main = %q, want %q", main, repo)
	}
}

func TestHasCheckedOutFiles(t *testing.T) {
	requireGit(t)
	repo := filepath.Join(tempDir(t), "repo")
	initRepo(t, repo)
	if !hasCheckedOutFiles(repo) {
		t.Error("repo with files should report checked out")
	}

	// A dir containing only .git mimics an uninitialized submodule.
	bare := filepath.Join(tempDir(t), "onlygit")
	if err := os.MkdirAll(filepath.Join(bare, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if hasCheckedOutFiles(bare) {
		t.Error("dir with only .git should report no checked out files")
	}
}
