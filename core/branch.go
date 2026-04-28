package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// BranchCreatedAtLayout formats feature-branch creation time for CLI JSON and the TUI list (local time, minute precision).
const BranchCreatedAtLayout = "2006-01-02 15:04"

type FeatureBranch struct {
	Name           string // git branch name (may contain /)
	CreatedAt      time.Time
	Repos          []string // sorted repo names
	Path           string
	HasDirty       bool
	NonMasterRepos map[string]bool // repo names whose source repo is not on master/main
}

// BranchToDirName encodes a git branch name for use as a single directory name.
// Slashes are replaced with double dashes.
func BranchToDirName(branch string) string {
	return strings.ReplaceAll(branch, "/", "--")
}

// DirNameToBranch reverses BranchToDirName.
func DirNameToBranch(dirName string) string {
	return strings.ReplaceAll(dirName, "--", "/")
}

func ListFeatureBranches(targetDir string) ([]FeatureBranch, error) {
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	candidates := make([]os.DirEntry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			candidates = append(candidates, e)
		}
	}

	branches := make([]FeatureBranch, len(candidates))
	var wg sync.WaitGroup
	wg.Add(len(candidates))

	for i, e := range candidates {
		i, e := i, e
		go func() {
			defer wg.Done()
			branchDir := filepath.Join(targetDir, e.Name())
			repos := ListReposOnDisk(branchDir)

			info, _ := e.Info()
			var created time.Time
			if info != nil {
				created = info.ModTime()
			}

			hasDirty := false
			nonMaster := make(map[string]bool)

			var mu sync.Mutex
			var repoWg sync.WaitGroup
			repoWg.Add(len(repos))
			for _, repoName := range repos {
				repoName := repoName
				go func() {
					defer repoWg.Done()
					wtPath := filepath.Join(branchDir, repoName)
					dirty := IsWorktreeDirty(wtPath)
					mainRepo, err := mainRepoFromWorktree(wtPath)
					notMain := err == nil && !IsOnMainBranch(mainRepo)
					mu.Lock()
					if dirty {
						hasDirty = true
					}
					if notMain {
						nonMaster[repoName] = true
					}
					mu.Unlock()
				}()
			}
			repoWg.Wait()

			branches[i] = FeatureBranch{
				Name:           DirNameToBranch(e.Name()),
				CreatedAt:      created,
				Repos:          repos,
				Path:           branchDir,
				HasDirty:       hasDirty,
				NonMasterRepos: nonMaster,
			}
		}()
	}
	wg.Wait()

	return branches, nil
}

func ListReposOnDisk(branchDir string) []string {
	entries, err := os.ReadDir(branchDir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		full := filepath.Join(branchDir, e.Name())
		if IsGitRepo(full) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

func SanitizeBranchName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("branch name is empty")
	}
	if strings.Contains(name, "..") || strings.HasPrefix(name, "-") || strings.Contains(name, " ") {
		return fmt.Errorf("invalid branch name: %q", name)
	}
	return nil
}

