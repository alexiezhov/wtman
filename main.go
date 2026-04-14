package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hibobio/wtman/core"
	"github.com/hibobio/wtman/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfgPath := flag.String("config", core.DefaultConfigPath(), "path to config file")
	sourceDir := flag.String("source-dir", "", "override source directory")
	targetDir := flag.String("target-dir", "", "override target directory")
	flag.Parse()

	cfg, err := core.LoadConfig(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wtman: failed to load config: %v\n", err)
		os.Exit(1)
	}

	if *sourceDir != "" {
		cfg.SourceDir = *sourceDir
	}
	if *targetDir != "" {
		cfg.TargetDir = *targetDir
	}

	if cfg.SourceDir == "" {
		cwd, _ := os.Getwd()
		cfg.SourceDir = cwd
	}
	if cfg.TargetDir == "" {
		cfg.TargetDir = cfg.SourceDir + "/branches"
	}

	app := tui.NewApp(cfg, *cfgPath)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "wtman: %v\n", err)
		os.Exit(1)
	}
}
