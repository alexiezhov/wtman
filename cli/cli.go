package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hibobio/wtman/core"
)

func Run(cfg core.Config, args []string) {
	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "ls":
		cmdLS(cfg, rest)
	case "repos":
		cmdRepos(cfg, rest)
	case "new":
		cmdNew(cfg, rest)
	case "rm":
		cmdRM(cfg, rest)
	case "update":
		cmdUpdate(cfg, rest)
	case "mv":
		cmdMV(cfg, rest)
	case "pull":
		cmdPull(cfg, rest)
	default:
		die("unknown command: " + cmd)
	}
}

func cmdLS(cfg core.Config, args []string) {
	fs := flag.NewFlagSet("ls", flag.ExitOnError)
	fs.Usage = func() { fmt.Fprintln(os.Stderr, "Usage: wtman ls"); os.Exit(0) }
	fs.Parse(args)

	branches, err := core.ListFeatureBranches(cfg.TargetDir)
	if err != nil {
		die(err.Error())
	}

	out := make([]map[string]any, 0, len(branches))
	for _, b := range branches {
		nm := make([]string, 0, len(b.NonMasterRepos))
		for k := range b.NonMasterRepos {
			nm = append(nm, k)
		}
		out = append(out, map[string]any{
			"name":       b.Name,
			"date":       b.CreatedAt.Format("2006-01-02"),
			"repos":      b.Repos,
			"path":       b.Path,
			"dirty":      b.HasDirty,
			"non_master": nm,
		})
	}
	jsonOut(out)
}

func cmdRepos(cfg core.Config, args []string) {
	fs := flag.NewFlagSet("repos", flag.ExitOnError)
	fs.Usage = func() { fmt.Fprintln(os.Stderr, "Usage: wtman repos"); os.Exit(0) }
	fs.Parse(args)

	repos, err := core.DiscoverRepos(cfg.SourceDir, cfg.ScanDepth)
	if err != nil {
		die(err.Error())
	}

	out := make([]map[string]any, 0, len(repos))
	for _, r := range repos {
		out = append(out, map[string]any{"name": r.Name, "path": r.Path})
	}
	jsonOut(out)
}

func cmdNew(cfg core.Config, args []string) {
	fs := flag.NewFlagSet("new", flag.ExitOnError)
	noHook := fs.Bool("n", false, "skip post_command hook")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: wtman new <branch> <repo,repo,...> [-n]")
		os.Exit(0)
	}
	fs.Parse(args)

	if fs.NArg() < 2 {
		fs.Usage()
	}
	branch := fs.Arg(0)
	repoNames := strings.Split(fs.Arg(1), ",")

	allRepos, err := core.DiscoverRepos(cfg.SourceDir, cfg.ScanDepth)
	if err != nil {
		die(err.Error())
	}
	repos := filterRepos(allRepos, repoNames)

	if err := core.CreateWorktrees(cfg.SourceDir, repos, branch, cfg.TargetDir); err != nil {
		die(err.Error())
	}

	branchDir := filepath.Join(cfg.TargetDir, core.BranchToDirName(branch))
	onDisk := core.ListReposOnDisk(branchDir)
	_ = core.CreateCursorWorkspace(branchDir, onDisk)

	if !*noHook {
		_ = core.RunPostCommand(cfg.PostCommand, branchDir)
	}

	jsonOut(map[string]any{"branch": branch, "path": branchDir, "repos": onDisk})
}

func cmdRM(cfg core.Config, args []string) {
	fs := flag.NewFlagSet("rm", flag.ExitOnError)
	force := fs.Bool("f", false, "force remove dirty worktrees")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: wtman rm <branch> [-f]")
		os.Exit(0)
	}
	fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
	}
	branch := fs.Arg(0)

	if err := core.DeleteFeatureBranch(cfg.TargetDir, branch, *force); err != nil {
		die(err.Error())
	}
	jsonOut(map[string]any{"ok": true})
}

func cmdUpdate(cfg core.Config, args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	force := fs.Bool("f", false, "force remove dirty worktrees")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: wtman update <branch> <repo,repo,...> [-f]")
		os.Exit(0)
	}
	fs.Parse(args)

	if fs.NArg() < 2 {
		fs.Usage()
	}
	branch := fs.Arg(0)
	repoNames := strings.Split(fs.Arg(1), ",")

	allRepos, err := core.DiscoverRepos(cfg.SourceDir, cfg.ScanDepth)
	if err != nil {
		die(err.Error())
	}
	repos := filterRepos(allRepos, repoNames)

	if !*force {
		dirty := core.DirtyRemovedWorktrees(repos, branch, cfg.TargetDir)
		if len(dirty) > 0 {
			jsonOut(map[string]any{"error": "dirty", "repos": dirty})
			os.Exit(1)
		}
	}

	if err := core.UpdateFeatureBranch(cfg.SourceDir, repos, branch, cfg.TargetDir, *force); err != nil {
		die(err.Error())
	}

	branchDir := filepath.Join(cfg.TargetDir, core.BranchToDirName(branch))
	onDisk := core.ListReposOnDisk(branchDir)
	_ = core.CreateCursorWorkspace(branchDir, onDisk)

	jsonOut(map[string]any{"ok": true})
}

func cmdMV(cfg core.Config, args []string) {
	fs := flag.NewFlagSet("mv", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: wtman mv <old> <new>")
		os.Exit(0)
	}
	fs.Parse(args)

	if fs.NArg() < 2 {
		fs.Usage()
	}

	if err := core.RenameFeatureBranch(cfg.TargetDir, fs.Arg(0), fs.Arg(1)); err != nil {
		die(err.Error())
	}
	jsonOut(map[string]any{"ok": true})
}

func cmdPull(cfg core.Config, args []string) {
	fs := flag.NewFlagSet("pull", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: wtman pull <branch>")
		os.Exit(0)
	}
	fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
	}

	if err := core.PullFeatureBranch(cfg.TargetDir, fs.Arg(0)); err != nil {
		die(err.Error())
	}
	jsonOut(map[string]any{"ok": true})
}

func jsonOut(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
}

func die(msg string) {
	fmt.Fprintln(os.Stderr, "wtman:", msg)
	os.Exit(1)
}

func filterRepos(all []core.RepoEntry, names []string) []core.RepoEntry {
	idx := make(map[string]core.RepoEntry, len(all))
	for _, r := range all {
		idx[r.Name] = r
	}
	var result []core.RepoEntry
	for _, n := range names {
		n = strings.TrimSpace(n)
		if r, ok := idx[n]; ok {
			result = append(result, r)
		} else {
			die("unknown repo: " + n)
		}
	}
	return result
}
