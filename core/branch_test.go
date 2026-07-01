package core

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestBranchDirNameRoundTrip(t *testing.T) {
	cases := []string{"feat-x", "a/feat/add-field", "plain", "one/two"}
	for _, branch := range cases {
		dir := BranchToDirName(branch)
		if strings.Contains(dir, "/") {
			t.Errorf("BranchToDirName(%q) = %q still contains /", branch, dir)
		}
		if got := DirNameToBranch(dir); got != branch {
			t.Errorf("round trip %q -> %q -> %q", branch, dir, got)
		}
	}
}

func TestBranchToDirName_encodesSlashes(t *testing.T) {
	if got := BranchToDirName("a/feat/add-field"); got != "a--feat--add-field" {
		t.Errorf("got %q", got)
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"feat-x", false},
		{"a/feat/add-field", false},
		{"", true},
		{"   ", true},
		{"has space", true},
		{"-leading-dash", true},
		{"dot..dot", true},
	}
	for _, tt := range tests {
		err := SanitizeBranchName(tt.name)
		if (err != nil) != tt.wantErr {
			t.Errorf("SanitizeBranchName(%q) err=%v wantErr=%v", tt.name, err, tt.wantErr)
		}
	}
}

func TestListReposOnDisk_sortedAndGitOnly(t *testing.T) {
	requireGit(t)
	branchDir := tempDir(t)
	initRepo(t, filepath.Join(branchDir, "zeta"))
	initRepo(t, filepath.Join(branchDir, "alpha"))
	// a plain dir is not a repo and must be ignored
	if err := os.MkdirAll(filepath.Join(branchDir, "not-a-repo"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := ListReposOnDisk(branchDir)
	want := []string{"alpha", "zeta"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("ListReposOnDisk = %v, want %v", got, want)
	}
}

// TestCreateWorktrees_default covers the default-base path: a worktree per repo,
// each on the feature branch, plus the encoded directory layout.
func TestCreateWorktrees_default(t *testing.T) {
	requireGit(t)
	source := tempDir(t)
	target := tempDir(t)
	repos := sourceWith(t, source, "auth", "billing")

	if err := CreateWorktrees(source, repos, "feat/x", target, ""); err != nil {
		t.Fatalf("CreateWorktrees: %v", err)
	}

	branchDir := filepath.Join(target, "feat--x")
	on := ListReposOnDisk(branchDir)
	if strings.Join(on, ",") != "auth,billing" {
		t.Fatalf("repos on disk = %v", on)
	}
	for _, name := range repos {
		wt := filepath.Join(branchDir, name.Name)
		if br, _ := CurrentBranch(wt); br != "feat/x" {
			t.Errorf("%s on branch %q, want feat/x", name.Name, br)
		}
	}
}

// TestCreateWorktrees_fromBase covers the explicit --from base ref path.
func TestCreateWorktrees_fromBase(t *testing.T) {
	requireGit(t)
	source := tempDir(t)
	target := tempDir(t)
	repos := sourceWith(t, source, "auth")
	// Add a develop branch with a distinguishing commit.
	git(t, repos[0].Path, "checkout", "-q", "-b", "develop")
	commitFile(t, repos[0].Path, "DEV.md", "dev\n", "dev commit")
	git(t, repos[0].Path, "checkout", "-q", "main")

	if err := CreateWorktrees(source, repos, "feat-y", target, "develop"); err != nil {
		t.Fatalf("CreateWorktrees from develop: %v", err)
	}

	wt := filepath.Join(target, "feat-y", "auth")
	if _, err := os.Stat(filepath.Join(wt, "DEV.md")); err != nil {
		t.Errorf("expected worktree based on develop (DEV.md present): %v", err)
	}
}

func TestCreateWorktrees_missingBaseErrors(t *testing.T) {
	requireGit(t)
	source := tempDir(t)
	target := tempDir(t)
	repos := sourceWith(t, source, "auth")

	err := CreateWorktrees(source, repos, "feat-z", target, "nonexistent-ref")
	if err == nil {
		t.Fatal("expected error for missing base ref")
	}
}

func TestListFeatureBranches(t *testing.T) {
	requireGit(t)
	source := tempDir(t)
	target := tempDir(t)
	repos := sourceWith(t, source, "auth", "billing")

	if err := CreateWorktrees(source, repos, "feat/one", target, ""); err != nil {
		t.Fatal(err)
	}
	if err := CreateWorktrees(source, repos[:1], "feat-two", target, ""); err != nil {
		t.Fatal(err)
	}

	branches, err := ListFeatureBranches(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 2 {
		t.Fatalf("expected 2 branches, got %d", len(branches))
	}

	byName := map[string][]string{}
	for _, b := range branches {
		byName[b.Name] = b.Repos
	}
	if got := byName["feat/one"]; strings.Join(got, ",") != "auth,billing" {
		t.Errorf("feat/one repos = %v", got)
	}
	if got := byName["feat-two"]; strings.Join(got, ",") != "auth" {
		t.Errorf("feat-two repos = %v", got)
	}
}

func TestListFeatureBranches_missingDir(t *testing.T) {
	branches, err := ListFeatureBranches(filepath.Join(tempDir(t), "does-not-exist"))
	if err != nil {
		t.Fatalf("expected nil error for missing dir, got %v", err)
	}
	if branches != nil {
		t.Errorf("expected nil branches, got %v", branches)
	}
}

func TestUpdateFeatureBranch_addAndRemove(t *testing.T) {
	requireGit(t)
	source := tempDir(t)
	target := tempDir(t)
	repos := sourceWith(t, source, "auth", "billing", "payments")

	// Start with auth + billing.
	if err := CreateWorktrees(source, repos[:2], "feat", target, ""); err != nil {
		t.Fatal(err)
	}

	// Update to billing + payments (drop auth, add payments).
	desired := []RepoEntry{repos[1], repos[2]}
	if err := UpdateFeatureBranch(source, desired, "feat", target, false); err != nil {
		t.Fatalf("UpdateFeatureBranch: %v", err)
	}

	branchDir := filepath.Join(target, "feat")
	on := ListReposOnDisk(branchDir)
	if strings.Join(on, ",") != "billing,payments" {
		t.Errorf("repos after update = %v, want billing,payments", on)
	}
	if _, err := os.Stat(filepath.Join(branchDir, "auth")); !os.IsNotExist(err) {
		t.Errorf("auth worktree should be removed, stat err=%v", err)
	}
}

func TestDirtyRemovedWorktrees(t *testing.T) {
	requireGit(t)
	source := tempDir(t)
	target := tempDir(t)
	repos := sourceWith(t, source, "auth", "billing")
	if err := CreateWorktrees(source, repos, "feat", target, ""); err != nil {
		t.Fatal(err)
	}

	// Dirty the auth worktree (tracked-file change).
	writeFile(t, filepath.Join(target, "feat", "auth", "README.md"), "changed\n")

	// Desired set keeps only billing => auth would be removed and is dirty.
	dirty := DirtyRemovedWorktrees(repos[1:], "feat", target)
	if strings.Join(dirty, ",") != "auth" {
		t.Errorf("DirtyRemovedWorktrees = %v, want [auth]", dirty)
	}

	// If auth stays in the desired set, it is not a "removed" dirty repo.
	if d := DirtyRemovedWorktrees(repos, "feat", target); len(d) != 0 {
		t.Errorf("expected no dirty removals when nothing removed, got %v", d)
	}
}

func TestUpdateFeatureBranch_dirtyRemovalNeedsForce(t *testing.T) {
	requireGit(t)
	source := tempDir(t)
	target := tempDir(t)
	repos := sourceWith(t, source, "auth", "billing")
	if err := CreateWorktrees(source, repos, "feat", target, ""); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(target, "feat", "auth", "README.md"), "changed\n")

	// forceRemove=true must drop the dirty worktree.
	if err := UpdateFeatureBranch(source, repos[1:], "feat", target, true); err != nil {
		t.Fatalf("forced update: %v", err)
	}
	if on := ListReposOnDisk(filepath.Join(target, "feat")); strings.Join(on, ",") != "billing" {
		t.Errorf("after forced removal repos = %v, want billing", on)
	}
}

func TestDirtyBranchWorktrees(t *testing.T) {
	requireGit(t)
	source := tempDir(t)
	target := tempDir(t)
	repos := sourceWith(t, source, "auth", "billing")
	if err := CreateWorktrees(source, repos, "feat", target, ""); err != nil {
		t.Fatal(err)
	}
	if d := DirtyBranchWorktrees("feat", target); len(d) != 0 {
		t.Fatalf("fresh worktrees should be clean, got %v", d)
	}
	writeFile(t, filepath.Join(target, "feat", "billing", "README.md"), "dirty\n")
	if d := DirtyBranchWorktrees("feat", target); strings.Join(d, ",") != "billing" {
		t.Errorf("DirtyBranchWorktrees = %v, want [billing]", d)
	}
}

func TestDeleteFeatureBranch(t *testing.T) {
	requireGit(t)
	source := tempDir(t)
	target := tempDir(t)
	repos := sourceWith(t, source, "auth", "billing")
	if err := CreateWorktrees(source, repos, "feat", target, ""); err != nil {
		t.Fatal(err)
	}

	if err := DeleteFeatureBranch(target, "feat", false); err != nil {
		t.Fatalf("DeleteFeatureBranch: %v", err)
	}

	branchDir := filepath.Join(target, "feat")
	if _, err := os.Stat(branchDir); !os.IsNotExist(err) {
		t.Errorf("branch dir should be gone, stat err=%v", err)
	}
	// The branch should no longer exist in the source repos.
	for _, r := range repos {
		if branchExistsLocally(r.Path, "feat") {
			t.Errorf("branch feat still present in %s", r.Name)
		}
	}
}

func TestDeleteFeatureBranch_dirtyNeedsForce(t *testing.T) {
	requireGit(t)
	source := tempDir(t)
	target := tempDir(t)
	repos := sourceWith(t, source, "auth")
	if err := CreateWorktrees(source, repos, "feat", target, ""); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(target, "feat", "auth", "README.md"), "dirty\n")

	// force=true removes even dirty worktrees and the directory.
	if err := DeleteFeatureBranch(target, "feat", true); err != nil {
		t.Fatalf("forced delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "feat")); !os.IsNotExist(err) {
		t.Errorf("forced delete should remove branch dir, err=%v", err)
	}
}

func TestRenameFeatureBranch(t *testing.T) {
	requireGit(t)
	source := tempDir(t)
	target := tempDir(t)
	repos := sourceWith(t, source, "auth", "billing")
	if err := CreateWorktrees(source, repos, "old/name", target, ""); err != nil {
		t.Fatal(err)
	}

	if err := RenameFeatureBranch(target, "old/name", "new-name"); err != nil {
		t.Fatalf("RenameFeatureBranch: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, "old--name")); !os.IsNotExist(err) {
		t.Errorf("old dir should be gone")
	}
	newDir := filepath.Join(target, "new-name")
	if _, err := os.Stat(newDir); err != nil {
		t.Fatalf("new dir missing: %v", err)
	}
	for _, name := range []string{"auth", "billing"} {
		if br, _ := CurrentBranch(filepath.Join(newDir, name)); br != "new-name" {
			t.Errorf("%s on branch %q, want new-name", name, br)
		}
	}
}

func TestRenameFeatureBranch_existingTarget(t *testing.T) {
	requireGit(t)
	source := tempDir(t)
	target := tempDir(t)
	repos := sourceWith(t, source, "auth")
	if err := CreateWorktrees(source, repos, "a", target, ""); err != nil {
		t.Fatal(err)
	}
	if err := CreateWorktrees(source, repos, "b", target, ""); err != nil {
		t.Fatal(err)
	}
	if err := RenameFeatureBranch(target, "a", "b"); err == nil {
		t.Fatal("expected error renaming onto an existing branch")
	}
}

func TestRunPostCommand_substitutesPlaceholders(t *testing.T) {
	branchDir := tempDir(t)
	out := filepath.Join(branchDir, "out.txt")
	cmd := "printf '%s\\n%s\\n' '{{dir}}' '{{workspace}}' > " + out

	if err := RunPostCommand(cmd, branchDir); err != nil {
		t.Fatalf("RunPostCommand: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %q", string(data))
	}
	if lines[0] != branchDir {
		t.Errorf("{{dir}} = %q, want %q", lines[0], branchDir)
	}
	wantWS := filepath.Join(branchDir, WorkspaceFileName(branchDir))
	if lines[1] != wantWS {
		t.Errorf("{{workspace}} = %q, want %q", lines[1], wantWS)
	}
}

func TestRunPostCommand_empty(t *testing.T) {
	if err := RunPostCommand("", tempDir(t)); err != nil {
		t.Errorf("empty post command should be a no-op, got %v", err)
	}
}

func TestPullSourceRepos_pullsAndSkips(t *testing.T) {
	requireGit(t)
	root := tempDir(t)

	// Bare upstream + seed clone to populate it.
	upstream := filepath.Join(root, "upstream.git")
	// -b main so the bare repo's HEAD matches the branch the seed pushes below;
	// otherwise the runner's default (e.g. master) leaves HEAD dangling and the
	// app clone checks out nothing.
	git(t, root, "init", "--bare", "-q", "-b", "main", upstream)
	seed := filepath.Join(root, "seed")
	git(t, root, "clone", "-q", upstream, seed)
	configRepo(t, seed)
	commitFile(t, seed, "README.md", "v1\n", "v1")
	git(t, seed, "branch", "-M", "main")
	git(t, seed, "push", "-q", "-u", "origin", "main")

	// source/app is the repo wtman will pull.
	source := filepath.Join(root, "source")
	app := filepath.Join(source, "app")
	git(t, root, "clone", "-q", upstream, app)
	configRepo(t, app)

	// A detached-HEAD repo must be skipped (no remote => pull would error).
	det := filepath.Join(source, "detached")
	initRepo(t, det)
	git(t, det, "checkout", "-q", "--detach")

	// New upstream commit via seed.
	commitFile(t, seed, "README.md", "v2\n", "v2")
	git(t, seed, "push", "-q", "origin", "main")

	if err := PullSourceRepos(source, 1); err != nil {
		t.Fatalf("PullSourceRepos: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(app, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "v2" {
		t.Errorf("app not updated, README = %q", string(data))
	}
}

func TestListReposOnDisk_stableSort(t *testing.T) {
	requireGit(t)
	branchDir := tempDir(t)
	names := []string{"c", "a", "b"}
	for _, n := range names {
		initRepo(t, filepath.Join(branchDir, n))
	}
	got := ListReposOnDisk(branchDir)
	sorted := append([]string(nil), got...)
	sort.Strings(sorted)
	if strings.Join(got, ",") != strings.Join(sorted, ",") {
		t.Errorf("ListReposOnDisk not sorted: %v", got)
	}
}
