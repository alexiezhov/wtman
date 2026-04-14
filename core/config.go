package core

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ColorsConfig struct {
	Primary      string `json:"primary,omitempty"`
	Green        string `json:"green,omitempty"`
	Dimmed       string `json:"dimmed,omitempty"`
	Highlight    string `json:"highlight,omitempty"`
	Cyan         string `json:"cyan,omitempty"`
	Red          string `json:"red,omitempty"`
	Separator    string `json:"separator,omitempty"`
	SelectedFg   string `json:"selected_fg,omitempty"`
}

func DefaultColors() ColorsConfig {
	return ColorsConfig{
		Primary:    "63",
		Green:      "42",
		Dimmed:     "241",
		Highlight:  "236",
		Cyan:       "86",
		Red:        "196",
		Separator:  "238",
		SelectedFg: "255",
	}
}

type Config struct {
	SourceDir   string       `json:"source_dir"`
	TargetDir   string       `json:"target_dir"`
	PostCommand string       `json:"post_command"`
	ScanDepth   int          `json:"scan_depth"`
	Colors      ColorsConfig `json:"colors,omitempty"`
}

func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "wtman", "config.json")
}

func DefaultConfig() Config {
	return Config{
		SourceDir:   "",
		TargetDir:   "",
		PostCommand: "tmux split-window -h 'cd {{dir}} && cursor --agent'",
		ScanDepth:   1,
		Colors:      DefaultColors(),
	}
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		path = DefaultConfigPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.ScanDepth < 1 {
		cfg.ScanDepth = 1
	}
	cfg.Colors = mergeColors(cfg.Colors, DefaultColors())
	return cfg, nil
}

func SaveConfig(path string, cfg Config) error {
	if path == "" {
		path = DefaultConfigPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func mergeColors(user, defaults ColorsConfig) ColorsConfig {
	if user.Primary == "" {
		user.Primary = defaults.Primary
	}
	if user.Green == "" {
		user.Green = defaults.Green
	}
	if user.Dimmed == "" {
		user.Dimmed = defaults.Dimmed
	}
	if user.Highlight == "" {
		user.Highlight = defaults.Highlight
	}
	if user.Cyan == "" {
		user.Cyan = defaults.Cyan
	}
	if user.Red == "" {
		user.Red = defaults.Red
	}
	if user.Separator == "" {
		user.Separator = defaults.Separator
	}
	if user.SelectedFg == "" {
		user.SelectedFg = defaults.SelectedFg
	}
	return user
}
