package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hibobio/wtman/cli"
	"github.com/hibobio/wtman/core"
	"github.com/hibobio/wtman/tui"
)

var cliCommands = map[string]bool{
	"ls": true, "repos": true, "new": true, "rm": true,
	"update": true, "mv": true, "pull": true,
}

func main() {
	cfgPath := flag.String("config", core.DefaultConfigPath(), "path to config file")
	sourceDir := flag.String("source-dir", "", "override source directory")
	shortSource := flag.String("s", "", "override source directory (short)")
	targetDir := flag.String("target-dir", "", "override target directory")
	shortTarget := flag.String("t", "", "override target directory (short)")
	showHelp := flag.Bool("h", false, "show help")
	flag.Parse()

	if *showHelp && flag.NArg() == 0 {
		printUsage()
		os.Exit(0)
	}

	cfg, err := core.LoadConfig(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wtman: failed to load config: %v\n", err)
		os.Exit(1)
	}

	src := coalesce(*sourceDir, *shortSource)
	tgt := coalesce(*targetDir, *shortTarget)
	if src != "" {
		cfg.SourceDir = src
	}
	if tgt != "" {
		cfg.TargetDir = tgt
	}
	if cfg.SourceDir == "" {
		cwd, _ := os.Getwd()
		cfg.SourceDir = cwd
	}
	if cfg.TargetDir == "" {
		cfg.TargetDir = cfg.SourceDir + "/branches"
	}

	args := flag.Args()
	if len(args) > 0 && cliCommands[args[0]] {
		cli.Run(cfg, args)
		return
	}

	app := tui.NewApp(cfg, *cfgPath)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "wtman: %v\n", err)
		os.Exit(1)
	}
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func printUsage() {
	fmt.Fprint(os.Stderr, `wtman - worktree manager

Usage: wtman [flags] [command]

No command launches the interactive TUI.

Commands:
  ls                          List feature branches (JSON)
  repos                       List available source repos (JSON)
  new  <branch> <repos> [-n]  Create branch with worktrees (-n skips post hook)
  rm   <branch> [-f]          Delete branch (-f force even if dirty)
  update <branch> <repos> [-f] Set repos for branch (-f force dirty removal)
  mv   <old> <new>            Rename branch
  pull <branch>               Pull all worktrees in branch

Global flags:
  --config <path>   Config file (default ~/.config/wtman/config.json)
  -s, --source-dir  Source repos directory
  -t, --target-dir  Target branches directory
  -h                Show this help

<repos> is a comma-separated list of repo names.
`)
}
