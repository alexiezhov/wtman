package core

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ColorsConfig struct {
	Title      string `json:"title,omitempty"`
	Success    string `json:"success,omitempty"`
	Muted      string `json:"muted,omitempty"`
	SelectedBg string `json:"selected_bg,omitempty"`
	Accent     string `json:"accent,omitempty"`
	Error      string `json:"error,omitempty"`
	Separator  string `json:"separator,omitempty"`
	SelectedFg string `json:"selected_fg,omitempty"`
}

func DefaultColors() ColorsConfig {
	return ColorsConfig{
		Title:      "99",
		Success:    "48",
		Muted:      "245",
		SelectedBg: "237",
		Accent:     "87",
		Error:      "203",
		Separator:  "240",
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
	if user.Title == "" {
		user.Title = defaults.Title
	}
	if user.Success == "" {
		user.Success = defaults.Success
	}
	if user.Muted == "" {
		user.Muted = defaults.Muted
	}
	if user.SelectedBg == "" {
		user.SelectedBg = defaults.SelectedBg
	}
	if user.Accent == "" {
		user.Accent = defaults.Accent
	}
	if user.Error == "" {
		user.Error = defaults.Error
	}
	if user.Separator == "" {
		user.Separator = defaults.Separator
	}
	if user.SelectedFg == "" {
		user.SelectedFg = defaults.SelectedFg
	}
	return user
}
