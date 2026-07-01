package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/alexiezhov/wtman/cli"
	"github.com/alexiezhov/wtman/core"
	"github.com/alexiezhov/wtman/tui"
)

var cliCommands = map[string]bool{
	"ls": true, "repos": true, "new": true, "rm": true,
	"update": true, "mv": true, "pull": true,
}

// Build metadata, injected at release time via -ldflags by GoReleaser.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cfgPath := flag.String("config", core.DefaultConfigPath(), "path to config file")
	sourceDir := flag.String("source-dir", "", "override source directory")
	shortSource := flag.String("s", "", "override source directory (short)")
	targetDir := flag.String("target-dir", "", "override target directory")
	shortTarget := flag.String("t", "", "override target directory (short)")
	showHelp := flag.Bool("h", false, "show help")
	showVersion := flag.Bool("version", false, "print version and exit")
	logLevel := flag.String("log-level", "", "log level: debug, info, warn, error, off")
	verbose := flag.Bool("v", false, "shorthand for --log-level debug")
	flag.Parse()

	if *showVersion || (flag.NArg() > 0 && flag.Arg(0) == "version") {
		v, c, d := buildInfo()
		fmt.Printf("wtman %s (commit %s, built %s)\n", v, c, d)
		os.Exit(0)
	}

	if *showHelp && flag.NArg() == 0 {
		printUsage()
		os.Exit(0)
	}

	cfg, err := core.LoadConfig(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wtman: failed to load config: %v\n", err)
		os.Exit(1)
	}

	levelName := cfg.LogLevel
	if *logLevel != "" {
		levelName = *logLevel
	}
	if *verbose {
		levelName = core.LogLevelDebug
	}
	level, err := core.ParseLogLevel(levelName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wtman: %v\n", err)
		os.Exit(1)
	}
	core.InitLogger(level, os.Stderr)
	slog.Debug("wtman starting",
		"config", *cfgPath,
		"log_level", core.NormalizeLogLevel(levelName),
		"source_dir", cfg.SourceDir,
		"target_dir", cfg.TargetDir,
	)

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

// buildInfo returns the version, commit, and date. GoReleaser injects these via
// -ldflags at release time; for `go install` builds (where the vars keep their
// defaults) it falls back to the VCS metadata Go embeds in the binary.
func buildInfo() (v, c, d string) {
	v, c, d = version, commit, date
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if v == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		v = info.Main.Version
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if c == "none" && s.Value != "" {
				c = s.Value
			}
		case "vcs.time":
			if d == "unknown" && s.Value != "" {
				d = s.Value
			}
		}
	}
	return
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
  version                     Print version and exit

Global flags:
  --config <path>   Config file (default ~/.config/wtman/config.json)
  -s, --source-dir  Source repos directory
  -t, --target-dir  Target branches directory
  --log-level       Log level: debug, info, warn, error, off (default from config)
  -v                Shorthand for --log-level debug
  -h                Show this help

<repos> is a comma-separated list of repo names.
`)
}
