package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// requireGit skips the test if the git binary is not available.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

// tempDir returns a fresh temp directory with symlinks resolved. On macOS
// t.TempDir() lives under /var which symlinks to /private/var; git records
// absolute, resolved worktree paths, so we resolve up front to keep the paths
// wtman computes and the paths git stored comparable.
func tempDir(t *testing.T) string {
	t.Helper()
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	return d
}

// git runs a git command in dir and fails the test on error.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := runGit(dir, args...)
	if err != nil {
		t.Fatalf("git %v in %s: %v", args, dir, err)
	}
	return out
}

// writeFile writes content to path, creating parent dirs, failing on error.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// initRepo creates a git repo at dir with a single commit on the `main` branch.
func initRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "init", "-q")
	configRepo(t, dir)
	writeFile(t, filepath.Join(dir, "README.md"), "hello\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "initial")
	git(t, dir, "branch", "-M", "main")
}

// configRepo sets identity and disables signing so commits work in CI.
func configRepo(t *testing.T, dir string) {
	t.Helper()
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "wtman test")
	git(t, dir, "config", "commit.gpgsign", "false")
}

// commitFile creates/updates a file and commits it in dir.
func commitFile(t *testing.T, dir, name, content, msg string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, name), content)
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", msg)
}

// sourceWith builds a sourceDir containing the named repos, each a fresh
// single-commit repo on main, and returns the matching []RepoEntry.
func sourceWith(t *testing.T, sourceDir string, names ...string) []RepoEntry {
	t.Helper()
	var repos []RepoEntry
	for _, n := range names {
		p := filepath.Join(sourceDir, n)
		initRepo(t, p)
		repos = append(repos, RepoEntry{Name: n, Path: p})
	}
	return repos
}
