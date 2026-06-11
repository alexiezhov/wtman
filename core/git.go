package core

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ensureNoTags adds --no-tags to fetch and pull so wtman never downloads remote tags.
func ensureNoTags(args []string) []string {
	if len(args) == 0 {
		return args
	}
	switch args[0] {
	case "fetch", "pull":
		for _, a := range args[1:] {
			if a == "--no-tags" || a == "--tags" {
				return args
			}
		}
		out := make([]string, 0, len(args)+1)
		out = append(out, args[0], "--no-tags")
		out = append(out, args[1:]...)
		return out
	default:
		return args
	}
}

func runGit(repoDir string, args ...string) (string, error) {
	args = ensureNoTags(args)
	slog.Debug("git", "repo", filepath.Base(repoDir), "args", strings.Join(args, " "))
	cmd := exec.Command("git", append([]string{"-C", repoDir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"LANGUAGE=C",
		"LC_ALL=C",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func branchExistsLocally(repoDir, branch string) bool {
	_, err := runGit(repoDir, "rev-parse", "--verify", "--quiet", branch)
	return err == nil
}

func branchExistsRemote(repoDir, branch string) bool {
	_ = fetchOrigin(repoDir)
	_, err := runGit(repoDir, "rev-parse", "--verify", "--quiet", "origin/"+branch)
	return err == nil
}

func fetchOrigin(repoDir string) error {
	_, err := runGit(repoDir, "fetch", "--quiet", "origin")
	return err
}

func defaultStartPoint(repoDir string) (string, error) {
	for _, candidate := range []string{"main", "master"} {
		if branchExistsLocally(repoDir, candidate) {
			return candidate, nil
		}
		if branchExistsRemote(repoDir, candidate) {
			return "origin/" + candidate, nil
		}
	}
	return "", fmt.Errorf("no main/master branch found in %s", filepath.Base(repoDir))
}

// resolveStartPoint resolves an explicit user-supplied base ref to a start point
// usable by `git worktree add -b`. It prefers a local branch/tag/SHA, then falls
// back to origin/<ref>. Returns an error if the ref cannot be found.
func resolveStartPoint(repoDir, ref string) (string, error) {
	if branchExistsLocally(repoDir, ref) {
		return ref, nil
	}
	if branchExistsRemote(repoDir, ref) {
		return "origin/" + ref, nil
	}
	return "", fmt.Errorf("base ref %q not found in %s", ref, filepath.Base(repoDir))
}

func IsGitRepo(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir() || info.Mode().IsRegular()
}

func IsWorktreeDirty(wtPath string) bool {
	// diff-index only checks tracked files against the index, which is much
	// faster than `status --porcelain` on large repos (no untracked scan).
	_, err := runGit(wtPath, "diff-index", "--quiet", "HEAD")
	return err != nil
}

func IsOnMainBranch(repoDir string) bool {
	branch, err := CurrentBranch(repoDir)
	if err != nil {
		return true
	}
	return branch == "main" || branch == "master"
}

func CurrentBranch(repoDir string) (string, error) {
	out, err := runGit(repoDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// hasCheckedOutFiles returns false when the working tree has no files —
// only a .git entry exists. This identifies uninitialized submodules where
// pulling would update the index without checking out any files, leaving
// every tracked file appearing as deleted.
func hasCheckedOutFiles(repoDir string) bool {
	entries, err := os.ReadDir(repoDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.Name() != ".git" {
			return true
		}
	}
	return false
}

var gitdirRe = regexp.MustCompile(`(?m)^gitdir:\s*(.+)$`)

// mainRepoFromWorktree resolves the primary repo path from a linked worktree.
// A linked worktree has a .git file (not dir) containing "gitdir: <path>".
// That path points to .git/worktrees/<name> in the main repo.
func mainRepoFromWorktree(wtPath string) (string, error) {
	gitEntry := filepath.Join(wtPath, ".git")
	info, err := os.Stat(gitEntry)
	if err != nil {
		return "", fmt.Errorf("not a git worktree: %s", wtPath)
	}
	if info.IsDir() {
		return wtPath, nil
	}
	data, err := os.ReadFile(gitEntry)
	if err != nil {
		return "", err
	}
	m := gitdirRe.FindSubmatch(data)
	if m == nil {
		return "", fmt.Errorf("cannot parse .git file in %s", wtPath)
	}
	gd := strings.TrimSpace(string(m[1]))
	if !filepath.IsAbs(gd) {
		gd = filepath.Join(wtPath, gd)
	}
	// gd = <main-repo>/.git/worktrees/<name>
	// go up 3 levels: worktrees -> .git -> main-repo
	main := filepath.Dir(filepath.Dir(filepath.Dir(gd)))
	return filepath.Clean(main), nil
}
