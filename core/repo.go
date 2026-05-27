package core

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type RepoEntry struct {
	Name      string
	Path      string
	NonMaster bool // source repo's current branch is not master/main
}

func DiscoverRepos(sourceDir string, maxDepth int) ([]RepoEntry, error) {
	if maxDepth < 1 {
		maxDepth = 1
	}
	info, err := os.Stat(sourceDir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "open", Path: sourceDir, Err: os.ErrNotExist}
	}

	var repos []RepoEntry
	if maxDepth == 1 {
		entries, err := os.ReadDir(sourceDir)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			full := filepath.Join(sourceDir, e.Name())
			if IsGitRepo(full) {
				repos = append(repos, RepoEntry{Name: e.Name(), Path: full})
			}
		}
	} else {
		err := filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(sourceDir, path)
			depth := len(strings.Split(rel, string(filepath.Separator)))
			if rel == "." {
				return nil
			}
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			if depth > maxDepth {
				return filepath.SkipDir
			}
			if IsGitRepo(path) {
				name := strings.ReplaceAll(rel, string(filepath.Separator), "/")
				repos = append(repos, RepoEntry{Name: name, Path: path})
				return filepath.SkipDir
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(repos, func(i, j int) bool {
		return strings.ToLower(repos[i].Name) < strings.ToLower(repos[j].Name)
	})
	slog.Debug("discovered repos", "source", sourceDir, "count", len(repos), "scan_depth", maxDepth)
	return repos, nil
}

// AnnotateNonMaster checks each repo's current branch in parallel and sets
// NonMaster=true for repos not on master/main. Call this only when the result
// is displayed to the user (repo select screen), not on every DiscoverRepos call.
func AnnotateNonMaster(repos []RepoEntry) []RepoEntry {
	annotated := make([]RepoEntry, len(repos))
	copy(annotated, repos)
	var wg sync.WaitGroup
	wg.Add(len(annotated))
	for i := range annotated {
		i := i
		go func() {
			defer wg.Done()
			annotated[i].NonMaster = !IsOnMainBranch(annotated[i].Path)
		}()
	}
	wg.Wait()
	return annotated
}