func CreateWorktrees(sourceDir string, repos []RepoEntry, branch, targetDir string) error {
	if err := SanitizeBranchName(branch); err != nil {
		return err
	}
	branchDir := filepath.Join(targetDir, BranchToDirName(branch))
	if err := os.MkdirAll(branchDir, 0o755); err != nil {
		return err
	}

	var errs []string
	for _, repo := range repos {
		wtPath := filepath.Join(branchDir, repo.Name)
		if err := addWorktree(repo.Path, branch, wtPath); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", repo.Name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed repos:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

func RunPostCommand(postCmd, branchDir string) error {
	if postCmd == "" {
		return nil
	}
	expanded := strings.ReplaceAll(postCmd, "{{dir}}", branchDir)
	cmd := execCommand("sh", "-c", expanded)
	cmd.Dir = branchDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func DeleteFeatureBranch(targetDir, branch string, force bool) error {
	if err := SanitizeBranchName(branch); err != nil {
		return err
	}
	branchDir := filepath.Join(targetDir, BranchToDirName(branch))
	repos := ListReposOnDisk(branchDir)

	mainRepos := make(map[string]bool)
	for _, repoName := range repos {
		wtPath := filepath.Join(branchDir, repoName)
		main, err := mainRepoFromWorktree(wtPath)
		if err == nil {
			mainRepos[main] = true
		}

		args := []string{"worktree", "remove"}
		if force {
			args = append(args, "--force")
		}
		args = append(args, wtPath)
		if main != "" {
			_, _ = runGit(main, args...)
		}
	}

	for mainRepo := range mainRepos {
		_, _ = runGit(mainRepo, "worktree", "prune")
		deleteFlag := "-d"
		if force {
			deleteFlag = "-D"
		}
		_, _ = runGit(mainRepo, "branch", deleteFlag, branch)
	}

	return os.RemoveAll(branchDir)
}

func RenameFeatureBranch(targetDir, oldName, newName string) error {
	if err := SanitizeBranchName(newName); err != nil {
		return err
	}
	oldDir := filepath.Join(targetDir, BranchToDirName(oldName))
	newDir := filepath.Join(targetDir, BranchToDirName(newName))

	if _, err := os.Stat(newDir); err == nil {
		return fmt.Errorf("branch %q already exists", newName)
	}

	repos := ListReposOnDisk(oldDir)
	for _, repoName := range repos {
		wtPath := filepath.Join(oldDir, repoName)
		if _, err := runGit(wtPath, "branch", "-m", oldName, newName); err != nil {
			return fmt.Errorf("%s: %w", repoName, err)
		}
	}

	return os.Rename(oldDir, newDir)
}

func PullSourceRepos(sourceDir string, scanDepth int) error {
	repos, err := DiscoverRepos(sourceDir, scanDepth)
	if err != nil {
		return err
	}

	type result struct {
		name string
		err  error
	}
	ch := make(chan result, len(repos))

	for _, repo := range repos {
		go func(r RepoEntry) {
			branch, err := CurrentBranch(r.Path)
			if err != nil || branch == "HEAD" {
				// detached HEAD = linked worktree; skip
				ch <- result{}
				return
			}
			if !hasCheckedOutFiles(r.Path) {
				// uninitialized submodule — pulling would dirty the index; skip
				ch <- result{}
				return
			}
			_, err = runGit(r.Path, "pull", "--no-tags")
			ch <- result{r.Name, err}
		}(repo)
	}

	var errs []string
	for range repos {
		if r := <-ch; r.err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", r.name, r.err))
		}
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("pull failed:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

func DirtyRemovedWorktrees(repos []RepoEntry, branch, targetDir string) []string {
	branchDir := filepath.Join(targetDir, BranchToDirName(branch))
	current := ListReposOnDisk(branchDir)

	desiredSet := make(map[string]bool, len(repos))
	for _, r := range repos {
		desiredSet[r.Name] = true
	}

	var dirty []string
	for _, name := range current {
		if desiredSet[name] {
			continue
		}
		wtPath := filepath.Join(branchDir, name)
		if IsWorktreeDirty(wtPath) {
			dirty = append(dirty, name)
		}
	}
	return dirty
}

func UpdateFeatureBranch(sourceDir string, repos []RepoEntry, branch, targetDir string, forceRemove bool) error {
	branchDir := filepath.Join(targetDir, BranchToDirName(branch))
	current := ListReposOnDisk(branchDir)

	currentSet := make(map[string]bool, len(current))
	for _, name := range current {
		currentSet[name] = true
	}
	desiredSet := make(map[string]bool, len(repos))
	for _, r := range repos {
		desiredSet[r.Name] = true
	}

	// Remove repos no longer desired
	for _, name := range current {
		if desiredSet[name] {
			continue
		}
		wtPath := filepath.Join(branchDir, name)
		mainRepo, err := mainRepoFromWorktree(wtPath)
		if err == nil {
			args := []string{"worktree", "remove"}
			if forceRemove {
				args = append(args, "--force")
			}
			args = append(args, wtPath)
			_, _ = runGit(mainRepo, args...)
			_, _ = runGit(mainRepo, "worktree", "prune")
		}
		if _, err := os.Stat(wtPath); err == nil {
			_ = os.RemoveAll(wtPath)
		}
	}

	// Add new repos
	var errs []string
	for _, repo := range repos {
		if currentSet[repo.Name] {
			continue
		}
		wtPath := filepath.Join(branchDir, repo.Name)
		if err := addWorktree(repo.Path, branch, wtPath); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", repo.Name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed repos:\n  %s", strings.Join(errs, "\n  "))
	}

	return nil
}

func addWorktree(repoPath, branch, wtPath string) error {
	if _, err := os.Stat(wtPath); err == nil {
		return fmt.Errorf("path already exists: %s", wtPath)
	}
	if err := os.MkdirAll(filepath.Dir(wtPath), 0o755); err != nil {
		return err
	}

	// Clean up stale worktree refs so re-adding a previously removed repo works
	_, _ = runGit(repoPath, "worktree", "prune")

	if branchExistsLocally(repoPath, branch) {
		_, err := runGit(repoPath, "worktree", "add", wtPath, branch)
		return err
	}
	if branchExistsRemote(repoPath, branch) {
		_, err := runGit(repoPath, "worktree", "add", "-b", branch, wtPath, "origin/"+branch)
		return err
	}
	sp, err := defaultStartPoint(repoPath)
	if err != nil {
		return err
	}
	_, err = runGit(repoPath, "worktree", "add", "-b", branch, wtPath, sp)
	return err
}

var execCommand = newExecCommand

func newExecCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
